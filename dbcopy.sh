#!/bin/bash
# Publish a scrubbed copy of the bot's toril.db: no hidden alts and no real
# names. The snapshot is taken with VACUUM INTO, which is safe while the bot
# is writing; the scrub is followed by a VACUUM, because deleted rows stay
# readable in the file until one runs; and each copy is renamed into place,
# so the website and downloads never see a half-written file.
#
# Usage: dbcopy.sh [source [destination...]]   (defaults below)
set -euo pipefail

src=${1:-/srv/bot/toril.db}
[ $# -gt 0 ] && shift
[ $# -gt 0 ] || set -- \
	/srv/web/grokh.com/html/toril.db \
	/home/trance/go/src/github.com/grokh/gobot/toril.db \
	/home/trance/mud/torilmud/toril.db \
	/srv/web/torileq/toril.db

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
db="$tmp/toril.db"

sqlite3 "$src" ".timeout 10000" "VACUUM INTO '$db'"
sqlite3 "$db" "DELETE FROM chars WHERE vis = 'f';
	UPDATE accounts SET player_name = NULL;
	VACUUM;"
[ "$(sqlite3 "$db" "PRAGMA integrity_check;")" = ok ]

for dest in "$@"; do
	# copy next to the old file and rename over it; where the folder can't
	# take a new file, copy over the old one in place as before
	if install -m 0644 "$db" "$dest.new" 2>/dev/null; then
		mv -f "$dest.new" "$dest"
	else
		cp "$db" "$dest"
	fi
done
