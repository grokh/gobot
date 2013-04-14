-- create new character tracking tables
CREATE TABLE race_types(
	race_type VARCHAR(20) PRIMARY KEY
	,anti_flag VARCHAR(20) NOT NULL
);
CREATE TABLE races(
	race_name VARCHAR(20) PRIMARY KEY
	,race_abbr VARCHAR(10) NOT NULL
	,anti_flag VARCHAR(20) NOT NULL
	,race_type VARCHAR(20) REFERENCES race_types(race_type) NOT NULL
);
CREATE TABLE class_types(
	class_type VARCHAR(10) PRIMARY KEY
	,anti_flag VARCHAR(20) NOT NULL
);
CREATE TABLE classes(
	class_name VARCHAR(30) PRIMARY KEY
	,class_abbr VARCHAR(3) NOT NULL
	,class_type VARCHAR(10) REFERENCES class_types(class_type) NOT NULL
	,anti_flag VARCHAR(20) NOT NULL
);
CREATE TABLE accounts(
	account_name VARCHAR(30) PRIMARY KEY
	,player_name VARCHAR(30)
);
CREATE TABLE chars(
	account_name VARCHAR(30) REFERENCES accounts(account_name)
	,char_name VARCHAR(30)
	,class_name VARCHAR(30) REFERENCES classes(class_name) NOT NULL
	,char_race VARCHAR(20) REFERENCES races(race_name) NOT NULL
	,char_level INTEGER NOT NULL
	,last_seen TIMESTAMP NOT NULL
	,vis BOOLEAN NOT NULL
	,PRIMARY KEY (account_name, char_name)
);


-- create new boot/load report tables
CREATE TABLE boots(
	boot_id INTEGER PRIMARY KEY
	,boot_time TIMESTAMP NOT NULL
	,uptime VARCHAR(10) NOT NULL
);
CREATE TABLE loads(
	boot_id INTEGER REFERENCES boots(boot_id) NOT NULL
	,report_time TIMESTAMP NOT NULL
	,report_text VARCHAR(320) NOT NULL
	,char_name VARCHAR(30) NOT NULL
	,deleted BOOLEAN NOT NULL
	,PRIMARY KEY (boot_id, report_time)
);


-- create new item stat tables
CREATE TABLE enchants(
	ench_name VARCHAR(25) PRIMARY KEY
	,ench_desc VARCHAR(100) NOT NULL
);
CREATE TABLE attribs(
	attrib_abbr VARCHAR(10) PRIMARY KEY
	,attrib_name VARCHAR(25) NOT NULL
	,attrib_display VARCHAR(25) NOT NULL
);
CREATE TABLE effects(
	effect_abbr VARCHAR(10) PRIMARY KEY
	,effect_name VARCHAR(25) NOT NULL
	,effect_display VARCHAR(25) NOT NULL
);
CREATE TABLE resists(
	resist_abbr VARCHAR(10) PRIMARY KEY
	,resist_name VARCHAR(25) NOT NULL
	,resist_display VARCHAR(25) NOT NULL
);
CREATE TABLE restricts(
	restrict_abbr VARCHAR(10) PRIMARY KEY
	,restrict_name VARCHAR(25) NOT NULL
);
CREATE TABLE flags(
	flag_abbr VARCHAR(10) PRIMARY KEY
	,flag_name VARCHAR(25) NOT NULL
	,flag_display VARCHAR(25) NOT NULL
);
CREATE TABLE slots(
	slot_abbr VARCHAR(10) PRIMARY KEY
	,worn_slot VARCHAR(25)
	,slot_display VARCHAR(25)
);
CREATE TABLE item_types(
	type_abbr VARCHAR(10) PRIMARY KEY
	,item_type VARCHAR(25) NOT NULL
	,type_display VARCHAR(25) NOT NULL
);
CREATE TABLE zones(
	zone_abbr VARCHAR(25) PRIMARY KEY
	,zone_name VARCHAR(150) NOT NULL
);
CREATE TABLE mobs(
	mob_name VARCHAR(150) PRIMARY KEY
	,mob_abbr VARCHAR(25)
	,from_zone VARCHAR(10) REFERENCES zones(zone_abbr)
	,from_quest BOOLEAN
	,has_quest BOOLEAN
	,is_rare BOOLEAN
	,from_invasion BOOLEAN
);
CREATE TABLE specials(
	item_type VARCHAR(10) REFERENCES item_types(type_abbr)
	,spec_abbr VARCHAR(10)
	,spec_display VARCHAR(25) NOT NULL
	,PRIMARY KEY (item_type, spec_abbr)
);
CREATE TABLE items(
	item_id INTEGER PRIMARY KEY
	,item_name VARCHAR(100) NOT NULL
	,keywords VARCHAR(100) NOT NULL
	,weight INTEGER
	,c_value INTEGER
	,item_type VARCHAR(10) REFERENCES item_types(type_abbr) NOT NULL
	,from_zone VARCHAR(25) REFERENCES zones(zone_abbr) NOT NULL
	,from_mob VARCHAR(150) REFERENCES mobs(mob_name)
	,no_identify BOOLEAN
	,is_rare BOOLEAN
	,from_store BOOLEAN
	,from_quest BOOLEAN
	,for_quest BOOLEAN
	,from_invasion BOOLEAN
	,out_of_game BOOLEAN
	,short_stats VARCHAR(450)
	,long_stats VARCHAR(900)
	,full_stats TEXT
	,comments TEXT
	,last_id date
);
CREATE INDEX idx_item_name ON items (item_name);
CREATE VIRTUAL TABLE item_search USING fts3(
	item_id
	,item_text
);
CREATE TABLE item_procs(
	item_id INTEGER REFERENCES items(item_id)
	,proc_name TEXT NOT NULL
	,proc_type VARCHAR(25)
	,proc_desc VARCHAR(25)
	,proc_trig VARCHAR(25)
	,proc_effect VARCHAR(25)
	,PRIMARY KEY (item_id, proc_name)
);
CREATE TABLE item_slots(
	item_id INTEGER REFERENCES items(item_id)
	,slot_abbr VARCHAR(10) REFERENCES slots(slot_abbr)
	,PRIMARY KEY (item_id, slot_abbr)
);
CREATE TABLE item_flags(
	item_id INTEGER REFERENCES items(item_id)
	,flag_abbr VARCHAR(10) REFERENCES flags(flag_abbr)
	,PRIMARY KEY (item_id, flag_abbr)
);
CREATE TABLE item_restricts(
	item_id INTEGER REFERENCES items(item_id)
	,restrict_abbr VARCHAR(10) REFERENCES restricts(restrict_abbr)
	,PRIMARY KEY (item_id, restrict_abbr)
);
CREATE TABLE item_resists(
	item_id INTEGER REFERENCES items(item_id)
	,resist_abbr VARCHAR(10) REFERENCES resists(resist_abbr)
	,resist_value INTEGER NOT NULL
	,PRIMARY KEY (item_id, resist_abbr)
);
CREATE TABLE item_effects(
	item_id INTEGER REFERENCES items(item_id)
	,effect_abbr VARCHAR(10) REFERENCES effects(effect_abbr)
	,PRIMARY KEY (item_id, effect_abbr)
);
CREATE TABLE item_specials(
	item_id INTEGER REFERENCES items(item_id)
	,item_type VARCHAR(10)
	,spec_abbr VARCHAR(10)
	,spec_value VARCHAR(30) NOT NULL
	,FOREIGN KEY (item_type, spec_abbr) REFERENCES specials (item_type, spec_abbr)
	,PRIMARY KEY (item_id, item_type, spec_abbr)
);
CREATE TABLE item_enchants(
	item_id INTEGER REFERENCES items(item_id)
	,ench_name VARCHAR(25) REFERENCES enchants(ench_name)
	,dam_pct INTEGER NOT NULL
	,freq_pct INTEGER NOT NULL
	,sv_mod INTEGER NOT NULL
	,duration INTEGER NOT NULL
	,PRIMARY KEY (item_id, ench_name)
);
CREATE TABLE item_attribs(
	item_id INTEGER REFERENCES items(item_id)
	,attrib_abbr VARCHAR(25) REFERENCES attribs(attrib_abbr)
	,attrib_value INTEGER NOT NULL
	,PRIMARY KEY (item_id, attrib_abbr)
);

