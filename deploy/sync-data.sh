#!/usr/bin/env bash
#
# Copies this machine's data onto an installed server.
#
#   ./deploy/sync-data.sh user@host
#   ./deploy/sync-data.sh user@host /var/lib/pixabros
#   PORT=10987 ./deploy/sync-data.sh user@host
#
# PORT is the ssh port, for a server that does not listen on 22.
#
# Run it from the repository on the machine that has the data, after
# deploy/install.sh has run on the server. It replaces the server's database,
# uploaded media and extracted game builds with this machine's.
#
# Three things it deliberately does not copy:
#
#   rendered-store  every page is re-rendered at startup, and a copied one
#                   would point at asset filenames the server has not built
#   assets          rebuilt from inside the binary on startup
#   admin-dist      a leftover from before the panel was embedded; nothing
#                   reads it
#
# The database is taken as a consistent snapshot rather than copied off disk:
# SQLite is very likely being written to while this runs, and a file copy of a
# live database with a write-ahead log beside it is a corrupt database.

set -euo pipefail

REPO_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)
LOCAL_DATA="$REPO_DIR/data"
SERVICE=pixabros

say()  { printf '\033[1m==>\033[0m %s\n' "$*"; }
warn() { printf '\033[33m !\033[0m %s\n' "$*" >&2; }
die()  { printf '\033[31m X\033[0m %s\n' "$*" >&2; exit 1; }

confirm() {
  local reply
  read -r -p "$1 [y/N]: " reply </dev/tty || true
  [ "$reply" = "y" ] || [ "$reply" = "Y" ]
}

# --- Arguments and preflight ------------------------------------------------

