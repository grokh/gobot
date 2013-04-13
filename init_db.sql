-- create new character tracking tables
CREATE TABLE race_types(
	race_type varchar(20) PRIMARY KEY
	,anti_flag varchar(20)
);
CREATE TABLE races(
	race_name varchar(20) PRIMARY KEY
	,race_abbr varchar(10)
	,anti_flag varchar(20)
	,race_type varchar(20) REFERENCES race_types(race_type)
);
CREATE TABLE class_types(
	class_type varchar(10) PRIMARY KEY
	,anti_flag varchar(20)
);
CREATE TABLE classes(
	class_name varchar(30) PRIMARY KEY
	,class_abbr varchar(3)
	,class_type varchar(10) REFERENCES class_types(class_type)
	,anti_flag varchar(20)
);
CREATE TABLE accounts(
	account_name varchar(30) PRIMARY KEY
	,player_name varchar(30)
);
CREATE TABLE chars(
	account_name varchar(30) REFERENCES accounts(account_name)
	,char_name varchar(30)
	,class_name varchar(30) REFERENCES classes(class_name)
	,char_race varchar(20) REFERENCES races(race_name)
	,char_level integer
	,last_seen timestamp
	,vis boolean
	,PRIMARY KEY (account_name, char_name)
);


-- create new boot/load report tables
CREATE TABLE boots(
	boot_id integer PRIMARY KEY
	,boot_time timestamp NOT NULL
	,uptime varchar(10) NOT NULL
);
CREATE TABLE loads(
	boot_id integer REFERENCES boots(boot_id) NOT NULL
	,report_time timestamp NOT NULL
	,report_text varchar(320) NOT NULL
	,char_name varchar(30) NOT NULL
	,deleted boolean NOT NULL
	,PRIMARY KEY (boot_id, report_time)
);


