#!/bin/bash
# Update the TorilEQ site: the gobot binary and the page files. The site is
# started again even if a step fails. Cron keeps the rest current: torildb.sh
# the site's TorilDB.db, and dbcopy.sh the toril.db downloads. A new site
# has no TorilDB.db until torildb.sh has run once.
set -euo pipefail

repo=~/go/src/github.com/grokh/gobot
site=/srv/web/torileq

sudo systemctl stop torileq
trap 'sudo systemctl start torileq' EXIT
cp ~/go/bin/gobot "$site/"
rsync -azv "$repo/html/" "$site/html/"