[ $# -ge 1 ] || die "Usage: $0 user@host [remote-data-dir]"
REMOTE=$1
REMOTE_DATA=${2:-/var/lib/pixabros}
STAGE="pixabros-sync"

# One place decides how to reach the server, so ssh and rsync cannot disagree
# about it. rsync needs the whole command rather than a port of its own.
#
# The connection is shared: this makes half a dozen of them, and without a
# master you type your password for every one. The socket lives under a short
# path because a unix socket name has far less room than a file name does.
SSH_PORT=${PORT:-22}
SSH_CONTROL="${TMPDIR:-/tmp}/pixabros-ssh.$$"
SSH_OPTS=(-p "$SSH_PORT" -o ControlMaster=auto -o "ControlPath=$SSH_CONTROL" -o ControlPersist=120)
SSH_CMD=(ssh "${SSH_OPTS[@]}")
RSYNC_SHELL="ssh -p $SSH_PORT -o ControlMaster=auto -o ControlPath=$SSH_CONTROL -o ControlPersist=120"

# Close the shared connection on the way out, however we leave.
close_ssh() {
  [ -S "$SSH_CONTROL" ] && ssh -O exit -o "ControlPath=$SSH_CONTROL" "$REMOTE" 2>/dev/null || true
}

# Old rsync -- the one macOS ships, and openrsync with it -- has no --info at
# all. --progress says less but every version understands it, and a transfer
# that reports nothing is worse than one that reports per file.
if rsync --help 2>&1 | grep -q -- '--info'; then
  PROGRESS=(--info=progress2)
else
  PROGRESS=(--progress)
fi

for tool in ssh rsync sqlite3; do
  command -v "$tool" >/dev/null 2>&1 || die "This needs $tool and cannot find it."
done
[ -d "$LOCAL_DATA" ] || die "No data directory at $LOCAL_DATA."
[ -f "$LOCAL_DATA/pixabros.db" ] || die "No database at $LOCAL_DATA/pixabros.db."

say "Checking the server"
# The service user is read off the installed unit rather than asked for: the
# unit is the truth about what is running there.
REMOTE_USER=$("${SSH_CMD[@]}" "$REMOTE" "systemctl show -p User --value $SERVICE" 2>/dev/null || true)
[ -n "$REMOTE_USER" ] || die "No $SERVICE service on $REMOTE. Run deploy/install.sh there first."

# rsync has to exist at both ends. Finding out at the far end mid-transfer is
# a confusing failure, and this is a fixable one: say what fixes it.
if ! "${SSH_CMD[@]}" "$REMOTE" "command -v rsync" >/dev/null 2>&1; then
  warn "$REMOTE has no rsync. Install it there and run this again:"
  warn "    ssh -p $SSH_PORT $REMOTE 'sudo apt update && sudo apt install -y rsync'"
  die "Nothing was copied."
fi

# Where the staged copy lands. Resolved now, as the login user: the move below
# runs under sudo, where $HOME is root's and the staged files are not.
REMOTE_HOME=$("${SSH_CMD[@]}" "$REMOTE" 'printf %s "$HOME"')
[ -n "$REMOTE_HOME" ] || die "Could not work out the home directory on $REMOTE."
STAGE_DIR="$REMOTE_HOME/$STAGE"

say "Service runs as $REMOTE_USER, data lives in $REMOTE_DATA"

# --- A database that is safe to move ---------------------------------------

SNAPSHOT=$(mktemp "${TMPDIR:-/tmp}/pixabros-db.XXXXXX")
trap 'rm -f "$SNAPSHOT" "$SNAPSHOT-wal" "$SNAPSHOT-shm"; close_ssh' EXIT

say "Taking a consistent snapshot of the database"
# .backup reads through SQLite rather than off the filesystem, so it is safe
# while the site is serving and it folds the write-ahead log in as it goes.
sqlite3 "$LOCAL_DATA/pixabros.db" ".backup '$SNAPSHOT'"

integrity=$(sqlite3 "$SNAPSHOT" "PRAGMA integrity_check;")
[ "$integrity" = "ok" ] || die "The snapshot is not sound: $integrity"

games=$(sqlite3 "$SNAPSHOT" "SELECT count(*) FROM games;")
media=$(sqlite3 "$SNAPSHOT" "SELECT count(*) FROM media;")
posts=$(sqlite3 "$SNAPSHOT" "SELECT count(*) FROM devlog_posts;")
admins=$(sqlite3 "$SNAPSHOT" "SELECT count(*) FROM admins;")

# Reading the snapshot put a write-ahead log beside it. Only the one file is
# sent, so what it holds is folded in first: a database that arrives without
# the log next to it is a database missing whatever the log still held.
sqlite3 "$SNAPSHOT" "PRAGMA wal_checkpoint(TRUNCATE);" >/dev/null
rm -f "$SNAPSHOT-wal" "$SNAPSHOT-shm"

# --- What is about to happen ------------------------------------------------

echo
say "About to replace, on $REMOTE:"
printf '   %-16s %s\n' "database"  "$games games, $media images, $posts posts, $admins admin(s)"
printf '   %-16s %s\n' "media/"    "$(du -sh "$LOCAL_DATA/media" 2>/dev/null | cut -f1)"
printf '   %-16s %s\n' "games/"    "$(du -sh "$LOCAL_DATA/games" 2>/dev/null | cut -f1)"
echo
warn "This mirrors: anything on the server that is not on this machine is"
warn "deleted. The server's current database is kept beside the new one."
warn "The accounts that will work afterwards are this machine's, not the ones"
warn "the installer created."
echo
confirm "Go ahead?" || die "Nothing was copied."

# --- Staging ----------------------------------------------------------------
#
# Copied into the login user's home first, then moved into place under one
# sudo. rsync cannot write into the data directory directly without sudo on
# the far side, and asking sudo to run rsync needs a terminal it does not have.
#
# The move is instant when home and the data directory share a filesystem,
# which on a single-partition server they do.

say "Copying media"
"${SSH_CMD[@]}" "$REMOTE" "mkdir -p '$STAGE_DIR'"
rsync -a --delete "${PROGRESS[@]}" -e "$RSYNC_SHELL" \
  "$LOCAL_DATA/media/" "$REMOTE:$STAGE_DIR/media/"

say "Copying game builds (this is the big one)"
rsync -a --delete "${PROGRESS[@]}" -e "$RSYNC_SHELL" \
  "$LOCAL_DATA/games/" "$REMOTE:$STAGE_DIR/games/"

say "Copying the database snapshot"
rsync -a -e "$RSYNC_SHELL" "$SNAPSHOT" "$REMOTE:$STAGE_DIR/pixabros.db"

# --- Putting it in place ----------------------------------------------------

say "Stopping the service and moving everything into place"
# -t so sudo can ask for a password. One block, so the service is down for as
# little time as possible and either all of it lands or none of it does.
"${SSH_CMD[@]}" -t "$REMOTE" "sudo bash -euo pipefail -s" <<REMOTE_SCRIPT
  stamp=\$(date +%Y%m%d-%H%M%S)

  systemctl stop $SERVICE

  if [ -f "$REMOTE_DATA/pixabros.db" ]; then
    cp -a "$REMOTE_DATA/pixabros.db" "$REMOTE_DATA/pixabros.db.replaced-\$stamp"
    echo "kept the old database as pixabros.db.replaced-\$stamp"
  fi

  rm -rf "$REMOTE_DATA/media" "$REMOTE_DATA/games"
  mv "$STAGE_DIR/media" "$REMOTE_DATA/media"
  mv "$STAGE_DIR/games" "$REMOTE_DATA/games"
  mv "$STAGE_DIR/pixabros.db" "$REMOTE_DATA/pixabros.db"

  # A new database file must never meet the old log: SQLite would replay one
  # into the other and the result would be neither.
  rm -f "$REMOTE_DATA/pixabros.db-wal" "$REMOTE_DATA/pixabros.db-shm"

  # Rendered pages and built assets are derived, and the ones there now were
  # made from the database that has just been replaced.
  rm -rf "$REMOTE_DATA/rendered-store" "$REMOTE_DATA/assets"

  chown -R "$REMOTE_USER:$REMOTE_USER" "$REMOTE_DATA"
  chmod 0750 "$REMOTE_DATA"

  rmdir "$STAGE_DIR" 2>/dev/null || true

  systemctl start $SERVICE
REMOTE_SCRIPT

# --- Did it work ------------------------------------------------------------

say "Checking"
sleep 3
if ! "${SSH_CMD[@]}" "$REMOTE" "systemctl is-active --quiet $SERVICE"; then
  warn "The service did not come back. The last of its log:"
  "${SSH_CMD[@]}" "$REMOTE" "journalctl -u $SERVICE --no-pager -n 30" || true
  die "The old database is still there as pixabros.db.replaced-*"
fi

"${SSH_CMD[@]}" "$REMOTE" "journalctl -u $SERVICE --no-pager -n 5 | sed 's/^/   /'" || true

echo
say "Done."
echo "   Every page was re-rendered on startup, which is what the reconcile"
echo "   line above is reporting."
echo
echo "   Once you are satisfied, the old database can go:"
echo "     ssh -p $SSH_PORT $REMOTE 'sudo rm $REMOTE_DATA/pixabros.db.replaced-*'"
echo
