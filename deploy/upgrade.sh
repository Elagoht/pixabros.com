#!/usr/bin/env bash
#
# Ships a new build to a server that is already running one.
#
#   ./deploy/upgrade.sh user@host
#   PORT=10987 ./deploy/upgrade.sh user@host
#
# It replaces the binary and restarts the service. Nothing else: the database,
# the uploaded media and the game builds are left exactly where they are.
#
# It does not touch the unit file or the environment file. If those changed --
# which they only do when deploy/pixabros.service or install.sh itself changed
# -- run deploy/install.sh again instead; it is safe to re-run and will keep
# the database it finds.
#
# The binary it replaces is kept, and put back if the new one will not start.

set -euo pipefail

REPO_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)
RELEASE="$REPO_DIR/dist/pixabros-linux-amd64/pixabros"
SERVICE=pixabros

say()  { printf '\033[1m==>\033[0m %s\n' "$*"; }
warn() { printf '\033[33m !\033[0m %s\n' "$*" >&2; }
die()  { printf '\033[31m X\033[0m %s\n' "$*" >&2; exit 1; }

confirm() {
  local reply
  read -r -p "$1 [y/N]: " reply </dev/tty || true
  [ "$reply" = "y" ] || [ "$reply" = "Y" ]
}

[ $# -ge 1 ] || die "Usage: $0 user@host"
REMOTE=$1

# One shared connection, so the password is asked for once rather than per step.
SSH_PORT=${PORT:-22}
SSH_CONTROL="${TMPDIR:-/tmp}/pixabros-up.$$"
SSH_CMD=(ssh -p "$SSH_PORT" -o ControlMaster=auto -o "ControlPath=$SSH_CONTROL" -o ControlPersist=120)
close_ssh() {
  [ -S "$SSH_CONTROL" ] && ssh -O exit -o "ControlPath=$SSH_CONTROL" "$REMOTE" 2>/dev/null || true
}
trap close_ssh EXIT

# --- The build --------------------------------------------------------------

if [ ! -x "$RELEASE" ] || confirm "Rebuild before shipping?"; then
  say "Building"
  ( cd "$REPO_DIR" && make release-linux >/dev/null )
fi
[ -x "$RELEASE" ] || die "No build at $RELEASE. Run 'make release-linux'."

# A binary built for this laptop would install cleanly and then never run.
case "$(file -b "$RELEASE" 2>/dev/null)" in
  *"ELF 64-bit"*x86-64*) ;;
  *) die "$RELEASE is not a Linux x86-64 binary. Run 'make release-linux'." ;;
esac

say "Shipping $(du -h "$RELEASE" | cut -f1), built $(date -r "$RELEASE" '+%H:%M')"

# --- Where it is going ------------------------------------------------------

INSTALL_PATH=$("${SSH_CMD[@]}" "$REMOTE" \
  "systemctl show -p ExecStart --value $SERVICE | sed -n 's/.*path=\([^ ;]*\).*/\1/p'" 2>/dev/null || true)
[ -n "$INSTALL_PATH" ] || die "No $SERVICE service on $REMOTE. Run deploy/install.sh there first."
say "Replacing $INSTALL_PATH"

# Resolved as the login user: the swap below runs under sudo, where $HOME is
# root's and the uploaded file is not.
REMOTE_HOME=$("${SSH_CMD[@]}" "$REMOTE" 'printf %s "$HOME"')
[ -n "$REMOTE_HOME" ] || die "Could not work out the home directory on $REMOTE."
UPLOAD="$REMOTE_HOME/pixabros.new"

"${SSH_CMD[@]}" "$REMOTE" "cat > '$UPLOAD'" < "$RELEASE"

# --- Swapping it in ---------------------------------------------------------

# Passed as an encoded argument rather than on standard input, so ssh keeps a
# terminal for sudo to ask for a password on.
REMOTE_SCRIPT=$(cat <<'SCRIPT'
  install -m 0755 __UPLOAD__ __INSTALL__.incoming
  rm -f __UPLOAD__

  # Kept, not overwritten: if the new one will not start, this is what goes
  # back, and a rollback that needs a rebuild is not a rollback.
  if [ -f __INSTALL__ ]; then
    cp -a __INSTALL__ __INSTALL__.previous
  fi

  mv -f __INSTALL__.incoming __INSTALL__
  systemctl restart __SERVICE__
  sleep 2

  if systemctl is-active --quiet __SERVICE__; then
    echo "running the new build"
    exit 0
  fi

  echo "the new build did not start -- putting the old one back" >&2
  if [ -f __INSTALL__.previous ]; then
    mv -f __INSTALL__.previous __INSTALL__
    systemctl restart __SERVICE__
  fi
  exit 1
SCRIPT
)
REMOTE_SCRIPT=${REMOTE_SCRIPT//__UPLOAD__/$UPLOAD}
REMOTE_SCRIPT=${REMOTE_SCRIPT//__INSTALL__/$INSTALL_PATH}
REMOTE_SCRIPT=${REMOTE_SCRIPT//__SERVICE__/$SERVICE}
ENCODED=$(printf '%s' "$REMOTE_SCRIPT" | base64 | tr -d '\n')

say "Installing and restarting"
if ! "${SSH_CMD[@]}" -t "$REMOTE" \
  "printf %s '$ENCODED' | base64 -d | sudo bash -euo pipefail -s"; then
  warn "Rolled back. The last of the log:"
  "${SSH_CMD[@]}" "$REMOTE" "journalctl -u $SERVICE --no-pager -n 30" || true
  die "The old build is running again."
fi

# --- Did it work ------------------------------------------------------------

say "Checking"
"${SSH_CMD[@]}" "$REMOTE" "journalctl -u $SERVICE --no-pager -n 6 | sed 's/^/   /'" || true

echo
say "Done."
echo "   Every page was re-rendered on startup, which is what the reconcile"
echo "   line above reports: the stylesheet's name changes with its contents,"
echo "   so a new build means new pages."
echo
echo "   The build it replaced is still there, if you need it back:"
echo "     ssh -p $SSH_PORT $REMOTE 'sudo mv $INSTALL_PATH.previous $INSTALL_PATH && sudo systemctl restart $SERVICE'"
echo
