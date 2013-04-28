-- Update existing tables
UPDATE items SET item_type =
	(SELECT item_types.item_type FROM item_types
		WHERE item_types.type_abbr = items.item_type);
UPDATE item_attribs SET attrib_abbr = 
	(SELECT attrib_name FROM attribs 
		WHERE item_attribs.attrib_abbr = attribs.attrib_abbr);
UPDATE item_effects SET effect_abbr = 
	(SELECT effect_name FROM effects 
		WHERE item_effects.effect_abbr = effects.effect_abbr);
UPDATE item_flags SET flag_abbr = 
	(SELECT flag_name FROM flags 
		WHERE item_flags.flag_abbr = flags.flag_abbr);
UPDATE item_resists SET resist_abbr = 
	(SELECT resist_name FROM resists 
		WHERE item_resists.resist_abbr = resists.resist_abbr);
UPDATE item_restricts SET restrict_abbr = 
	(SELECT restrict_name FROM restricts 
		WHERE item_restricts.restrict_abbr = restricts.restrict_abbr);
UPDATE item_slots SET slot_abbr = 
	(SELECT worn_slot FROM slots 
		WHERE item_slots.slot_abbr = slots.slot_abbr);
UPDATE specials SET item_type = 
	(SELECT item_type FROM item_types 
		WHERE item_types.type_abbr = specials.item_type);
UPDATE item_specials SET item_type = 
	(SELECT item_type FROM item_types
		WHERE item_specials.item_type = item_types.type_abbr);


BEGIN TRANSACTION;
-- create new character tracking tables
CREATE TEMPORARY TABLE race_types_bak(
	race_type TEXT PRIMARY KEY
	,anti_flag TEXT NOT NULL
);
INSERT INTO race_types_bak
	SELECT race_type, anti_flag
	FROM race_types;
DROP TABLE race_types;
CREATE TEMPORARY TABLE races_bak(
	race_name TEXT PRIMARY KEY
	,race_abbr TEXT NOT NULL
	,anti_flag TEXT NOT NULL
	,race_type TEXT REFERENCES race_types_bak(race_type) NOT NULL
);
INSERT INTO races_bak
	SELECT race_name, race_abbr, anti_flag, race_type
	FROM races;
DROP TABLE races;
CREATE TEMPORARY TABLE class_types_bak(
	class_type TEXT PRIMARY KEY
	,anti_flag TEXT NOT NULL
);
INSERT INTO class_types_bak
	SELECT class_type, anti_flag
	FROM class_types;
DROP TABLE class_types;
CREATE TEMPORARY TABLE classes_bak(
	class_name TEXT PRIMARY KEY
	,class_abbr TEXT NOT NULL
	,class_type TEXT REFERENCES class_types_bak(class_type) NOT NULL
	,anti_flag TEXT NOT NULL
);
INSERT INTO classes_bak
	SELECT class_name, class_abbr, class_type, anti_flag
	FROM classes;
DROP TABLE classes;
CREATE TEMPORARY TABLE accounts_bak(
	account_name TEXT PRIMARY KEY
	,player_name TEXT
);
INSERT INTO accounts_bak
	SELECT account_name, player_name
	FROM accounts;
DROP TABLE accounts;
CREATE TEMPORARY TABLE chars_bak(
	account_name TEXT REFERENCES accounts_bak(account_name)
	,char_name TEXT
	,class_name TEXT REFERENCES classes_bak(class_name) NOT NULL
	,char_race TEXT REFERENCES races_bak(race_name) NOT NULL
	,char_level INTEGER NOT NULL
	,last_seen DATETIME NOT NULL
	,vis BOOLEAN NOT NULL
	,PRIMARY KEY (account_name, char_name)
);
INSERT INTO chars_bak
	SELECT account_name, char_name, class_name, char_race, char_level, last_seen, vis
	FROM chars;
DROP TABLE chars;


-- create new boot/load report tables
CREATE TEMPORARY TABLE boots_bak(
	boot_id INTEGER PRIMARY KEY
	,boot_time DATETIME NOT NULL
	,uptime TEXT NOT NULL
);
INSERT INTO boots_bak
	SELECT boot_id, boot_time, uptime
	FROM boots;