-- create new item stat tables
CREATE TABLE enchants(
	ench_name varchar(25) PRIMARY KEY
	,ench_desc varchar(100)
);
CREATE TABLE attribs(
	attrib_abbr varchar(10) PRIMARY KEY
	,attrib_name varchar(25)
	,attrib_display varchar(25)
);
CREATE TABLE effects(
	effect_abbr varchar(10) PRIMARY KEY
	,effect_name varchar(25)
	,effect_display varchar(25)
);
CREATE TABLE resists(
	resist_abbr varchar(10) PRIMARY KEY
	,resist_name varchar(25)
	,resist_display varchar(25)
);
CREATE TABLE restricts(
	restrict_abbr varchar(10) PRIMARY KEY
	,restrict_name varchar(25)
);
CREATE TABLE flags(
	flag_abbr varchar(10) PRIMARY KEY
	,flag_name varchar(25)
	,flag_display varchar(25)
);
CREATE TABLE slots(
	slot_abbr varchar(10) PRIMARY KEY
	,worn_slot varchar(25)
	,slot_display varchar(25)
);
CREATE TABLE item_types(
	type_abbr varchar(10) PRIMARY KEY
	,item_type varchar(25)
	,type_display varchar(25)
);
CREATE TABLE zones(
	zone_abbr varchar(25) PRIMARY KEY
	,zone_name varchar(150)
);
CREATE TABLE mobs(
	mob_name varchar(150) PRIMARY KEY
	,mob_abbr varchar(25)
	,from_zone varchar(10) REFERENCES zones(zone_abbr)
	,from_quest boolean
	,has_quest boolean
	,is_rare boolean
	,from_invasion boolean
);
CREATE TABLE specials(
	item_type varchar(10) REFERENCES item_types(type_abbr)
	,spec_abbr varchar(10)
	,spec_display varchar(25)
	,PRIMARY KEY (item_type, spec_abbr)
);
CREATE TABLE items(
	item_id integer PRIMARY KEY
	,item_name varchar(100) NOT NULL
	,keywords varchar(100)
	,weight integer
	,c_value integer
	,item_type varchar(10) REFERENCES item_types(type_abbr)
	,from_zone varchar(25) REFERENCES zones(zone_abbr)
	,from_mob varchar(150) REFERENCES mobs(mob_name)
	,no_identify boolean
	,is_rare boolean
	,from_store boolean
	,from_quest boolean
	,for_quest boolean
	,from_invasion boolean
	,out_of_game boolean
	,short_stats varchar(450)
	,long_stats varchar(900)
	,full_stats text
	,comments text
	,last_id date
	,tsv text
);
CREATE INDEX idx_item_name ON items (item_name);
CREATE TABLE item_procs(
	item_id integer REFERENCES items(item_id)
	,proc_name text
	,proc_type varchar(25)
	,proc_desc varchar(25)
	,proc_trig varchar(25)
	,proc_effect varchar(25)
	,PRIMARY KEY (item_id, proc_name)
);
CREATE TABLE item_slots(
	item_id integer REFERENCES items(item_id)
	,slot_abbr varchar(10) REFERENCES slots(slot_abbr)
	,PRIMARY KEY (item_id, slot_abbr)
);
CREATE TABLE item_flags(
	item_id integer REFERENCES items(item_id)
	,flag_abbr varchar(10) REFERENCES flags(flag_abbr)
	,PRIMARY KEY (item_id, flag_abbr)
);
CREATE TABLE item_restricts(
	item_id integer REFERENCES items(item_id)
	,restrict_abbr varchar(10) REFERENCES restricts(restrict_abbr)
	,PRIMARY KEY (item_id, restrict_abbr)
);
CREATE TABLE item_resists(
	item_id integer REFERENCES items(item_id)
	,resist_abbr varchar(10) REFERENCES resists(resist_abbr)
	,resist_value integer
	,PRIMARY KEY (item_id, resist_abbr)
);
CREATE TABLE item_effects(
	item_id integer REFERENCES items(item_id)
	,effect_abbr varchar(10) REFERENCES effects(effect_abbr)
	,PRIMARY KEY (item_id, effect_abbr)
);
CREATE TABLE item_specials(
	item_id integer REFERENCES items(item_id)
	,item_type varchar(10)
	,spec_abbr varchar(10)
	,spec_value varchar(30)
	,FOREIGN KEY (item_type, spec_abbr) REFERENCES specials (item_type, spec_abbr)
	,PRIMARY KEY (item_id, item_type, spec_abbr)
);
CREATE TABLE item_enchants(
	item_id integer REFERENCES items(item_id)
	,ench_name varchar(25) REFERENCES enchants(ench_name)
	,dam_pct integer
	,freq_pct integer
	,sv_mod integer
	,duration integer
	,PRIMARY KEY (item_id, ench_name)
);
CREATE TABLE item_attribs(
	item_id integer REFERENCES items(item_id)
	,attrib_abbr varchar(25) REFERENCES attribs(attrib_abbr)
	,attrib_value integer
	,PRIMARY KEY (item_id, attrib_abbr)
);

CREATE TABLE legacy (
	id integer PRIMARY KEY,
	varName text, varKeywords text, varZone text, varLoad text, varQuest text, varNoID text,
	varType text, varWorn text, varWt text, varHolds text, varValue text, intAC integer,
	varArmor text, varPages text, varHP text, varDice text, varWType text, varWClass text,
	varCRange text, varCBonus text, intHit integer, intDam integer, varSpell text,
	varBreath text, varPara text, varPetri text, varRod text, varStr text, varAgi text,
	varDex text, varCon text, varPow text, varInt text, varWis text, varCha text,
	varMaxstr text, varMaxagi text, varMaxdex text, varMaxcon text, varMaxpow text,
	varMaxint text, varMaxwis text, varMaxcha text, varLuck text, varKarma text, varMana text,
	varMove text, varAge text, varWeight text, varHeight text, varMR text, varSFEle text,
	varSFEnc text, varSFHeal text, varSFIll text, varSFInv text, varSFNature text,
	varSFNec text, varSFProt text, varSFPsi text, varSFSpirit text, varSFSum text,
	varSFTele text, varPsp text, varQuality text, varStutter text, varMin text,
	varPoison text, varLevel text, varApplications text, varCharge text, varMaxcharge text,
	varWlevel text, varWspell text, varRes text, varCRes text, varEnchant text,
	varEffects text, varCrit text, varBonus text, varCeffects text, varUnarmd text,
	varSlash text, varBludgn text, varPierce text, varRange text, varSpells text,
	varSonic text, varPos text, varNeg text, varPsi text, varMental text, varGoods text,
	varEvils text, varLaw text, varChaos text, varForce text, varFire text, varCold text,
	varElect text, varAcid text, varPois text, varAflags text, varIflags text, varDate text
);
