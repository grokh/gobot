#!/bin/bash
# Fetch Kegor's TorilDB.db, which the bot, the TorilEQ site and the tintin
# setup in ~/mud/torilmud read item stats from, and put it in place once
# gobot has checked it can use it: a
# whole database, with every column gobot reads, and not much smaller than
# the copy in use. Each copy is renamed into place, so nothing ever reads a
# half-written file, and the copy it replaces is kept as TorilDB.db.prev.
# A download that fails the check changes nothing.
#
# Kegor's download link isn't in the repo, since it's his to share: put it
# on the first line of torildb.url in the bot's folder, next to tokens.tin.
#
# Run it daily from cron, e.g.:
#     17 6 * * * $HOME/go/src/github.com/grokh/gobot/torildb.sh
#
# Usage: torildb.sh [destination...]   (defaults below)
# TORILDB_URL, GOBOT and BOT_DIR override the defaults below too.
set -euo pipefail

gobot=${GOBOT:-$HOME/go/bin/gobot}
# the check runs here, so it compares against the bot's copy and logs its
# result to the bot's logs/bot.log; failures also go to stderr, so cron
# mails them
bot=${BOT_DIR:-/srv/bot}

url=${TORILDB_URL:-}
if [ -z "$url" ] && [ -f "$bot/torildb.url" ]; then
	url=$(head -n 1 "$bot/torildb.url" | tr -d '[:space:]')
fi
if [ -z "$url" ]; then
	echo "torildb.sh: no download link: put Kegor's TorilDB.db link in" \
		"$bot/torildb.url" >&2
	exit 1
fi

# the bot, the site, and ~/mud/torilmud, whose tintin setup answers stat on
# the ASSOC channel and in tells by running gobot in that folder
[ $# -gt 0 ] || set -- \
	/srv/bot/TorilDB.db \
	/srv/web/torileq/TorilDB.db \
	/home/trance/mud/torilmud/TorilDB.db

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
new="$tmp/TorilDB.db"

curl -fsS --retry 3 --max-time 600 -o "$new" "$url"
(cd "$bot" && "$gobot" -checkitems "$new" >/dev/null)

for dest in "$@"; do
	# unchanged since the last run: keep the .prev it already has
	if cmp -s "$new" "$dest"; then
		continue
	fi
	if [ -f "$dest" ]; then
		cp -p "$dest" "$dest.prev"
	fi
	install -m 0644 "$new" "$dest.new"
	mv -f "$dest.new" "$dest"
done