DROP TABLE boots;
CREATE TEMPORARY TABLE loads_bak(
	boot_id INTEGER REFERENCES boots_bak(boot_id) NOT NULL
	,report_time DATETIME NOT NULL
	,report_text TEXT NOT NULL
	,char_name TEXT NOT NULL
	,deleted BOOLEAN NOT NULL
	,PRIMARY KEY (boot_id, report_time)
);
INSERT INTO loads_bak
	SELECT boot_id, report_time, report_text, char_name, deleted
	FROM loads;
DROP TABLE loads;


-- create new item stat tables
CREATE TEMPORARY TABLE enchants_bak(
	ench_name TEXT PRIMARY KEY
	,ench_desc TEXT NOT NULL
);
INSERT INTO enchants_bak
	SELECT ench_name, ench_desc
	FROM enchants;
DROP TABLE enchants;
CREATE TEMPORARY TABLE attribs_bak(
	attrib_name TEXT PRIMARY KEY
	,attrib_abbr TEXT NOT NULL
	,attrib_display TEXT NOT NULL
);
INSERT INTO attribs_bak
	SELECT attrib_name, attrib_abbr, attrib_display
	FROM attribs;
DROP TABLE attribs;
CREATE TEMPORARY TABLE effects_bak(
	effect_name TEXT PRIMARY KEY
	,effect_abbr TEXT NOT NULL
	,effect_display TEXT NOT NULL
);
INSERT INTO effects_bak
	SELECT effect_name, effect_abbr, effect_display
	FROM effects;
DROP TABLE effects;
CREATE TEMPORARY TABLE resists_bak(
	resist_name TEXT PRIMARY KEY
	,resist_abbr TEXT NOT NULL
	,resist_display TEXT NOT NULL
);
INSERT INTO resists_bak
	SELECT resist_name, resist_abbr, resist_display
	FROM resists;
DROP TABLE resists;
CREATE TEMPORARY TABLE restricts_bak(
	restrict_name TEXT PRIMARY KEY
	,restrict_abbr TEXT NOT NULL
);
INSERT INTO restricts_bak
	SELECT restrict_name, restrict_abbr
	FROM restricts;
DROP TABLE restricts;
CREATE TEMPORARY TABLE flags_bak(
	flag_name TEXT PRIMARY KEY
	,flag_abbr TEXT NOT NULL
	,flag_display TEXT NOT NULL
);
INSERT INTO flags_bak
	SELECT flag_name, flag_abbr, flag_display
	FROM flags;
DROP TABLE flags;
CREATE TEMPORARY TABLE slots_bak(
	worn_slot TEXT PRIMARY KEY
	,slot_abbr TEXT NOT NULL
	,slot_display TEXT NOT NULL
);
INSERT INTO slots_bak
	SELECT worn_slot, slot_abbr, slot_display
	FROM slots;
DROP TABLE slots;
CREATE TEMPORARY TABLE item_types_bak(
	item_type TEXT PRIMARY KEY
	,type_abbr TEXT NOT NULL
	,type_display TEXT NOT NULL
);
INSERT INTO item_types_bak
	SELECT item_type, type_abbr, type_display
	FROM item_types;
DROP TABLE item_types;
CREATE TEMPORARY TABLE zones_bak(
	zone_abbr TEXT PRIMARY KEY
	,zone_name TEXT NOT NULL
);
INSERT INTO zones_bak
	SELECT zone_abbr, zone_name
	FROM zones;
DROP TABLE zones;
CREATE TEMPORARY TABLE mobs_bak(
	mob_name TEXT PRIMARY KEY
	,mob_abbr TEXT
	,from_zone TEXT REFERENCES zones_bak(zone_abbr)
);
INSERT INTO mobs_bak
	SELECT mob_name, mob_abbr, from_zone
	FROM mobs;
DROP TABLE mobs;
CREATE TEMPORARY TABLE specials_bak(
	item_type TEXT REFERENCES item_types_bak(item_type)
	,spec_abbr TEXT NOT NULL
	,spec_display TEXT NOT NULL
	,PRIMARY KEY (item_type, spec_abbr)
);
INSERT INTO specials_bak
	SELECT item_type, spec_abbr, spec_display
	FROM specials;
DROP TABLE specials;
CREATE TEMPORARY TABLE supps_bak(
	supp_abbr TEXT PRIMARY KEY
	,supp_display TEXT NOT NULL
	,supp_value TEXT NOT NULL
);
INSERT INTO supps_bak
	SELECT supp_abbr, supp_display, supp_value
	FROM supps;
