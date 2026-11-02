#!/bin/bash
# Update the TorilEQ site: a fresh scrubbed toril.db from dbcopy.sh (the site
# only needs the item tables), the gobot binary, and the page files. The
# site is started again even if a step fails.
set -euo pipefail

repo=~/go/src/github.com/grokh/gobot

sudo systemctl stop torileq
trap 'sudo systemctl start torileq' EXIT
"$repo/dbcopy.sh"
cp ~/go/bin/gobot /srv/web/torileq/
rsync -azv "$repo/html/" /srv/web/torileq/html/
