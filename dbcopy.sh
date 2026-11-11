#!/bin/bash
# Publish a copy of the bot's toril.db that holds only what's meant to be
# public. The copy starts as a new, empty database and gets what
# dbcopy_public.sql lists, read from the bot's database, which is only ever
# opened read-only. Hidden alts, real names and the hourly counts in online
# aren't on that list, so they're never written to the copy, and nor is
# anything added to the bot's database later, until it's listed.
#
# The bot can be writing meanwhile: the read is one transaction, which waits
# up to 10 seconds for a write to finish. Each copy is renamed into place,
# so downloads never see a half-written file. The TorilEQ site doesn't need
# one: it reads item stats from TorilDB.db.
#
# The copy's items and item_flags are filled from the TorilDB.db beside the
# source, for NyyLIB, the Mudlet package (see dbcopy_items.sql). If that's
# missing or the fill fails, they get the item tables the bot's database
# used to keep, if it still has them, and the reason goes to stderr.
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
sql=$(dirname "$0")

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
db="$tmp/toril.db"

# Both databases are opened read-only, so they're never created or changed
# here. -bail stops at the first error, which rolls back what that step had
# done: without it, sqlite3 would go on and commit what was left.
bot="ATTACH 'file:$src?mode=ro' AS bot"
sqlite3 -bail "$db" ".timeout 10000" "$bot" ".read '$sql/dbcopy_public.sql'"

filled=
if [ ! -f "$itemdb" ]; then
	echo "dbcopy.sh: no $itemdb, so the item tables aren't refreshed" >&2
elif sqlite3 -bail "$db" "ATTACH 'file:$itemdb?mode=ro' AS k" \
	".read '$sql/dbcopy_items.sql'"; then
	filled=yes
else
	echo "dbcopy.sh: the item tables weren't refreshed from $itemdb" >&2
fi
# Without Kegor's items, the copy gets the ones the bot's database used to
# keep, if it still has them: the two tables NyyLIB reads, and none of the
# details that went with them.
cols="item_id, item_name, keywords, weight, c_value, item_type, from_zone,
	from_mob, short_stats, long_stats, full_stats, comments, last_id"
if [ -z "$filled" ] && [ "$(sqlite3 "$db" "$bot" "SELECT count(*)
	FROM bot.sqlite_master WHERE type = 'table'
	AND name IN ('items', 'item_flags')")" = 2 ]; then
	sqlite3 -bail "$db" ".timeout 10000" "$bot" "BEGIN;
		INSERT INTO main.items ($cols)
		SELECT $cols FROM bot.items ORDER BY item_id;
		INSERT INTO main.item_flags (item_id, flag_abbr)
		SELECT item_id, flag_abbr FROM bot.item_flags ORDER BY rowid;
		COMMIT;"
fi
# VACUUM is for the download's size: the indexes fill unevenly as the rows
# go in, and it packs them.
sqlite3 "$db" "VACUUM;"
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
