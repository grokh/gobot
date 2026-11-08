#!/bin/bash
# Publish a scrubbed copy of the bot's toril.db: no hidden alts and no real
# names. The snapshot is taken with VACUUM INTO, which is safe while the bot
# is writing; the scrub is followed by a VACUUM, because deleted rows stay
# readable in the file until one runs; and each copy is renamed into place,
# so downloads never see a half-written file. The TorilEQ site doesn't need
# one: it reads item stats from TorilDB.db.
#
# The copy's items and item_flags are refilled from the TorilDB.db beside
# the source, for NyyLIB, the Mudlet package (see dbcopy_items.sql). If
# that's missing or the refill fails, the copy keeps the item tables it
# has, and the reason goes to stderr.
#
# Usage: dbcopy.sh [source [destination...]]   (defaults below)
set -euo pipefail

src=${1:-/srv/bot/toril.db}
[ $# -gt 0 ] && shift
[ $# -gt 0 ] || set -- \
	/srv/web/grokh.com/html/toril.db \
	/home/trance/go/src/github.com/grokh/gobot/toril.db \
	/home/trance/mud/torilmud/toril.db
itemdb=$(dirname "$src")/TorilDB.db

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
db="$tmp/toril.db"

sqlite3 "$src" ".timeout 10000" "VACUUM INTO '$db'"
# TorilDB.db is opened read-only, so it's never created or changed here.
# -bail stops at the first error, which rolls the refill back: without it,
# sqlite3 would go on and commit what was left.
if [ ! -f "$itemdb" ]; then
	echo "dbcopy.sh: no $itemdb, so the item tables aren't refreshed" >&2
elif ! sqlite3 -bail "$db" "ATTACH 'file:$itemdb?mode=ro' AS k" \
	".read '$(dirname "$0")/dbcopy_items.sql'"; then
	echo "dbcopy.sh: the item tables weren't refreshed from $itemdb" >&2
fi
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
