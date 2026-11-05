-- toril.db: the characters, accounts, boots and load reports the bot keeps.
-- Item stats come from Kegor's TorilDB.db instead (see items.go); the item
-- tables toril.db used to have are in this file's history.

-- create new character tracking tables
BEGIN TRANSACTION;
CREATE TABLE race_types(
	race_type TEXT PRIMARY KEY
	,anti_flag TEXT NOT NULL
);
INSERT INTO "race_types" VALUES('Good','ANTI-GOODRACE');
INSERT INTO "race_types" VALUES('Evil','ANTI-EVILRACE');
INSERT INTO "race_types" VALUES('Neutral','');
CREATE TABLE races(
	race_name TEXT PRIMARY KEY
	,race_abbr TEXT NOT NULL
	,anti_flag TEXT NOT NULL
	,race_type TEXT REFERENCES race_types(race_type) NOT NULL
);
INSERT INTO "races" VALUES('Human','Human','ANTI-HUMAN','Good');
INSERT INTO "races" VALUES('Barbarian','Barb','ANTI-BARBARIAN','Good');
INSERT INTO "races" VALUES('Half-Elf','Helf','ANTI-HALFELF','Good');
INSERT INTO "races" VALUES('Shield Dwarf','Dwarf','ANTI-DWARF','Good');
INSERT INTO "races" VALUES('Half-Orc','Horc','ANTI-HALFORC','Neutral');
INSERT INTO "races" VALUES('Duergar','Duergar','ANTI-DUERGAR','Evil');
INSERT INTO "races" VALUES('Gnome','Gnome','ANTI-GNOME','Good');
INSERT INTO "races" VALUES('Halfling','Halfling','ANTI-HALFLING','Good');
INSERT INTO "races" VALUES('Ogre','Ogre','ANTI-OGRE','Evil');
INSERT INTO "races" VALUES('Troll','Troll','ANTI-TROLL','Evil');
INSERT INTO "races" VALUES('Illithid','Squid','ANTI-ILLITHID','Evil');
INSERT INTO "races" VALUES('Orc','Orc','ANTI-ORC','Evil');
INSERT INTO "races" VALUES('Yuan-Ti','Yuan','ANTI-YUANTI','Evil');
INSERT INTO "races" VALUES('Drow Elf','Drow','ANTI-DROWELF','Evil');
INSERT INTO "races" VALUES('Moon Elf','Elf','ANTI-GREYELF','Good');
CREATE TABLE class_types(
	class_type TEXT PRIMARY KEY
	,anti_flag TEXT NOT NULL
);
INSERT INTO "class_types" VALUES('Fighter','NO-WARRIOR');
INSERT INTO "class_types" VALUES('Priest','NO-CLERIC');
INSERT INTO "class_types" VALUES('Rogue','NO-THIEF');
INSERT INTO "class_types" VALUES('Mage','NO-MAGE');
CREATE TABLE classes(
	class_name TEXT PRIMARY KEY
	,class_abbr TEXT NOT NULL
	,class_type TEXT REFERENCES class_types(class_type) NOT NULL
	,anti_flag TEXT NOT NULL
);
INSERT INTO "classes" VALUES('Warrior','War','Fighter','ANTI-WARRIOR');
INSERT INTO "classes" VALUES('Ranger','Ran','Fighter','ANTI-RANGER');
INSERT INTO "classes" VALUES('Paladin','Pal','Fighter','ANTI-PALADIN');
INSERT INTO "classes" VALUES('Anti-Paladin','A-P','Fighter','ANTI-ANTIPALADIN');
INSERT INTO "classes" VALUES('Dire Raider','Dir','Fighter','ANTI-DIRERAIDER');
INSERT INTO "classes" VALUES('Cleric','Cle','Priest','ANTI-CLERIC');
INSERT INTO "classes" VALUES('Druid','Dru','Priest','ANTI-DRUID');
INSERT INTO "classes" VALUES('Shaman','Sha','Priest','ANTI-SHAMAN');
INSERT INTO "classes" VALUES('Rogue','Rog','Rogue','ANTI-ROGUE');
INSERT INTO "classes" VALUES('Bard','Bar','Rogue','ANTI-BARD');
INSERT INTO "classes" VALUES('Battlechanter','Ctr','Rogue','ANTI-BARD');
INSERT INTO "classes" VALUES('Enchanter','Enc','Mage','ANTI-ENCHANTER');
INSERT INTO "classes" VALUES('Invoker','Inv','Mage','ANTI-INVOKER');
INSERT INTO "classes" VALUES('Elementalist','Ele','Mage','ANTI-ELEMENTALIST');
INSERT INTO "classes" VALUES('Necromancer','Nec','Mage','ANTI-NECROMANCER');
INSERT INTO "classes" VALUES('Illusionist','Ill','Mage','ANTI-ILLUSIONIST');
INSERT INTO "classes" VALUES('Psionicist','Psi','Mage','ANTI-PSIONICIST');
INSERT INTO "classes" VALUES('Lich','Lic','Mage','ANTI-LICH');
CREATE TABLE accounts(
	account_name TEXT PRIMARY KEY
	,player_name TEXT
);
CREATE TABLE chars(
	account_name TEXT REFERENCES accounts(account_name)
	,char_name TEXT
	,class_name TEXT REFERENCES classes(class_name) NOT NULL
	,char_race TEXT REFERENCES races(race_name) NOT NULL
	,char_level INTEGER NOT NULL
	,last_seen DATETIME NOT NULL
	,vis BOOLEAN NOT NULL
	,PRIMARY KEY (account_name, char_name)
);


-- create new boot/load report tables
CREATE TABLE boots(
	boot_id INTEGER PRIMARY KEY
	,boot_time DATETIME NOT NULL
	,uptime TEXT NOT NULL
);
INSERT INTO boots (boot_time, uptime) VALUES(
	(SELECT datetime('now', '-1 minute'))
	, '0:01:00'
);
CREATE TABLE loads(
	boot_id INTEGER REFERENCES boots(boot_id) NOT NULL
	,report_time DATETIME NOT NULL
	,report_text TEXT NOT NULL
	,char_name TEXT NOT NULL
	,deleted BOOLEAN NOT NULL
	,PRIMARY KEY (boot_id, report_time)
);
-- how many players who counted, by UTC hour (see onlineSchema in online.go)
CREATE TABLE online(
	hour DATETIME PRIMARY KEY
	,checks INTEGER NOT NULL
	,total INTEGER NOT NULL
	,low INTEGER NOT NULL
	,high INTEGER NOT NULL
);

COMMIT;