#!/usr/bin/env bash
#
# Installs the Pixabros server as a systemd service, from nothing.
#
#   sudo ./deploy/install.sh
#
# It asks for what it cannot know, writes the service user, the environment
# file and the unit, creates the first administrator, and starts the thing.
# Running it again on the same machine is safe: it reuses what is already
# there and only changes what you tell it to.
#
# Nothing here is silent. Every path it writes to and every decision it makes
# is printed, because an installer that hides what it did is impossible to
# undo.

set -euo pipefail

REPO_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)
UNIT_TEMPLATE="$REPO_DIR/deploy/pixabros.service"
UNIT_PATH=/etc/systemd/system/pixabros.service
SERVICE=pixabros

# --- Saying things ----------------------------------------------------------

say()  { printf '\033[1m==>\033[0m %s\n' "$*"; }
warn() { printf '\033[33m !\033[0m %s\n' "$*" >&2; }
die()  { printf '\033[31m X\033[0m %s\n' "$*" >&2; exit 1; }

# ask NAME PROMPT DEFAULT -- reads into NAME, falling back to DEFAULT on enter.
ask() {
  local __name=$1 __prompt=$2 __default=${3-} __reply
  if [ -n "$__default" ]; then
    read -r -p "$__prompt [$__default]: " __reply </dev/tty || true
  else
    read -r -p "$__prompt: " __reply </dev/tty || true
  fi
  printf -v "$__name" '%s' "${__reply:-$__default}"
}

# ask_secret NAME PROMPT -- reads twice without echoing, and insists they match.
ask_secret() {
  local __name=$1 __prompt=$2 __first __second
  while true; do
    read -r -s -p "$__prompt: " __first </dev/tty; echo
    read -r -s -p "$__prompt (again): " __second </dev/tty; echo
    if [ "$__first" != "$__second" ]; then
      warn "They did not match. Try again."
      continue
    fi
    # The server refuses anything shorter, so refusing it here saves finding
    # out after the service is already running.
    if [ "${#__first}" -lt 8 ]; then
      warn "At least 8 characters, please."
      continue
    fi
    if [ "${#__first}" -gt 72 ]; then
      warn "At most 72 characters -- that is bcrypt's limit, not ours."
      continue
    fi
    printf -v "$__name" '%s' "$__first"
    return
  done
}

confirm() {
  local reply
  read -r -p "$1 [y/N]: " reply </dev/tty || true
  [ "$reply" = "y" ] || [ "$reply" = "Y" ]
}

# --- What has to be true before anything is written -------------------------

[ "$(id -u)" -eq 0 ] || die "Run this with sudo: it creates a user and writes to /etc."
[ -f "$UNIT_TEMPLATE" ] || die "Missing $UNIT_TEMPLATE -- run this from a checkout of the repository."

# Checked here rather than being discovered half way through, when some of the
# machine has already been changed.
for tool in systemctl useradd install runuser sed; do
  command -v "$tool" >/dev/null 2>&1 || die "This needs $tool and cannot find it."
done

# The path differs between distributions and a wrong one makes useradd fail.
NOLOGIN=$(command -v nologin || true)
for candidate in /usr/sbin/nologin /sbin/nologin /bin/false; do
  [ -n "$NOLOGIN" ] && break
  [ -x "$candidate" ] && NOLOGIN=$candidate
done
[ -n "$NOLOGIN" ] || NOLOGIN=/bin/false

# The binary carries the admin panel inside it, so a stale one is a stale
# panel. Prefer one that was just built; offer to build if the toolchain is
# here; refuse to guess.
BINARY="$REPO_DIR/pixabros"
if [ ! -x "$BINARY" ]; then
  say "No built binary at $BINARY."
  if command -v make >/dev/null 2>&1 && command -v go >/dev/null 2>&1; then
    if confirm "Build it now with make build?"; then
      ( cd "$REPO_DIR" && make build )
    fi
  fi
fi
[ -x "$BINARY" ] || die "Need a built binary at $BINARY. Run 'make build' first."

# --- What only you know -----------------------------------------------------

echo
say "Pixabros installer"
echo "   Enter accepts the default in brackets."
echo

ask SERVICE_USER "Service user"                  pixabros
ask INSTALL_DIR  "Where the binary goes"         /opt/pixabros
ask DATA_DIR     "Where the data goes"           /var/lib/pixabros
echo
echo "   The address it listens on. Keep it on localhost and put a reverse"
echo "   proxy in front unless this machine is the only thing on the network."
ask LISTEN_ADDR  "Listen address"                127.0.0.1:8080

ENV_FILE=/etc/pixabros/pixabros.env
DB_PATH="$DATA_DIR/pixabros.db"

# The admin panel's session cookie is issued Secure, so a browser will not send
# it back over plain HTTP. Saying so now saves an hour of "why am I logged out
# the moment I log in".
echo
warn "The panel needs HTTPS. Its session cookie is marked Secure, so over"
warn "plain http:// a browser accepts the login and then drops the cookie."
warn "Terminate TLS in front of this service, or you cannot sign in."
echo

FIRST_RUN=true
[ -f "$DB_PATH" ] && FIRST_RUN=false

if [ "$FIRST_RUN" = true ]; then
  ask ADMIN_USER "First administrator's username" admin
  ask_secret ADMIN_PASS "Password for $ADMIN_USER"
  SET_PASSWORD=create
else
  say "A database already exists at $DB_PATH, so this is an upgrade."
  SET_PASSWORD=skip
  if confirm "Reset an administrator's password while we are here?"; then
    ask ADMIN_USER "Which username" admin
    ask_secret ADMIN_PASS "New password for $ADMIN_USER"
    SET_PASSWORD=reset
  fi
fi

