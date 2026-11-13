# gobot

Katumi, a helper bot for TorilMUD, and the TorilEQ item search at grokh.com/torileq/. It is one Go program (`package main`, no subpackages) that tintin++ runs once for each event. README.md describes all of it: read it before changing how anything behaves, and update it in the same change.

## Checks

    gofmt -l .                       # must print nothing
    go vet ./... && go test ./...

- They need gcc, since go-sqlite3 compiles SQLite, and the `sqlite3` CLI. The first run takes about a minute.
- `Test_Tintin` runs `katumi.tin` in tt++ 2.02.20 or later, and skips without one. `go test -short` leaves it out.
- CI (`.github/workflows/ci.yml`) runs the same checks on every pull request, and fails if `Test_Tintin` skips.
- `ChkErr` ends the program with `log.Fatal`. In `Test_All` the log goes to `logs/test.log`, so a database call that fails there stops the whole test binary with nothing on the console but `FAIL`. Look in that file.

## Where things are

| File | Holds |
|---|---|
| `gobot.go` | the flags, `main`, and the log's clean-up of what players send |
| `tell.go` | every tell command and its help, `tellSafe`, `tintinSafe`, `maxReplyBytes` |
| `items.go`, `mobs.go` | lookups in Kegor's `TorilDB.db` (its `eq` and `mobs` tables) |
| `guess.go` | what Katumi suggests for a mistyped tell |
| `who.go`, `time.go`, `online.go` | who lists, boots, hourly player counts |
| `web.go`, `html/` | the TorilEQ site (`gobot -port`) |
| `db.go` | opening, backing up and restoring `toril.db` |
| `local.go` | `-glist` |
| `ntfy.go` | reboot alerts |
| `katumi.tin` | the tintin script that logs Katumi in and calls gobot |
| `init_db.sql` | the schema of the bot's `toril.db`, which the tests build from |
| `dbcopy.sh`, `dbcopy_public.sql`, `dbcopy_items.sql` | the published copy of `toril.db` |
| `torildb.sh`, `publish.sh`, `torileq.service` | what the server runs |

## Rules

- **Nothing here may connect to torilmud.com.** `katumi.tin` logs in as soon as tintin starts, so never run `tt++ katumi.tin` by hand. The tests read copies with the `#session` lines and login events taken out, and refuse to run if one is left. Keep that guard.
- **Replies are run by tintin as commands.** Every reply line goes through `tellSafe`, and text from `TorilDB.db` through `tintinSafe` as well. A reply sends the MUD at most `maxReplyBytes`.
- **MUD text reaches gobot through a shell.** `shellsafe` in `katumi.tin` cleans it, and the text always sits inside double quotes. Keep both when changing how gobot is called.
- **Two tintin setups call gobot.** Katumi runs it from `/srv/bot`. `torilmud.tin`, in the private mud repo, runs `-tell "stat ..."`, `-item` and `-glist` from `~/mud/torilmud`. A change to those flags, to what they print, or to where gobot looks for `TorilDB.db` and `logs/` has to work for both. `TORILMUD_TIN=<path to torilmud.tin> go test -run Tintin ./...` checks the second.
- **NyyLIB reads Katumi's `clist` and `char` replies, the NHC welcome, and the published `toril.db`.** `Test_NyyLIB` and `Test_DBCopyItems` pin them. Don't reword those lines or reshape those tables to get a test passing. Say what would change in the pull request, because Nyyrazzilyss has to be told first.
- **The published `toril.db` is an allowlist.** Only what `dbcopy_public.sql` lists is copied. Real names (`accounts.player_name`), hidden alts (`vis = 'f'`), load reports and the `online` table stay out, and nothing may add them. A new table or column stays private until it is listed there.
- **`init_db.sql` has to match the live database**, lookup rows included. A class or race added by hand on the server goes into it too, and into `testNewClasses` in `full_test.go`.
- **Never in the repo:** `toril.db`, `TorilDB.db`, `torildb.url` (Kegor's download link is his to share), `tokens.tin`, `ntfy.url`. `.gitignore` covers them.
- **No inline script or style in `html/`.** grokh.com sends `Content-Security-Policy: default-src 'self'`.
- The live `toril.db` stays in rollback-journal mode, by the owner's choice. `dbcopy.sh` works with either mode.

## Previewing the site

The page links `/torileq/style.css` and `/torileq/form.js`, and nginx strips that prefix on the server. A bare `gobot -port 8080` therefore shows the page unstyled. Put a reverse proxy in front that strips `/torileq/`, and run gobot in a folder holding a `TorilDB.db` copy, `html/` and an empty `logs/`.

## The server

The checkout is `~/go/src/github.com/grokh/gobot`. A merged change goes live when the owner pulls there, and then:

- the bot: `go install`, and the next event uses the new binary
- the site: `./publish.sh`
- `katumi.tin`: copied to `/srv/bot`, then Katumi's tintin restarted
- `dbcopy.sh`, its two SQL files and `torildb.sh`: nothing more, since cron runs them from the checkout

The owner does all of that. If you are asked to look at the server, read only, and mind the bot's database: a query on the live `/srv/bot/toril.db` holds a read lock for as long as it runs, and the bot's writes then fail with "database is locked". For anything slower than a count, copy it first (`sqlite3 /srv/bot/toril.db ".backup <tmp>/toril.db"`) and query the copy. Put `timeout 20` in front of every remote `sqlite3`.

## Working here

- Branch from `origin/main` and open a pull request. The owner merges.
- A shared checkout may sit on an old branch. Run `git fetch` and read `origin/main` before trusting what its files show.
- On Windows:
  - The tests use Git for Windows' `sh`, `gzip` and `zcat`, and WinTin++ for `Test_Tintin`.
  - The Windows `sqlite3` can't open MSYS paths such as `/c/x` or `/tmp/x`. Give it `C:/x` (`cygpath -m`).
  - A checkout with CRLF line endings makes `gofmt -l .` list every file. Check one with `tr -d '\r' < file.go | gofmt -l`.
