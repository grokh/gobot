-- What dbcopy.sh publishes from the bot's toril.db. It runs this in a new,
-- empty database, with the bot's own attached read-only as bot, so the
-- published copy holds only what's named here: these tables, these columns,
-- these rows. Anything else in the bot's database stays there, and so does
-- whatever is added to it later, until it's added below.
--
-- Left out:
--   * hidden alts: only chars whose vis is 't' are copied.
--   * real names: accounts has its player_name column, since gobot run on a
--     copy reads it, but nothing is ever put in it.
--   * load reports: nothing reads them from a copy, and they'd be a third
--     of the download. They're players' own words, years of them, with the
--     ones they deleted, where lr only ever shows the current boot's. loads
--     is there, empty, so that gobot run on a copy can still answer lr.
--   * online, who's hourly counts: nothing reads them from a copy, and
--     they'd be over 40% of the download.
--   * the item tables the bot kept until Kegor's TorilDB.db replaced them,
--     and the lists that went with them (zones, slots, flags and the rest).
--     dbcopy.sh fills items and item_flags itself.
--
-- Who reads a copy: NyyLIB, the Mudlet package, reads chars, items and
-- item_flags from the download. gobot, run in a folder that has a copy,
-- works from it as the bot does from its own, which takes chars, accounts,
-- boots, races and a loads table, and classes when a char is added.
--
-- The tables are made the way init_db.sql makes them. All of it is read in
-- one transaction, so the copy is the bot's database as it was at one
-- moment, and rows go in in the order the bot has them.

BEGIN;

CREATE TABLE main.race_types(
	race_type TEXT PRIMARY KEY
	,anti_flag TEXT NOT NULL
);
INSERT INTO main.race_types (race_type, anti_flag)
SELECT race_type, anti_flag FROM bot.race_types ORDER BY rowid;

CREATE TABLE main.races(
	race_name TEXT PRIMARY KEY
	,race_abbr TEXT NOT NULL
	,anti_flag TEXT NOT NULL
	,race_type TEXT REFERENCES race_types(race_type) NOT NULL
);
INSERT INTO main.races (race_name, race_abbr, anti_flag, race_type)
SELECT race_name, race_abbr, anti_flag, race_type FROM bot.races
ORDER BY rowid;

CREATE TABLE main.class_types(
	class_type TEXT PRIMARY KEY
	,anti_flag TEXT NOT NULL
);
INSERT INTO main.class_types (class_type, anti_flag)
SELECT class_type, anti_flag FROM bot.class_types ORDER BY rowid;

CREATE TABLE main.classes(
	class_name TEXT PRIMARY KEY
	,class_abbr TEXT NOT NULL
	,class_type TEXT REFERENCES class_types(class_type) NOT NULL
	,anti_flag TEXT NOT NULL
);
INSERT INTO main.classes (class_name, class_abbr, class_type, anti_flag)
SELECT class_name, class_abbr, class_type, anti_flag FROM bot.classes
ORDER BY rowid;

-- every account, by its name alone
CREATE TABLE main.accounts(
	account_name TEXT PRIMARY KEY
	,player_name TEXT
);
INSERT INTO main.accounts (account_name)
SELECT account_name FROM bot.accounts ORDER BY rowid;

-- vis = 't', not vis <> 'f': a char is only published if it's marked
-- visible
CREATE TABLE main.chars(
	account_name TEXT REFERENCES accounts(account_name)
	,char_name TEXT
	,class_name TEXT REFERENCES classes(class_name) NOT NULL
	,char_race TEXT REFERENCES races(race_name) NOT NULL
	,char_level INTEGER NOT NULL
	,last_seen DATETIME NOT NULL
	,vis BOOLEAN NOT NULL
	,PRIMARY KEY (account_name, char_name)
);
INSERT INTO main.chars (account_name, char_name, class_name, char_race,
	char_level, last_seen, vis)
SELECT account_name, char_name, class_name, char_race, char_level,
	last_seen, vis
FROM bot.chars WHERE vis = 't' ORDER BY rowid;

CREATE TABLE main.boots(
	boot_id INTEGER PRIMARY KEY
	,boot_time DATETIME NOT NULL
	,uptime TEXT NOT NULL
);
INSERT INTO main.boots (boot_id, boot_time, uptime)
SELECT boot_id, boot_time, uptime FROM bot.boots ORDER BY boot_id;

-- the table alone, for lr and lrdel: no load reports are copied
CREATE TABLE main.loads(
	boot_id INTEGER REFERENCES boots(boot_id) NOT NULL
	,report_time DATETIME NOT NULL
	,report_text TEXT NOT NULL
	,char_name TEXT NOT NULL
	,deleted BOOLEAN NOT NULL
	,PRIMARY KEY (boot_id, report_time)
);

-- NyyLIB's item tables, empty: dbcopy.sh fills them, from TorilDB.db with
-- dbcopy_items.sql, which has the same two definitions and says what NyyLIB
-- needs of them
CREATE TABLE main.items(
	item_id INTEGER PRIMARY KEY
	,item_name TEXT NOT NULL
	,keywords TEXT NOT NULL
	,weight INTEGER
	,c_value INTEGER
	,item_type TEXT NOT NULL
	,from_zone TEXT NOT NULL
	,from_mob TEXT
	,short_stats TEXT
	,long_stats TEXT
	,full_stats TEXT
	,comments TEXT
	,last_id DATE
);
CREATE TABLE main.item_flags(
	item_id INTEGER REFERENCES items(item_id)
	,flag_abbr TEXT
	,PRIMARY KEY (item_id, flag_abbr)
);

COMMIT;