# A port below 1024 is privileged, and the unit drops every capability. Binding
# one back is possible and worth being explicit about rather than doing behind
# your back.
PORT=${LISTEN_ADDR##*:}
BIND_LINES=""
case "$PORT" in
  ''|*[!0-9]*) die "The listen address must end in a port, like 127.0.0.1:8080." ;;
esac
if [ "$PORT" -lt 1024 ]; then
  echo
  warn "Port $PORT is privileged. The service otherwise holds no capabilities"
  warn "at all; binding it needs CAP_NET_BIND_SERVICE handed back."
  if confirm "Grant CAP_NET_BIND_SERVICE so it can bind port $PORT?"; then
    BIND_LINES=$'AmbientCapabilities=CAP_NET_BIND_SERVICE\nCapabilityBoundingSet=CAP_NET_BIND_SERVICE'
  else
    die "Then pick a port above 1024 and put a proxy in front of it."
  fi
fi

echo
say "About to write:"
echo "   user          $SERVICE_USER (system account, no login, no home)"
echo "   binary        $INSTALL_DIR/pixabros"
echo "   data          $DATA_DIR  (owned by $SERVICE_USER, mode 0750)"
echo "   environment   $ENV_FILE  (root only, mode 0600)"
echo "   unit          $UNIT_PATH"
echo "   listening on  $LISTEN_ADDR"
echo
confirm "Go ahead?" || die "Nothing was written."

# --- Writing ----------------------------------------------------------------

if id -u "$SERVICE_USER" >/dev/null 2>&1; then
  say "User $SERVICE_USER already exists, leaving it alone."
else
  say "Creating system user $SERVICE_USER"
  useradd --system --no-create-home --home-dir "$DATA_DIR" \
          --shell "$NOLOGIN" "$SERVICE_USER"
fi

say "Installing the binary"
install -d -m 0755 "$INSTALL_DIR"
# To a temporary name and then moved, so a running service is replaced
# atomically rather than being written through underneath itself.
install -m 0755 "$BINARY" "$INSTALL_DIR/pixabros.new"
mv -f "$INSTALL_DIR/pixabros.new" "$INSTALL_DIR/pixabros"

say "Preparing $DATA_DIR"
install -d -m 0750 -o "$SERVICE_USER" -g "$SERVICE_USER" "$DATA_DIR"

say "Writing $ENV_FILE"
install -d -m 0755 "$(dirname "$ENV_FILE")"
umask 077
cat >"$ENV_FILE" <<EOF
# Written by deploy/install.sh. Read by systemd as root, before it drops to
# $SERVICE_USER, which is why this file need not be readable by that user.
PIXABROS_ADDR=$LISTEN_ADDR
PIXABROS_DB_PATH=$DB_PATH
PIXABROS_DATA_DIR=$DATA_DIR
EOF
chown root:root "$ENV_FILE"
chmod 0600 "$ENV_FILE"
umask 022

say "Writing $UNIT_PATH"
sed -e "s|@USER@|$SERVICE_USER|g" \
    -e "s|@INSTALL_DIR@|$INSTALL_DIR|g" \
    -e "s|@DATA_DIR@|$DATA_DIR|g" \
    -e "s|@ENV_FILE@|$ENV_FILE|g" \
    "$UNIT_TEMPLATE" >"$UNIT_PATH"
if [ -n "$BIND_LINES" ]; then
  # Replace the two empty capability lines rather than appending, so the unit
  # never carries both an empty set and a granted one.
  sed -i -e '/^CapabilityBoundingSet=$/d' -e '/^AmbientCapabilities=$/d' "$UNIT_PATH"
  printf '\n# Added by install.sh: this instance binds a privileged port.\n%s\n' \
    "$BIND_LINES" >>"$UNIT_PATH"
fi
chmod 0644 "$UNIT_PATH"

# The database is created by whichever command opens it first. Doing it as the
# service user means the file is owned correctly from the start rather than
# being chowned afterwards.
if [ "$SET_PASSWORD" = create ]; then
  say "Creating administrator $ADMIN_USER"
  runuser -u "$SERVICE_USER" -- env \
    PIXABROS_DB_PATH="$DB_PATH" PIXABROS_DATA_DIR="$DATA_DIR" \
    "$INSTALL_DIR/pixabros" create-admin -username "$ADMIN_USER" -password "$ADMIN_PASS"
elif [ "$SET_PASSWORD" = reset ]; then
  say "Resetting the password for $ADMIN_USER"
  runuser -u "$SERVICE_USER" -- env \
    PIXABROS_DB_PATH="$DB_PATH" PIXABROS_DATA_DIR="$DATA_DIR" \
    "$INSTALL_DIR/pixabros" reset-password -username "$ADMIN_USER" -password "$ADMIN_PASS"
fi
unset ADMIN_PASS

say "Starting the service"
systemctl daemon-reload
systemctl enable "$SERVICE" >/dev/null
systemctl restart "$SERVICE"

# Give it a moment to fail, then say so plainly rather than claiming success.
sleep 2
if ! systemctl is-active --quiet "$SERVICE"; then
  warn "The service did not stay up. The last of its log:"
  journalctl -u "$SERVICE" --no-pager -n 30 || true
  die "Fix the above, then: systemctl restart $SERVICE"
fi

echo
say "Running."
echo "   status    systemctl status $SERVICE"
echo "   logs      journalctl -u $SERVICE -f"
echo "   restart   systemctl restart $SERVICE"
echo
echo "   Serving on $LISTEN_ADDR. The panel is at /I-am-a-pixabro/ once TLS is"
echo "   in front of it."
echo
echo "   One thing left, in the panel under Site settings: fill in the site"
echo "   address. Every canonical link, share card and piece of structured"
echo "   data is built from it, and until it is set they are left out."
echo
