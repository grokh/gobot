#!/bin/bash

cp /srv/bot/toril.db /tmp/toril.db
sqlite3 /tmp/toril.db "DELETE FROM chars WHERE vis = 'f';"
cp /tmp/toril.db /srv/web/grokh.com/html/toril.db
cp /tmp/toril.db /home/trance/go/src/github.com/grokh/gobot/toril.db
cp /tmp/toril.db /home/trance/mud/torilmud/toril.db
cp /tmp/toril.db /srv/web/torileq/toril.db
