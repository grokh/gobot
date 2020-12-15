#!/bin/bash

cp /srv/bot/toril.db /tmp/
sqlite3 /tmp/toril.db "delete from chars where vis = 'f';"
cp /tmp/toril.db /srv/web/grokh.com/html/
