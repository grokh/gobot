#!/bin/bash
sudo systemctl stop torileq
cp /srv/bot/toril.db /srv/web/torileq/toril.db
cp ~/go/bin/gobot /srv/web/torileq/
rsync -azv ~/go/src/github.com/grokh/gobot/html/ /srv/web/torileq/html/
sudo systemctl start torileq
