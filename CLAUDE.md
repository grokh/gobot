# gobot

Katumi, a helper bot for TorilMUD, and the TorilEQ item search at grokh.com/torileq/. It is one Go program (`package main`, no subpackages) that tintin++ runs once for each event. README.md describes all of it: read it before changing how anything behaves, and update it in the same change.

## Checks

    gofmt -l .                       # must print nothing
    go vet ./... && go test ./...

- They need gcc, since go-sqlite3 compiles SQLite, and the `sqlite3` CLI. The first run takes about a minute.
- `Test_Tintin` runs `katumi.tin` in tt++ 2.02.20 or later, and skips without one. `go test -short` leaves it out.
- CI (`.github/workflows/ci.yml`) runs the same checks on every pull request, and fails if `Test_Tintin` skips.
- `staticcheck ./...` and `shellcheck *.sh` print nothing either. CI doesn't run them, so run them where they are installed and keep it that way.
- `ChkErr` and `fatal` panic with a `fatalError`. `main` ends the run on one as `log.Fatal` would, logging it under the run's flags (`endRun`), `ReplyTo` answers the tell with `recordsDown` instead, and the site answers 500. Call `fatal`, not `log.Fatal`, which ends the program at once, without the run's name: in `Test_All`, whose log goes to `logs/test.log`, one that fails would stop the whole test binary with nothing on the console but `FAIL`.

## Where things are

| File | Holds |
|---|---|
| `gobot.go` | the flags, `main`, how a run that ends early is logged (`endRun`), and the log's clean-up of what players send |
| `tell.go` | every tell command but `fstat`, the help for all of them, `tellSafe`, `tintinSafe`, `maxReplyBytes` |
| `fstat.go` | `fstat`: how it reads a search, for tells and the site's `?fstat=` links, and its guesses at a field it can't read |
| `items.go`, `mobs.go` | lookups in Kegor's `TorilDB.db` (its `eq` and `mobs` tables) |
| `guess.go` | what Katumi suggests for a mistyped tell |
| `who.go`, `time.go`, `online.go` | who lists, boots, hourly player counts |
| `web.go`, `html/` | the TorilEQ site (`gobot -port`) |
| `paste.go` | reading item names out of a list pasted into the site, and `-glist`'s |
| `db.go` | opening, backing up and restoring `toril.db` |
| `local.go` | `-glist` |
| `ntfy.go` | reboot alerts, and the alert when Katumi stops logging in |
| `katumi.tin` | the tintin script that logs Katumi in and calls gobot |
| `init_db.sql` | the schema of the bot's `toril.db`, which the tests build from |
| `dbcopy.sh`, `dbcopy_public.sql`, `dbcopy_items.sql` | the published copy of `toril.db` |
| `torildb.sh`, `publish.sh`, `torileq.service` | what the server runs |

## Rules

