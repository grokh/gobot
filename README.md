gobot
=====

TorilMUD Bot for item stat database, account and character tracking, boot tracking, and load reporting.

Originally created for:

* CentOS 6
* Tintin++ 2.01.0
* Go 1.0
* SQLite 3.6

Go SQLite3 package: go get github.com/mattn/go-sqlite3

To initialize DB:

```
sqlite3 toril.db
PRAGMA foreign_keys = ON;
.read init_db.sql
.read dump.sql
```

To build on Mac OS X:

* Install Homebrew: http://mxcl.github.io/homebrew/

```
brew install pkgconfig
brew install sqlite
export PKG_CONFIG_PATH="/usr/local/Cellar/sqlite/3.7.16.2/lib/pkgconfig/"
```