DROP TABLE supps;
CREATE TEMPORARY TABLE items_bak(
	item_id INTEGER PRIMARY KEY
	,item_name TEXT NOT NULL
	,keywords TEXT NOT NULL
	,weight INTEGER
	,c_value INTEGER
	,item_type TEXT REFERENCES item_types_bak(item_type) NOT NULL
	,from_zone TEXT REFERENCES zones_bak(zone_abbr) NOT NULL
	,from_mob TEXT REFERENCES mobs_bak(mob_name)
	,short_stats TEXT
	,long_stats TEXT
	,full_stats TEXT
	,comments TEXT
	,last_id DATE
);
INSERT INTO items_bak
	SELECT item_id, item_name, keywords, weight, c_value, item_type, from_zone,
	from_mob, short_stats, long_stats, full_stats, comments, last_id
	FROM items;
DROP TABLE items;
CREATE TEMPORARY TABLE item_procs_bak(
	item_id INTEGER REFERENCES items_bak(item_id)
	,proc_name TEXT NOT NULL
	,proc_type TEXT
	,proc_desc TEXT
	,proc_trig TEXT
	,proc_effect TEXT
	,PRIMARY KEY (item_id, proc_name)
);
INSERT INTO item_procs_bak
	SELECT item_id, proc_name, proc_type, proc_desc, proc_trig, proc_effect
	FROM item_procs;
DROP TABLE item_procs;
CREATE TEMPORARY TABLE item_slots_bak(
	item_id INTEGER REFERENCES items_bak(item_id)
	,worn_slot TEXT REFERENCES slots_bak(worn_slot)
	,PRIMARY KEY (item_id, worn_slot)
);
INSERT INTO item_slots_bak
	SELECT item_id, slot_abbr
	FROM item_slots;
DROP TABLE item_slots;
CREATE TEMPORARY TABLE item_flags_bak(
	item_id INTEGER REFERENCES items_bak(item_id)
	,flag_name TEXT REFERENCES flags_bak(flag_name)
	,PRIMARY KEY (item_id, flag_name)
);
INSERT INTO item_flags_bak
	SELECT item_id, flag_abbr
	FROM item_flags;
DROP TABLE item_flags;
CREATE TEMPORARY TABLE item_restricts_bak(
	item_id INTEGER REFERENCES items_bak(item_id)
	,restrict_name TEXT REFERENCES restricts_bak(restrict_name)
	,PRIMARY KEY (item_id, restrict_name)
);
INSERT INTO item_restricts_bak
	SELECT item_id, restrict_abbr
	FROM item_restricts;
DROP TABLE item_restricts;
CREATE TEMPORARY TABLE item_resists_bak(
	item_id INTEGER REFERENCES items_bak(item_id)
	,resist_name TEXT REFERENCES resists_bak(resist_name)
	,resist_value INTEGER NOT NULL
	,PRIMARY KEY (item_id, resist_name)
);
INSERT INTO item_resists_bak
	SELECT item_id, resist_abbr, resist_value
	FROM item_resists;
DROP TABLE item_resists;
CREATE TEMPORARY TABLE item_effects_bak(
	item_id INTEGER REFERENCES items_bak(item_id)
	,effect_name TEXT REFERENCES effects_bak(effect_name)
	,PRIMARY KEY (item_id, effect_name)
);
INSERT INTO item_effects_bak
	SELECT item_id, effect_abbr
	FROM item_effects;
DROP TABLE item_effects;
CREATE TEMPORARY TABLE item_specials_bak(
	item_id INTEGER REFERENCES items(item_id)
	,item_type TEXT
	,spec_abbr TEXT
	,spec_value TEXT NOT NULL
	,FOREIGN KEY (item_type, spec_abbr) REFERENCES specials_bak (item_type, spec_abbr)
	,PRIMARY KEY (item_id, item_type, spec_abbr)
);
INSERT INTO item_specials_bak
	SELECT item_id, item_type, spec_abbr, spec_value
	FROM item_specials;
DROP TABLE item_specials;
CREATE TEMPORARY TABLE item_enchants_bak(
	item_id INTEGER REFERENCES items_bak(item_id)
	,ench_name TEXT REFERENCES enchants_bak(ench_name)
	,dam_pct INTEGER NOT NULL
	,freq_pct INTEGER NOT NULL
	,sv_mod INTEGER NOT NULL
	,duration INTEGER NOT NULL
	,PRIMARY KEY (item_id, ench_name)
);
INSERT INTO item_enchants_bak
	SELECT item_id, ench_name, dam_pct, freq_pct, sv_mod, duration
	FROM item_enchants;