- **Nothing here may connect to torilmud.com.** `katumi.tin` logs in as soon as tintin starts, so never run `tt++ katumi.tin` by hand. The tests read copies with the `#session` lines and login events taken out, and refuse to run if one is left. Keep that guard.
- **Replies are run by tintin as commands.** Every reply line goes through `tellSafe`, and text from `TorilDB.db` through `tintinSafe` as well. A reply sends the MUD at most `maxReplyBytes`.
- **A tell command is one entry in `tellCommands`**, which `ReplyTo` answers from and the typo guesses pick from. Give a new one a `help` entry and a place in the `?` menu (`menu`): `testHelp` fails until both are there.
- **tintin fires only the first `#act` that matches a line.** Actions are tried by priority, then in ASCII order of their patterns, so a trigger with no anchor takes any tell that quotes its words. The tell trigger runs at priority 4 for that reason. A trigger that does something with what it matches needs an anchor, or should act only while it's needed, as the bad-character `#zap` does.
- **tintin fires a time event, like `#event {DAY}`, in every session that has it,** and the MUD's session starts with a copy of the startup session's events. `katumi.tin` drops the copy of its midnight `-bak` in `SESSION CREATED`, so the backup runs once, from the startup session. A new time event needs the same, or it runs twice. `reconnect` checks the backup.
- **MUD text reaches gobot through a shell.** `shellsafe` in `katumi.tin` cleans it, and the text always sits inside double quotes. Keep both when changing how gobot is called.
- **What matters goes in `logs/bot.log`, not stderr.** tintin leaves gobot's stderr on its screen, and tmux keeps about two hours of that. So `endRun` logs a fatal error or a bug under the run's flags, and flag errors are logged before gobot exits. What only tintin sees, like a drop or a login gone wrong, `katumi.tin`'s `botlog` writes there as a `tintin:` line, dated the way gobot dates its own. `checkBotLog` in `reconnect_test.go` pins those lines through each login the tests put it through.
- **Two tintin setups call gobot.** Katumi runs it from `/srv/bot`. `torilmud.tin`, in the private mud repo, runs `-tell "stat ..."`, `-item` and `-glist` from `~/mud/torilmud`. A change to those flags, to what they print, or to where gobot looks for `TorilDB.db` and `logs/` has to work for both. `TORILMUD_TIN=<path to torilmud.tin> go test -run Tintin ./...` checks the second.
- **NyyLIB reads Katumi's `clist` and `char` replies, the NHC welcome, and the published `toril.db`.** `Test_NyyLIB` and `Test_DBCopyItems` pin them. Don't reword those lines or reshape those tables to get a test passing. Say what would change in the pull request, because Nyyrazzilyss has to be told first.
- **The published `toril.db` is an allowlist.** Only what `dbcopy_public.sql` lists is copied. Real names (`accounts.player_name`), hidden alts (`vis = 'f'`), load reports and the `online` table stay out, and nothing may add them. A new table or column stays private until it is listed there.
- **`init_db.sql` has to match the live database**, lookup rows included. A class or race added by hand on the server goes into it too, and into `testNewClasses` in `full_test.go`. A new table also goes into `botTables` in `db.go` (`Test_BotTables` checks), and into the live database before that gobot is installed: `-bak` and `-res` refuse a database that lacks one of those tables.
- **Never in the repo:** `toril.db`, `TorilDB.db`, `torildb.url` (Kegor's download link is his to share), `tokens.tin`, `ntfy.url`. `.gitignore` covers them.
- **No inline script or style in `html/`.** grokh.com sends `Content-Security-Policy: default-src 'self'`.
- **Begin every transaction on `toril.db` with `Begin()` on `OpenDB`'s handle**, which starts it with `BEGIN IMMEDIATE`. A transaction that reads before its first write, begun any other way, gets "database is locked" at once whenever another gobot is writing, without the 5-second wait. `testWhoWaits` checks `WhoBatch`.
- The live `toril.db` stays in rollback-journal mode, by the owner's choice. `dbcopy.sh` works with either mode.

## Previewing the site

The page links `/torileq/style.css` and `/torileq/form.js`, and nginx strips that prefix on the server. A bare `gobot -port 8080` therefore shows the page unstyled. Put a reverse proxy in front that strips `/torileq/`, and run gobot in a folder holding a `TorilDB.db` copy, `html/` and an empty `logs/`.

## The server

The checkout is `~/go/src/github.com/grokh/gobot`. A merged change goes live when the owner pulls there, and then:

- the bot: `go install`, and the next event uses the new binary
- the site: `./publish.sh`
- `katumi.tin`: copied to `/srv/bot`, then Katumi's tintin restarted. If it passes gobot a flag gobot didn't take before, gobot is installed first: an older one rejects the flag, logs `Bad flags for ...`, and the event is lost.
- `dbcopy.sh`, its two SQL files and `torildb.sh`: nothing more, since cron runs them from the checkout

The owner does all of that. If you are asked to look at the server, read only, and mind the bot's database: a query on the live `/srv/bot/toril.db` holds a read lock for as long as it runs, and the bot's writes then fail with "database is locked". For anything slower than a count, copy it first (`sqlite3 /srv/bot/toril.db ".backup <tmp>/toril.db"`) and query the copy. Put `timeout 20` in front of every remote `sqlite3`.

## Working here

- Branch from `origin/main` and open a pull request. The owner merges.
- A shared checkout may sit on an old branch. Run `git fetch` and read `origin/main` before trusting what its files show.
- On Windows:
  - The tests use Git for Windows' `sh`, `gzip` and `zcat`, and WinTin++ for `Test_Tintin`.
  - The Windows `sqlite3` can't open MSYS paths such as `/c/x` or `/tmp/x`. Give it `C:/x` (`cygpath -m`).
  - A checkout with CRLF line endings makes `gofmt -l .` list every file. Check one with `tr -d '\r' < file.go | gofmt -l`.
