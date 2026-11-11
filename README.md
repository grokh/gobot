gobot
=====

The TorilMUD helper bot Katumi and the [TorilEQ](https://www.grokh.com/torileq/) item search. They use two SQLite databases: Kegor's `TorilDB.db` for item and mob stats, read-only, and the bot's own `toril.db` for characters and accounts, boots, and load reports.

## How it works

`tt++ katumi.tin`, run in the bot folder (`/srv/bot`), logs in to torilmud.com and calls `gobot` for each event: `-time`, `-who` and `-online <n>` every 30 seconds to track boots, who's online and how many, `-char <name> -tell "<text>"` for each tell, and `-bak` daily. MUD text is escaped for the shell by `shellsafe`, and replies go back to the MUD as commands, so keep tintin++ syntax out of them (`tellSafe` in `tell.go`, which every reply line goes through; `tintinSafe` also drops `@` from `TorilDB.db` text).

`-online` takes the count from who's "Total players online" line and adds it to its UTC hour in toril.db's `online` table (`online.go`): the who checks that hour, their total, and the fewest and most at once, so an hour's average is `total / checks`. gobot makes the table in a toril.db that predates it. Its history from May 2013 on came from Katumi's session logs, timed by the MUD's own clock read as US Eastern time; that clock has run up to about 40 minutes slow, so those hours can be off by that much. The table is only in the bot's own `toril.db`: `dbcopy.sh` leaves it out of the copies it publishes.

When `-time` sees a new boot, gobot sends a reboot alert to the [ntfy](https://ntfy.sh) topic whose URL is on the first line of `ntfy.url` in the bot's folder, ex. `https://ntfy.sh/<topic>`. Subscribe to that topic in the ntfy app to get the alerts. Anyone who knows the topic can read them and send their own, so give it a long random name. To check the alerts without waiting for a reboot, run `gobot -testalert` in the bot's folder: it sends a test alert the same way, and if that fails, says why on stderr and exits 1.

Tells: `stat`, `astat`, `fstat`, `mstat`, `who`, `clist`, `char`, `find`, `class`, `name`, `addname`, `delalt`, `addalt`, `hidden`, `lr`, `lrdel`; `tell katumi help <cmd>` explains each. `stat` and `astat` answer with Kegor's own short and long stats; `fstat` and the website search his `eq` table by stat, resist, slot, effect, flag, where an item comes from, who can use it, type and zone, mapped from the names players use in `items.go`, and `fstat` by weapon type too. `fstat` lists 10 items, the most of the first stat compared first, then says how many more there are, with a TorilEQ link to all of them. The link spells its operators as words (`?fstat=dam+gt+2,+slot+neck`), so it needs no escaping in tintin or the MUD. `mstat` answers from his `mobs` table (`mobs.go`): his short stats for the mob, the eq he lists it carrying, and its quest's items. Mobs that share a name, which he numbers (the bartender 1 to 20), come back together, up to 10. `astat` adds a line naming up to 5 mobs that carry the item. A tell Katumi can't make out gets a guess at what was meant (`guess.go`):
- a mistyped command gets the command it's a typo of (`stats` is `stat`)
- a command with nothing to look up gets that command's help
- a `stat`, `astat` or `mstat` that finds nothing gets up to 3 names that match once each word in no name is read as a typo of the closest word that is

Guesses are only ever offered, never run, since some commands change what Katumi keeps. A reply sends the MUD at most 3,000 bytes (`maxReplyBytes` in `tell.go`), because the MUD disconnects a character that sends it more than about 4 KB at once. A longer one, like a `clist` of an account with dozens of alts, ends with a count of the tells left out.

The website is `gobot -port 8080` (`torileq.service`, behind nginx, which has to proxy to `127.0.0.1:8080`: gobot listens there only), reading the `TorilDB.db` in its own folder. Its searches work by GET as well as by the form's POST, so each results page links to its own search, and `?fstat=<search>` runs an fstat search there, which is where `fstat`'s links go. Results can be ordered by either attribute. Other flags: `-item`, `-glist` (items missing from `TorilDB.db` or last identified before 2020-11-01), and `-checkitems <file>`. Logs: `logs/bot.log`.

## TorilDB.db

`torildb.sh` downloads Kegor's `TorilDB.db` and puts it in the bot's folder, the website's, and `/home/trance/mud/torilmud`, where another character's tintin setup also answers `stat` with gobot. His download link stays out of the repo: put it on the first line of `/srv/bot/torildb.url`. Run the script daily from cron:

    17 6 * * * $HOME/go/src/github.com/grokh/gobot/torildb.sh

It only replaces a copy once `gobot -checkitems` passes the download: the file is whole, has every `eq` and `mobs` column gobot reads, and holds at least 90% as many items as the copy in use. The copy it replaces is kept as `TorilDB.db.prev`. If the check fails, nothing changes, and the reason goes to stderr (mailed by cron) and `logs/bot.log`. If `TorilDB.db` goes missing or can't be read, `stat`, `astat` and `fstat` say item stats are unavailable, `mstat` that mob stats are, and the website answers 503. If only the `mobs` table can't be read, `astat` leaves out its line of mobs.

New item stats go to Kegor by mwrite, ideally with the zone, the load type, and any quest that uses the item. His load types, in `eq`'s `load` column, are R rare, Q quest reward, S store, I invasion, ! out of game, C common (loads on more than one mob or zone), K key, X no take, and N for most others; C, K and X mostly keep junk out of his zone stats. gobot doesn't read the item tables in old `toril.db` files any more, but the published copy's are filled from `TorilDB.db` (see `dbcopy.sh` below).

## Build and run

You need Go 1.21+ and gcc, since `mattn/go-sqlite3` compiles SQLite into gobot.

    go vet ./... && go test ./...   # needs the sqlite3 CLI
    go install                      # the bot uses it from its next event
    ./publish.sh                    # the website

If tt++ 2.02.20 or later is installed, `go test` also runs `katumi.tin` in it (`Test_Tintin`). Fake tells go through the real tell trigger, `shellsafe` and replies to a gobot built from the checkout, and some try to get a shell command, a tintin command or `$cpw` through. Then the script logs in to a stand-in MUD on 127.0.0.1 that is down when tintin starts, comes up, then drops it. It has to keep trying while the MUD is down, wait out its relogin delay (cut from 30 seconds to half a second) after the drop, and send its account, password and character each time, once asked. It never reaches torilmud.com: for the tells, tintin reads a copy of the script without `#session` lines or login events, and for the logins, a copy whose only `#session` line is for the stand-in. `TORILMUD_TIN=/home/trance/mud/torilmud/torilmud.tin go test ./...` checks that script too, logging in with `play ynndchiarhlizz;login` as the server's `basic.tin` does. On Windows it runs WinTin++ with Git for Windows' `sh`, and `go test -short` skips it.

[NyyLIB](https://github.com/Nyyrazzilyss/NyyLIB), the Mudlet package many players use, reads two kinds of line from Katumi with triggers: clist and char replies fill its who list, and the NHC welcome makes it look the newcomer up. `Test_NyyLIB` runs NyyLIB's own trigger patterns against gobot's replies, so a wording change that would break them fails. Tell Nyyrazzilyss before changing those lines on purpose.

GitHub runs the same checks on every pull request and every push to main (`.github/workflows/ci.yml`): `gofmt`, `go vet` and `go test`, on the Go and tintin++ versions the server runs. Those versions are set at the top of the workflow, so update them along with the list at the end of this file. The run fails if `Test_Tintin` skips. It can't check `torilmud.tin`, which is in a private repo.

`publish.sh` stops the site, copies in the bot's `TorilDB.db`, the binary and `html/`, and always restarts it. It also runs `dbcopy.sh`, which publishes a copy of `toril.db` to the download and two local folders. The copy starts as an empty database and gets only what `dbcopy_public.sql` lists: visible characters, accounts by name, boots, and the races and classes. Hidden alts, real names, load reports and the `online` table aren't on the list, so they're never written to the copy, and a table or column added to the bot's `toril.db` later stays out until it's listed there. The bot's `toril.db` is opened read-only and read in one transaction, so the bot can keep writing meanwhile, and the copy is an ordinary rollback-journal database whether or not the bot's is in WAL mode. NyyLIB downloads the copy with `@updatedb` and reads item stats from it, so `dbcopy.sh` also fills its `items` and `item_flags` from the bot's `TorilDB.db` with `dbcopy_items.sql`, in the shape NyyLIB reads. A potion, scroll or poison row NyyLIB couldn't read would stop it starting, so the fill fails rather than publish one. If it fails, or `TorilDB.db` is missing, the copy gets the old `items` and `item_flags` from the bot's `toril.db`, if it still has them, and the reason goes to stderr.

`gobot -bak` writes `bak/toril.db.<date>.gz`, keeps 60 days plus the first of each month, and fails rather than save a partial dump. To restore, move `toril.db` aside and run `gobot -res <file>`.

`sqlite3 toril.db ".read init_db.sql"` builds a new database. Not in the repo: `toril.db`, `TorilDB.db`, `torildb.url` (Kegor's download link), `tokens.tin` (the bot's `acct`, `char`, `cpw`), and `ntfy.url` (the ntfy topic for reboot alerts). `fix_times_and_stats.sql` converted old times to UTC, and `conv.sql` is the 2013 schema conversion.

Currently running on Debian 13, Tintin++ 2.02.61, and Go 1.27.2, with SQLite 3.54.1 built into gobot (mattn/go-sqlite3 v1.14.53) and the sqlite3 3.46.1 CLI for backups and `dbcopy.sh`.