DROP TABLE item_enchants;
CREATE TEMPORARY TABLE item_attribs_bak(
	item_id INTEGER REFERENCES items_bak(item_id)
	,attrib_name TEXT REFERENCES attribs_bak(attrib_name)
	,attrib_value INTEGER NOT NULL
	,PRIMARY KEY (item_id, attrib_name)
);
INSERT INTO item_attribs_bak
	SELECT item_id, attrib_abbr, attrib_value
	FROM item_attribs;
DROP TABLE item_attribs;
CREATE TEMPORARY TABLE item_supps_bak(
	item_id INTEGER REFERENCES items_bak(item_id)
	,supp_abbr TEXT REFERENCES supps_bak(supp_abbr)
);
INSERT INTO item_supps_bak
	SELECT item_id, supp_abbr
	FROM item_supps;
DROP TABLE item_supps;


-- rebuild new, good tables!
-- create new character tracking tables
CREATE TABLE race_types(
	race_type TEXT PRIMARY KEY
	,anti_flag TEXT NOT NULL
);
INSERT INTO race_types
	SELECT race_type, anti_flag
	FROM race_types_bak;
DROP TABLE race_types_bak;
CREATE TABLE races(
	race_name TEXT PRIMARY KEY
	,race_abbr TEXT NOT NULL
	,anti_flag TEXT NOT NULL
	,race_type TEXT REFERENCES race_types(race_type) NOT NULL
);
INSERT INTO races
	SELECT race_name, race_abbr, anti_flag, race_type
	FROM races_bak;
DROP TABLE races_bak;
CREATE TABLE class_types(
	class_type TEXT PRIMARY KEY
	,anti_flag TEXT NOT NULL
);
INSERT INTO class_types
	SELECT class_type, anti_flag
	FROM class_types_bak;
DROP TABLE class_types_bak;
CREATE TABLE classes(
	class_name TEXT PRIMARY KEY
	,class_abbr TEXT NOT NULL
	,class_type TEXT REFERENCES class_types(class_type) NOT NULL
	,anti_flag TEXT NOT NULL
);
INSERT INTO classes
	SELECT class_name, class_abbr, class_type, anti_flag
	FROM classes_bak;
DROP TABLE classes_bak;
CREATE TABLE accounts(
	account_name TEXT PRIMARY KEY
	,player_name TEXT
);
INSERT INTO accounts
	SELECT account_name, player_name
	FROM accounts_bak;
DROP TABLE accounts_bak;
CREATE TABLE chars(
	account_name TEXT REFERENCES accounts(account_name)
	,char_name TEXT
	,class_name TEXT REFERENCES classes(class_name) NOT NULL
	,char_race TEXT REFERENCES races(race_name) NOT NULL
	,char_level INTEGER NOT NULL
	,last_seen TIMESTAMP NOT NULL
	,vis BOOLEAN NOT NULL
	,PRIMARY KEY (account_name, char_name)
);
INSERT INTO chars
	SELECT account_name, char_name, class_name, char_race, char_level, last_seen, vis
	FROM chars_bak;
DROP TABLE chars_bak;


-- create new boot/load report tables
CREATE TABLE boots(
	boot_id INTEGER PRIMARY KEY
	,boot_time TIMESTAMP NOT NULL
	,uptime TEXT NOT NULL
);
INSERT INTO boots
	SELECT boot_id, boot_time, uptime
	FROM boots_bak;
DROP TABLE boots_bak;
CREATE TABLE loads(
	boot_id INTEGER REFERENCES boots(boot_id) NOT NULL
	,report_time TIMESTAMP NOT NULL
	,report_text TEXT NOT NULL
	,char_name TEXT NOT NULL
	,deleted BOOLEAN NOT NULL
	,PRIMARY KEY (boot_id, report_time)
);
INSERT INTO loads
	SELECT boot_id, report_time, report_text, char_name, deleted
	FROM loads_bak;
DROP TABLE loads_bak;


-- create new item stat tables
CREATE TABLE enchants(
	ench_name TEXT PRIMARY KEY
	,ench_desc TEXT NOT NULL
);
INSERT INTO enchants
	SELECT ench_name, ench_desc
	FROM enchants_bak;
DROP TABLE enchants_bak;
CREATE TABLE attribs(
	attrib_name TEXT PRIMARY KEY
	,attrib_abbr TEXT NOT NULL
	,attrib_display TEXT NOT NULL
);
INSERT INTO attribs
	SELECT attrib_name, attrib_abbr, attrib_display
	FROM attribs_bak;
