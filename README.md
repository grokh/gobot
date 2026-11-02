gobot
=====

The TorilMUD helper bot Katumi and the [TorilEQ](https://www.grokh.com/torileq/) item search. They use two SQLite databases: Kegor's `TorilDB.db` for item stats, read-only, and the bot's own `toril.db` for characters and accounts, boots, and load reports.

## How it works

`tt++ katumi.tin`, run in the bot folder (`/srv/bot`), logs in to torilmud.com and calls `gobot` for each event: `-time` and `-who` every 30 seconds to track boots and who's online, `-char <name> -tell "<text>"` for each tell, and `-bak` daily. MUD text is escaped for the shell by `shellsafe`, and replies go back to the MUD as commands, so keep tintin++ syntax out of them (`tintinSafe` in `tell.go`, which every line from `TorilDB.db` goes through).

Tells: `stat`, `astat`, `fstat`, `who`, `clist`, `char`, `find`, `class`, `name`, `addname`, `delalt`, `addalt`, `hidden`, `lr`, `lrdel`; `tell katumi help <cmd>` explains each. `stat` and `astat` answer with Kegor's own short and long stats; `fstat` and the website search his `eq` table's stat columns, mapped from the names players use in `items.go`.

The website is `gobot -port 8080` (`torileq.service`, behind nginx), reading the `TorilDB.db` in its own folder. Other flags: `-item`, `-glist` (items missing from `TorilDB.db` or last identified before 2020-11-01), and `-checkitems <file>`. Logs: `logs/bot.log`.

## TorilDB.db

`torildb.sh` downloads Kegor's `TorilDB.db` and puts it in the bot's folder and the website's. His download link stays out of the repo: put it on the first line of `/srv/bot/torildb.url`. Run the script daily from cron:

    17 6 * * * $HOME/go/src/github.com/grokh/gobot/torildb.sh

It only replaces a copy once `gobot -checkitems` passes the download: the file is whole, has every `eq` column gobot reads, and holds at least 90% as many items as the copy in use. The copy it replaces is kept as `TorilDB.db.prev`. If the check fails, nothing changes, and the reason goes to stderr (mailed by cron) and `logs/bot.log`. If `TorilDB.db` goes missing or can't be read, `stat`, `astat` and `fstat` say item stats are unavailable, and the website answers 503.

New item stats go to Kegor. The item tables still in old `toril.db` files aren't read any more.

## Build and run

You need Go 1.21+ and gcc, since `mattn/go-sqlite3` compiles SQLite into gobot.

    go vet ./... && go test ./...   # needs the sqlite3 CLI; -email also sends a test email
    go install                      # the bot uses it from its next event
    ./publish.sh                    # the website

`publish.sh` stops the site, copies in the bot's `TorilDB.db`, the binary and `html/`, and always restarts it. It also runs `dbcopy.sh`, which publishes `toril.db` without hidden alts or real names to the download and two local folders.

`gobot -bak` writes `bak/toril.db.<date>.gz`, keeps 60 days plus the first of each month, and fails rather than save a partial dump. To restore, move `toril.db` aside and run `gobot -res <file>`.

`sqlite3 toril.db ".read init_db.sql"` builds a new database. Not in the repo: `toril.db`, `TorilDB.db`, `torildb.url` (Kegor's download link), `tokens.tin` (the bot's `acct`, `char`, `cpw`), and `tokens.txt` (reboot email sender, password, recipients). `fix_times_and_stats.sql` converted old times to UTC, and `conv.sql` is the 2013 schema conversion.

Currently running on Debian 13, Tintin++ 2.02.05, and Go 1.27.2, with SQLite 3.53.4 built into gobot (mattn/go-sqlite3 v1.14.52) and the sqlite3 3.46.1 CLI for backups and `dbcopy.sh`.
