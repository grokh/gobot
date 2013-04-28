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
	attrib_name TEXT PRIMARY KEY
	,attrib_abbr TEXT NOT NULL
	,attrib_display TEXT NOT NULL
);
CREATE TABLE effects(
	effect_name TEXT PRIMARY KEY
	,effect_abbr TEXT NOT NULL
	,effect_display TEXT NOT NULL
);
CREATE TABLE resists(
	resist_name TEXT PRIMARY KEY
	,resist_abbr TEXT NOT NULL
	,resist_display TEXT NOT NULL
);
CREATE TABLE restricts(
	restrict_name TEXT PRIMARY KEY
	,restrict_abbr TEXT NOT NULL
);
CREATE TABLE flags(
	flag_name TEXT PRIMARY KEY
	,flag_abbr TEXT NOT NULL
	,flag_display TEXT NOT NULL
);
CREATE TABLE slots(
	worn_slot TEXT PRIMARY KEY
	,slot_abbr TEXT NOT NULL
	,slot_display TEXT NOT NULL
);
CREATE TABLE item_types(
	item_type TEXT PRIMARY KEY
	,type_abbr TEXT NOT NULL
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
	item_type TEXT REFERENCES item_types(item_type)
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
	,item_type TEXT REFERENCES item_types(item_type) NOT NULL
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
	,worn_slot TEXT REFERENCES slots(worn_slot)
	,PRIMARY KEY (item_id, worn_slot)
);
CREATE TABLE item_flags(
	item_id INTEGER REFERENCES items(item_id)
	,flag_name TEXT REFERENCES flags(flag_name)
	,PRIMARY KEY (item_id, flag_name)
);
CREATE TABLE item_restricts(
	item_id INTEGER REFERENCES items(item_id)
	,restrict_name TEXT REFERENCES restricts(restrict_name)
	,PRIMARY KEY (item_id, restrict_name)
);
CREATE TABLE item_resists(
	item_id INTEGER REFERENCES items(item_id)
	,resist_name TEXT REFERENCES resists(resist_name)
	,resist_value INTEGER NOT NULL
	,PRIMARY KEY (item_id, resist_name)
);
CREATE TABLE item_effects(
	item_id INTEGER REFERENCES items(item_id)
	,effect_name TEXT REFERENCES effects(effect_name)
	,PRIMARY KEY (item_id, effect_name)
);
CREATE TABLE item_specials(
	item_id INTEGER REFERENCES items(item_id)
	,item_type TEXT
	,spec_abbr TEXT
	,spec_value TEXT NOT NULL
	,FOREIGN KEY (item_type, spec_abbr) REFERENCES specials (item_type, spec_abbr)
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
	,attrib_name TEXT REFERENCES attribs(attrib_name)
	,attrib_value INTEGER NOT NULL
	,PRIMARY KEY (item_id, attrib_name)
);
CREATE TABLE item_supps(
	item_id INTEGER REFERENCES items(item_id)
	,supp_abbr TEXT REFERENCES supps(supp_abbr)
);