DROP TABLE attribs_bak;
CREATE TABLE effects(
	effect_name TEXT PRIMARY KEY
	,effect_abbr TEXT NOT NULL
	,effect_display TEXT NOT NULL
);
INSERT INTO effects
	SELECT effect_name, effect_abbr, effect_display
	FROM effects_bak;
DROP TABLE effects_bak;
CREATE TABLE resists(
	resist_name TEXT PRIMARY KEY
	,resist_abbr TEXT NOT NULL
	,resist_display TEXT NOT NULL
);
INSERT INTO resists
	SELECT resist_name, resist_abbr, resist_display
	FROM resists_bak;
DROP TABLE resists_bak;
CREATE TABLE restricts(
	restrict_name TEXT PRIMARY KEY
	,restrict_abbr TEXT NOT NULL
);
INSERT INTO restricts
	SELECT restrict_name, restrict_abbr
	FROM restricts_bak;
DROP TABLE restricts_bak;
CREATE TABLE flags(
	flag_name TEXT PRIMARY KEY
	,flag_abbr TEXT NOT NULL
	,flag_display TEXT NOT NULL
);
INSERT INTO flags
	SELECT flag_name, flag_abbr, flag_display
	FROM flags_bak;
DROP TABLE flags_bak;
CREATE TABLE slots(
	worn_slot TEXT PRIMARY KEY
	,slot_abbr TEXT NOT NULL
	,slot_display TEXT NOT NULL
);
INSERT INTO slots
	SELECT worn_slot, slot_abbr, slot_display
	FROM slots_bak;
DROP TABLE slots_bak;
CREATE TABLE item_types(
	item_type TEXT PRIMARY KEY
	,type_abbr TEXT NOT NULL
	,type_display TEXT NOT NULL
);
INSERT INTO item_types
	SELECT item_type, type_abbr, type_display
	FROM item_types_bak;
DROP TABLE item_types_bak;
CREATE TABLE zones(
	zone_abbr TEXT PRIMARY KEY
	,zone_name TEXT NOT NULL
);
INSERT INTO zones
	SELECT zone_abbr, zone_name
	FROM zones_bak;
DROP TABLE zones_bak;
CREATE TABLE mobs(
	mob_name TEXT PRIMARY KEY
	,mob_abbr TEXT
	,from_zone TEXT REFERENCES zones(zone_abbr)
);
INSERT INTO mobs
	SELECT mob_name, mob_abbr, from_zone
	FROM mobs_bak;
DROP TABLE mobs_bak;
CREATE TABLE specials(
	item_type TEXT REFERENCES item_types(item_type)
	,spec_abbr TEXT NOT NULL
	,spec_display TEXT NOT NULL
	,PRIMARY KEY (item_type, spec_abbr)
);
INSERT INTO specials
	SELECT item_type, spec_abbr, spec_display
	FROM specials_bak;
DROP TABLE specials_bak;
CREATE TABLE supps(
	supp_abbr TEXT PRIMARY KEY
	,supp_display TEXT NOT NULL
	,supp_value TEXT NOT NULL
);
INSERT INTO supps
	SELECT supp_abbr, supp_display, supp_value
	FROM supps_bak;
DROP TABLE supps_bak;
CREATE TABLE items(
	item_id INTEGER PRIMARY KEY
	,item_name TEXT NOT NULL
	,keywords TEXT NOT NULL
	,weight INTEGER
	,c_value INTEGER
	,item_type TEXT REFERENCES item_types(item_type) NOT NULL
	,from_zone TEXT REFERENCES zones(zone_abbr) NOT NULL
	,from_mob TEXT REFERENCES mobs(mob_name)
	,short_stats TEXT
	,long_stats TEXT
	,full_stats TEXT
	,comments TEXT
	,last_id DATE
);
INSERT INTO items
	SELECT item_id, item_name, keywords, weight, c_value, item_type, from_zone,
	from_mob, short_stats, long_stats, full_stats, comments, last_id
	FROM items_bak;
DROP TABLE items_bak;
CREATE TABLE item_procs(
	item_id INTEGER REFERENCES items(item_id)
	,proc_name TEXT NOT NULL
	,proc_type TEXT
	,proc_desc TEXT
	,proc_trig TEXT
	,proc_effect TEXT
	,PRIMARY KEY (item_id, proc_name)
);
INSERT INTO item_procs
	SELECT item_id, proc_name, proc_type, proc_desc, proc_trig, proc_effect
	FROM item_procs_bak;
