#!/bin/sh
echo '.dump' | sqlite3 toril.db | gzip -c >toril.db.`date +"%Y-%m-%d"`.gz
#zcat toril.db.`date +"%Y-%m-%d"`.gz | sqlite3 toril.db
