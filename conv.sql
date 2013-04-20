PRAGMA foreign_keys = ON;
CREATE TABLE supps(
	supp_abbr VARCHAR(10) PRIMARY KEY
	,supp_display VARCHAR(25) NOT NULL
	,supp_value VARCHAR(25) NOT NULL
);
CREATE TABLE item_supps(
	item_id INTEGER REFERENCES items(item_id)
	,supp_abbr VARCHAR(10) REFERENCES supps(supp_abbr)
	,PRIMARY KEY(item_id, supp_abbr)
);
INSERT INTO supps VALUES('no_identify', 'No Identify', 'NoID');
INSERT INTO supps VALUES('is_rare', 'Is Rare', 'R');
INSERT INTO supps VALUES('from_store', 'From Store', 'S');
INSERT INTO supps VALUES('from_quest', 'From Quest', 'Q');
INSERT INTO supps VALUES('for_quest', 'For Quest', 'U');
INSERT INTO supps VALUES('from_invasion', 'From Invasion', 'I');
INSERT INTO supps VALUES('out_of_game', 'Out Of Game', 'O');

INSERT INTO item_supps (item_id, supp_abbr)
SELECT item_id, 'no_identify'
FROM items WHERE no_identify = 't';
INSERT INTO item_supps (item_id, supp_abbr)
SELECT item_id, 'is_rare'
FROM items WHERE is_rare = 't';
INSERT INTO item_supps (item_id, supp_abbr)
SELECT item_id, 'from_store'
FROM items WHERE from_store = 't';
INSERT INTO item_supps (item_id, supp_abbr)
SELECT item_id, 'from_quest'
FROM items WHERE from_quest = 't';
INSERT INTO item_supps (item_id, supp_abbr)
SELECT item_id, 'for_quest'
FROM items WHERE for_quest = 't';
INSERT INTO item_supps (item_id, supp_abbr)
SELECT item_id, 'from_invasion'
FROM items WHERE from_invasion = 't';
INSERT INTO item_supps (item_id, supp_abbr)
SELECT item_id, 'out_of_game'
FROM items WHERE out_of_game = 't';

PRAGMA foreign_keys = OFF;
BEGIN TRANSACTION;
CREATE TEMPORARY TABLE items_backup(
	item_id INTEGER PRIMARY KEY
	,item_name VARCHAR(100) NOT NULL
	,keywords VARCHAR(100) NOT NULL
	,weight INTEGER
	,c_value INTEGER
	,item_type VARCHAR(10) REFERENCES item_types(type_abbr) NOT NULL
	,from_zone VARCHAR(25) REFERENCES zones(zone_abbr) NOT NULL
	,from_mob VARCHAR(150) REFERENCES mobs(mob_name)
	,short_stats VARCHAR(500)
	,long_stats VARCHAR(800)
	,full_stats TEXT
	,comments TEXT
	,last_id DATE
);
INSERT INTO items_backup
	SELECT item_id, item_name, keywords, weight, c_value, item_type,
	from_zone, from_mob, short_stats, long_stats, full_stats,
	comments, last_id
	FROM items;
DROP TABLE items;
CREATE TABLE items(
	item_id INTEGER PRIMARY KEY
	,item_name VARCHAR(100) NOT NULL
	,keywords VARCHAR(100) NOT NULL
	,weight INTEGER
	,c_value INTEGER
	,item_type VARCHAR(10) REFERENCES item_types(type_abbr) NOT NULL
	,from_zone VARCHAR(25) REFERENCES zones(zone_abbr) NOT NULL
	,from_mob VARCHAR(150) REFERENCES mobs(mob_name)
	,short_stats VARCHAR(500)
	,long_stats VARCHAR(800)
	,full_stats TEXT
	,comments TEXT
	,last_id DATE
);
INSERT INTO items
	SELECT item_id, item_name, keywords, weight, c_value, item_type,
	from_zone, from_mob, short_stats, long_stats, full_stats,
	comments, last_id
	FROM items_backup;
DROP TABLE items_backup;
COMMIT;

--UPDATE items SET c_value = -1 WHERE c_value IS NULL;
--UPDATE items SET weight = -1 WHERE weight IS NULL;
