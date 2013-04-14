gobot
=====

TorilMUD Bot for item stat database, account and character tracking, boot tracking, and load reporting.

Originally created for:

* CentOS 6
* Tintin++ 2.01.0
* Go 1.0
* SQLite 3.6

Originally created with:

* Chrome Secure Shell 0.8
* vim 7.2
* Lots of Google searches

SQLite3 package: go get github.com/mattn/go-sqlite3

Initialize DB:

* sqlite3 toril.db
* PRAGMA foreign_keys = ON;
* .read init_db.sql
* .read dump.sql