DROP TABLE item_procs_bak;
CREATE TABLE item_slots(
	item_id INTEGER REFERENCES items(item_id)
	,worn_slot TEXT REFERENCES slots(worn_slot)
	,PRIMARY KEY (item_id, worn_slot)
);
INSERT INTO item_slots
	SELECT item_id, worn_slot
	FROM item_slots_bak;
DROP TABLE item_slots_bak;
CREATE TABLE item_flags(
	item_id INTEGER REFERENCES items(item_id)
	,flag_name TEXT REFERENCES flags(flag_name)
	,PRIMARY KEY (item_id, flag_name)
);
INSERT INTO item_flags
	SELECT item_id, flag_name
	FROM item_flags_bak;
DROP TABLE item_flags_bak;
CREATE TABLE item_restricts(
	item_id INTEGER REFERENCES items(item_id)
	,restrict_name TEXT REFERENCES restricts(restrict_name)
	,PRIMARY KEY (item_id, restrict_name)
);
INSERT INTO item_restricts
	SELECT item_id, restrict_name
	FROM item_restricts_bak;
DROP TABLE item_restricts_bak;
CREATE TABLE item_resists(
	item_id INTEGER REFERENCES items(item_id)
	,resist_name TEXT REFERENCES resists(resist_name)
	,resist_value INTEGER NOT NULL
	,PRIMARY KEY (item_id, resist_name)
);
INSERT INTO item_resists
	SELECT item_id, resist_name, resist_value
	FROM item_resists_bak;
DROP TABLE item_resists_bak;
CREATE TABLE item_effects(
	item_id INTEGER REFERENCES items(item_id)
	,effect_name TEXT REFERENCES effects(effect_name)
	,PRIMARY KEY (item_id, effect_name)
);
INSERT INTO item_effects
	SELECT item_id, effect_name
	FROM item_effects_bak;
DROP TABLE item_effects_bak;
CREATE TABLE item_specials(
	item_id INTEGER REFERENCES items(item_id)
	,item_type TEXT
	,spec_abbr TEXT
	,spec_value TEXT NOT NULL
	,FOREIGN KEY (item_type, spec_abbr) REFERENCES specials (item_type, spec_abbr)
	,PRIMARY KEY (item_id, item_type, spec_abbr)
);
INSERT INTO item_specials
	SELECT item_id, item_type, spec_abbr, spec_value
	FROM item_specials_bak;
DROP TABLE item_specials_bak;
CREATE TABLE item_enchants(
	item_id INTEGER REFERENCES items(item_id)
	,ench_name TEXT REFERENCES enchants(ench_name)
	,dam_pct INTEGER NOT NULL
	,freq_pct INTEGER NOT NULL
	,sv_mod INTEGER NOT NULL
	,duration INTEGER NOT NULL
	,PRIMARY KEY (item_id, ench_name)
);
INSERT INTO item_enchants
	SELECT item_id, ench_name, dam_pct, freq_pct, sv_mod, duration
	FROM item_enchants_bak;
DROP TABLE item_enchants_bak;
CREATE TABLE item_attribs(
	item_id INTEGER REFERENCES items(item_id)
	,attrib_name TEXT REFERENCES attribs(attrib_name)
	,attrib_value INTEGER NOT NULL
	,PRIMARY KEY (item_id, attrib_name)
);
INSERT INTO item_attribs
	SELECT item_id, attrib_name, attrib_value
	FROM item_attribs_bak;
DROP TABLE item_attribs_bak;
CREATE TABLE item_supps(
	item_id INTEGER REFERENCES items(item_id)
	,supp_abbr TEXT REFERENCES supps(supp_abbr)
);
INSERT INTO item_supps
	SELECT item_id, supp_abbr
	FROM item_supps_bak;
DROP TABLE item_supps_bak;
COMMIT;

--Convert to postgresql: http://stackoverflow.com/questions/4581727/convert-sqlite-sql-dump-file-to-postgresql
--echo '.dump' | sqlite3 toril.db | gzip -c >toril.sql.gz
--gunzip toril.sql.gz
--SET CONSTRAINTS ALL DEFERRED; - add right after BEGIN
--:%s/INTEGER/SERIAL/gc - only on boots.boot_id and items.item_id
--psql -d torildb -U kalkinine -W < test.sql
