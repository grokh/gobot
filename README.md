gobot
=====

TorilMUD Bot for item stat database, account and character tracking, boot tracking, and load reporting.

Item lookups (`stat`, `astat`, `-item`) fall back to `TorilDB.db` when `toril.db` has no match, if a copy sits next to `toril.db` in the bot's working directory. It's opened read-only; if it's missing or unreadable, the bot answers from `toril.db` alone. Replace it with a rename (download to a temp file, then `mv`) so a lookup never reads a half-copied file.

`gobot -bak` saves a gzipped `.dump` of `toril.db` in `bak/` and keeps 60 days of dailies plus the first backup of each month; restore one with `zcat bak/toril.db.<date>.gz | sqlite3 toril.db`. `dbcopy.sh` publishes the scrubbed copy, and `publish.sh` updates the TorilEQ site.

Currently running on:

* Debian 13
* Tintin++ 2.02.05
* Go 1.27.2
* SQLite 3.46.1