CREATE TABLE legacy (
	id INTEGER PRIMARY KEY,
	varName TEXT, varKeywords TEXT, varZone TEXT, varLoad TEXT, varQuest TEXT, varNoID TEXT,
	varType TEXT, varWorn TEXT, varWt TEXT, varHolds TEXT, varValue TEXT, intAC INTEGER,
	varArmor TEXT, varPages TEXT, varHP TEXT, varDice TEXT, varWType TEXT, varWClass TEXT,
	varCRange TEXT, varCBonus TEXT, intHit INTEGER, intDam INTEGER, varSpell TEXT,
	varBreath TEXT, varPara TEXT, varPetri TEXT, varRod TEXT, varStr TEXT, varAgi TEXT,
	varDex TEXT, varCon TEXT, varPow TEXT, varInt TEXT, varWis TEXT, varCha TEXT,
	varMaxstr TEXT, varMaxagi TEXT, varMaxdex TEXT, varMaxcon TEXT, varMaxpow TEXT,
	varMaxint TEXT, varMaxwis TEXT, varMaxcha TEXT, varLuck TEXT, varKarma TEXT, varMana TEXT,
	varMove TEXT, varAge TEXT, varWeight TEXT, varHeight TEXT, varMR TEXT, varSFEle TEXT,
	varSFEnc TEXT, varSFHeal TEXT, varSFIll TEXT, varSFInv TEXT, varSFNature TEXT,
	varSFNec TEXT, varSFProt TEXT, varSFPsi TEXT, varSFSpirit TEXT, varSFSum TEXT,
	varSFTele TEXT, varPsp TEXT, varQuality TEXT, varStutter TEXT, varMin TEXT,
	varPoison TEXT, varLevel TEXT, varApplications TEXT, varCharge TEXT, varMaxcharge TEXT,
	varWlevel TEXT, varWspell TEXT, varRes TEXT, varCRes TEXT, varEnchant TEXT,
	varEffects TEXT, varCrit TEXT, varBonus TEXT, varCeffects TEXT, varUnarmd TEXT,
	varSlash TEXT, varBludgn TEXT, varPierce TEXT, varRange TEXT, varSpells TEXT,
	varSonic TEXT, varPos TEXT, varNeg TEXT, varPsi TEXT, varMental TEXT, varGoods TEXT,
	varEvils TEXT, varLaw TEXT, varChaos TEXT, varForce TEXT, varFire TEXT, varCold TEXT,
	varElect TEXT, varAcid TEXT, varPois TEXT, varAflags TEXT, varIflags TEXT, varDate TEXT
);
