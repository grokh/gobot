#!/bin/bash
# Update the TorilEQ site: the gobot binary, the page files, and the bot's
# copy of TorilDB.db, which the site reads item stats from (torildb.sh keeps
# both copies current from then on). It also refreshes the scrubbed toril.db
# downloads with dbcopy.sh. The site is started again even if a step fails.
set -euo pipefail

repo=~/go/src/github.com/grokh/gobot
site=/srv/web/torileq

sudo systemctl stop torileq
trap 'sudo systemctl start torileq' EXIT
"$repo/dbcopy.sh"
install -m 0644 /srv/bot/TorilDB.db "$site/TorilDB.db.new"
mv -f "$site/TorilDB.db.new" "$site/TorilDB.db"
cp ~/go/bin/gobot "$site/"
rsync -azv "$repo/html/" "$site/html/"
