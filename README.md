gobot
=====

TorilMUD Bot for item stat database, account and character tracking, boot tracking, and load reporting.

Item lookups (`stat`, `astat`, `-item`) fall back to `TorilDB.db` when `toril.db` has no match, if a copy sits next to `toril.db` in the bot's working directory. It's opened read-only; if it's missing or unreadable, the bot answers from `toril.db` alone. Replace it with a rename (download to a temp file, then `mv`) so a lookup never reads a half-copied file.

Currently running on:

* Debian 13
* Tintin++ 2.02.05
* Go 1.27.2
* SQLite 3.46.1
