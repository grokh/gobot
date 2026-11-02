gobot
=====

The TorilMUD helper bot Katumi and the [TorilEQ](https://www.grokh.com/torileq/) item search, both built on one SQLite database, `toril.db`: items and their stats, characters and accounts, boots, and load reports.

## How it works

`tt++ katumi.tin`, run in the bot folder (`/srv/bot`), logs in to torilmud.com and calls `gobot` for each event: `-time` and `-who` every 30 seconds to track boots and who's online, `-char <name> -tell "<text>"` for each tell, and `-bak` daily. MUD text is escaped for the shell by `shellsafe`, and replies go back to the MUD as commands, so keep tintin++ syntax out of them (`tintinSafe` in `tell.go`).

Tells: `stat`, `astat`, `fstat`, `who`, `clist`, `char`, `find`, `class`, `name`, `addname`, `delalt`, `addalt`, `hidden`, `lr`, `lrdel`; `tell katumi help <cmd>` explains each. When `stat`, `astat`, or `-item` finds nothing in `toril.db`, it tries `TorilDB.db` beside it, read-only, if present; update it by renaming a finished download.

The website is `gobot -port 8080` (`torileq.service`, behind nginx) on a scrubbed copy of `toril.db`. Other flags: `-item`, `-glist` (items missing or last identified before 2020-11-01), and `-import <file>` for identify output (SQL logged to `import.sql`), then `-s` to rebuild stat lines. Logs: `logs/bot.log`.

## Build and run

You need Go 1.21+ and gcc, since `mattn/go-sqlite3` compiles SQLite into gobot.

    go vet ./... && go test ./...   # needs the sqlite3 CLI; -email also sends a test email
    go install                      # the bot uses it from its next event
    ./publish.sh                    # the website

`publish.sh` stops the site, runs `dbcopy.sh` (which publishes `toril.db` without hidden alts or real names to the site, the download, and two local folders), copies in the binary and `html/`, and always restarts it.

`gobot -bak` writes `bak/toril.db.<date>.gz`, keeps 60 days plus the first of each month, and fails rather than save a partial dump. To restore, move `toril.db` aside and run `gobot -res <file>`.

`sqlite3 toril.db ".read init_db.sql"` builds a new database. Not in the repo: `toril.db`, `TorilDB.db`, `tokens.tin` (the bot's `acct`, `char`, `cpw`), and `tokens.txt` (reboot email sender, password, recipients). `fix_times_and_stats.sql` converted old times to UTC, and `conv.sql` is the 2013 schema conversion.

Currently running on Debian 13, Tintin++ 2.02.05, and Go 1.27.2, with SQLite 3.53.4 built into gobot (mattn/go-sqlite3 v1.14.52) and the sqlite3 3.46.1 CLI for backups and `dbcopy.sh`.
