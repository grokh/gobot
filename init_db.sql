-- create new character tracking tables
CREATE TABLE race_types(
	race_type TEXT PRIMARY KEY
	,anti_flag TEXT NOT NULL
);
CREATE TABLE races(
	race_name TEXT PRIMARY KEY
	,race_abbr TEXT NOT NULL
	,anti_flag TEXT NOT NULL
	,race_type TEXT REFERENCES race_types(race_type) NOT NULL
);
CREATE TABLE class_types(
	class_type TEXT PRIMARY KEY
	,anti_flag TEXT NOT NULL
);
CREATE TABLE classes(
	class_name TEXT PRIMARY KEY
	,class_abbr TEXT NOT NULL
	,class_type TEXT REFERENCES class_types(class_type) NOT NULL
	,anti_flag TEXT NOT NULL
);
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


-- create new item stat tables
CREATE TABLE enchants(
	ench_name TEXT PRIMARY KEY
	,ench_desc TEXT NOT NULL
);
CREATE TABLE attribs(
	attrib_abbr TEXT PRIMARY KEY
	,attrib_name TEXT NOT NULL
	,attrib_display TEXT NOT NULL
);
CREATE TABLE effects(
	effect_abbr TEXT PRIMARY KEY
	,effect_name TEXT NOT NULL
	,effect_display TEXT NOT NULL
);
CREATE TABLE resists(
	resist_abbr TEXT PRIMARY KEY
	,resist_name TEXT NOT NULL
	,resist_display TEXT NOT NULL
);
CREATE TABLE restricts(
	restrict_abbr TEXT PRIMARY KEY
	,restrict_name TEXT NOT NULL
);
CREATE TABLE flags(
	flag_abbr TEXT PRIMARY KEY
	,flag_name TEXT NOT NULL
	,flag_display TEXT NOT NULL
);
CREATE TABLE slots(
	slot_abbr TEXT PRIMARY KEY
	,worn_slot TEXT NOT NULL
	,slot_display TEXT NOT NULL
);
CREATE TABLE item_types(
	type_abbr TEXT PRIMARY KEY
	,item_type TEXT NOT NULL
	,type_display TEXT NOT NULL
);
CREATE TABLE zones(
	zone_abbr TEXT PRIMARY KEY
	,zone_name TEXT NOT NULL
);
CREATE TABLE mobs(
	mob_name TEXT PRIMARY KEY
	,mob_abbr TEXT
	,from_zone TEXT REFERENCES zones(zone_abbr)
);
CREATE TABLE specials(
	item_type TEXT REFERENCES item_types(type_abbr)
	,spec_abbr TEXT NOT NULL
	,spec_display TEXT NOT NULL
	,PRIMARY KEY (item_type, spec_abbr)
);
CREATE TABLE supps(
	supp_abbr TEXT PRIMARY KEY
	,supp_display TEXT NOT NULL
	,supp_value TEXT NOT NULL
);
CREATE TABLE items(
	item_id INTEGER PRIMARY KEY
	,item_name TEXT NOT NULL
	,keywords TEXT NOT NULL
	,weight INTEGER
	,c_value INTEGER
	,item_type TEXT REFERENCES item_types(type_abbr) NOT NULL
	,from_zone TEXT REFERENCES zones(zone_abbr) NOT NULL
	,from_mob TEXT REFERENCES mobs(mob_name)
	,short_stats TEXT
	,long_stats TEXT
	,full_stats TEXT
	,comments TEXT
	,last_id DATE
);
CREATE INDEX idx_item_name ON items (item_name);
CREATE TABLE item_procs(
	item_id INTEGER REFERENCES items(item_id)
	,proc_name TEXT NOT NULL
	,proc_type TEXT
	,proc_desc TEXT
	,proc_trig TEXT
	,proc_effect TEXT
	,PRIMARY KEY (item_id, proc_name)
);
CREATE TABLE item_slots(
	item_id INTEGER REFERENCES items(item_id)
	,slot_abbr TEXT REFERENCES slots(slot_abbr)
	,PRIMARY KEY (item_id, slot_abbr)
);
CREATE TABLE item_flags(
	item_id INTEGER REFERENCES items(item_id)
	,flag_abbr TEXT REFERENCES flags(flag_abbr)
	,PRIMARY KEY (item_id, flag_abbr)
);
CREATE TABLE item_restricts(
	item_id INTEGER REFERENCES items(item_id)
	,restrict_abbr TEXT REFERENCES restricts(restrict_abbr)
	,PRIMARY KEY (item_id, restrict_abbr)
);
CREATE TABLE item_resists(
	item_id INTEGER REFERENCES items(item_id)
	,resist_abbr TEXT REFERENCES resists(resist_abbr)
	,resist_value INTEGER NOT NULL
	,PRIMARY KEY (item_id, resist_abbr)
);
CREATE TABLE item_effects(
	item_id INTEGER REFERENCES items(item_id)
	,effect_abbr TEXT REFERENCES effects(effect_abbr)
	,PRIMARY KEY (item_id, effect_abbr)
);
CREATE TABLE item_specials(
	item_id INTEGER REFERENCES items(item_id)
	,item_type TEXT
	,spec_abbr TEXT
	,spec_value TEXT NOT NULL
	,FOREIGN KEY (item_type, spec_abbr) REFERENCES specials (type_abbr, spec_abbr)
	,PRIMARY KEY (item_id, item_type, spec_abbr)
);
CREATE TABLE item_enchants(
	item_id INTEGER REFERENCES items(item_id)
	,ench_name TEXT REFERENCES enchants(ench_name)
	,dam_pct INTEGER NOT NULL
	,freq_pct INTEGER NOT NULL
	,sv_mod INTEGER NOT NULL
	,duration INTEGER NOT NULL
	,PRIMARY KEY (item_id, ench_name)
);
CREATE TABLE item_attribs(
	item_id INTEGER REFERENCES items(item_id)
	,attrib_abbr TEXT REFERENCES attribs(attrib_abbr)
	,attrib_value INTEGER NOT NULL
	,PRIMARY KEY (item_id, attrib_abbr)
);
CREATE TABLE item_supps(
	item_id INTEGER REFERENCES items(item_id)
	,supp_abbr TEXT REFERENCES supps(supp_abbr)
);