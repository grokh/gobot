-- dbcopy.sh runs this on its copy of toril.db, with the bot's TorilDB.db
-- attached read-only as k, to refill the copy's items and item_flags from
-- Kegor's eq table. gobot itself doesn't read these tables any more; they're
-- for NyyLIB, the Mudlet package, whose @updatedb downloads the published
-- toril.db. What NyyLIB reads from them:
--   * @id: short_stats, with LIKE.
--   * @statitems, @cid and its container listings: item_name, with LIKE,
--     taking the first match. Items go in shortest name first, so that's
--     the closest one, as with Katumi's stat.
--   * Its weapon buttons: item_type 'ranged' is a bow, and 'weapon' with an
--     item_flags row for 'two_hand' is a two-hander.
--   * Each time it starts: the short_stats of potions, scrolls and poisons,
--     for the "(Potion) Lvl:31 heal *" part (or Scroll, Poison), which
--     labels them in bags. A row with "(Potion) " but not the rest of that
--     shape, or with no short_stats, stops it with a Lua error, so this file
--     fails rather than publish one, and dbcopy.sh keeps the old tables.
--   * A script of its own inserts all 13 items columns by position.

BEGIN;

-- a toril.db made since the item tables left init_db.sql has none
CREATE TABLE IF NOT EXISTS items(
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
CREATE TABLE IF NOT EXISTS item_flags(
	item_id INTEGER REFERENCES items(item_id)
	,flag_abbr TEXT
	,PRIMARY KEY (item_id, flag_abbr)
);

-- the old items' details: nothing fills these now, and NyyLIB doesn't read
-- them
DROP TABLE IF EXISTS item_attribs;
DROP TABLE IF EXISTS item_effects;
DROP TABLE IF EXISTS item_enchants;
DROP TABLE IF EXISTS item_procs;
DROP TABLE IF EXISTS item_resists;
DROP TABLE IF EXISTS item_restricts;
DROP TABLE IF EXISTS item_slots;
DROP TABLE IF EXISTS item_specials;
DROP TABLE IF EXISTS item_supps;
DELETE FROM item_flags;
DELETE FROM items;

-- Kegor's types are the old item_types.item_type, and NyyLIB reads the
-- type_abbr. Types the old table didn't have are 'other'.
CREATE TEMP TABLE type_abbrs(type TEXT PRIMARY KEY, abbr TEXT NOT NULL);
INSERT INTO type_abbrs VALUES
	('ARMOR', 'armor'), ('WORN', 'worn'), ('WEAPON', 'weapon'),
	('FIRE_WEAPON', 'ranged'), ('QUIVER', 'quiver'), ('MISSILE', 'ammo'),
	('CONTAINER', 'container'), ('SPELLBOOK', 'spellbook'),
	('PEN', 'quill'), ('PICK', 'pick'), ('BOAT', 'boat'),
	('LIGHT', 'light'), ('STAFF', 'staff'), ('WAND', 'wand'),
	('SCROLL', 'scroll'), ('POTION', 'potion'), ('KEY', 'key'),
	('POISON', 'poison'), ('LIQUID_CONT', 'drink'), ('FOOD', 'food'),
	('SUMMON', 'summon'), ('INSTRUMENT', 'instrument'),
	('PSP_CRYSTAL', 'crystal'), ('COMPONENT_BAG', 'comp_bag'),
	('SPELL_COMPONENT', 'spell_comp'), ('BOOK', 'book'),
	('SWITCH', 'switch'), ('NOTE', 'note'), ('TREASURE', 'treasure'),
	('TRASH', 'trash'), ('OTHER', 'other'), ('TELEPORT', 'teleport');

-- the old flags, by their words in Kegor's item_flags and worn
CREATE TEMP TABLE flag_abbrs(word TEXT PRIMARY KEY, abbr TEXT NOT NULL);
INSERT INTO flag_abbrs VALUES
	('TWOHANDS', 'two_hand'), ('2H', 'two_hand'), ('LIT', 'lit'),
	('INVISIBLE', 'invis'), ('FLOAT', 'float'),
	('TRANSIENT', 'transient'), ('MAGIC', 'magic'), ('BLESS', 'bless'),
	('SECRET', 'hidden'), ('GLOW', 'glow'), ('DARK', 'dark'),
	('WHOLE-HEAD', 'whole_head'), ('WHOLE-BODY', 'whole_body'),
	('NOSUMMON', 'no_sum'), ('NOSLEEP', 'no_sleep'),
	('NOCHARM', 'no_charm'), ('NOBURN', 'no_burn'),
	('NODROP', 'no_drop'), ('NOLOCATE', 'no_loc'), ('NOSELL', 'no_sell'),
	('NORENT', 'no_rent'), ('NOTAKE', 'no_take');

INSERT INTO items (item_name, keywords, weight, c_value, item_type,
	from_zone, from_mob, short_stats, long_stats, full_stats, comments,
	last_id)
SELECT e.name, coalesce(e.keywords, ''), e.wt, e.value,
	-- Kegor's bows, crossbows and slings are WEAPONs
	CASE WHEN e.type = 'WEAPON' AND (e.w_type LIKE '%bow'
			OR e.w_type LIKE 'sling') THEN 'ranged'
		ELSE coalesce((SELECT abbr FROM type_abbrs t WHERE t.type = e.type),
			'other') END,
	coalesce(e.zone, ''), NULL,
	-- the consumables' part in the old form, which NyyLIB labels them by:
	-- "POTION (Level 31 heal)" becomes "(Potion) Lvl:31 heal", and
	-- "POISON (Level 21 fumble, 5 Apps, 3 Hits)" becomes
	-- "(Poison) Lvl:21 Type:fumble Apps:5 Hits:3"
	coalesce(CASE e.type
		WHEN 'POTION' THEN replace(e.short_stats,
			'POTION (Level ' || e.s_level || ' ' || e.s_spell || ')',
			'(Potion) Lvl:' || e.s_level || ' ' || e.s_spell)
		WHEN 'SCROLL' THEN replace(e.short_stats,
			'SCROLL (Level ' || e.s_level || ' ' || e.s_spell || ')',
			'(Scroll) Lvl:' || e.s_level || ' ' || e.s_spell)
		WHEN 'POISON' THEN replace(e.short_stats,
			'POISON (Level ' || e.p_level || ' ' || e.p_poison || ', ' ||
				e.p_apps || ' Apps' ||
				coalesce(', ' || e.p_hits || ' Hits', '') || ')',
			'(Poison) Lvl:' || e.p_level || ' Type:' || e.p_poison ||
				' Apps:' || e.p_apps || coalesce(' Hits:' || e.p_hits, ''))
		END, e.short_stats, e.name),
	e.long_stats, NULL, NULL, e.date
FROM k.eq e
ORDER BY length(e.name), e.name;

-- the potions, scrolls and poisons NyyLIB can't read a label from: what
-- follows the "(Potion) " must be letters, digits, ':', ' ' or '-', at
-- least one, up to a '*'
CREATE TEMP VIEW unreadable AS
SELECT item_id FROM (
	SELECT item_id, short_stats, tag,
		substr(short_stats, instr(short_stats, tag) + length(tag)) AS rest
	FROM (SELECT item_id, short_stats, CASE item_type
			WHEN 'potion' THEN '(Potion)'
			WHEN 'scroll' THEN '(Scroll)'
			WHEN 'poison' THEN '(Poison)' END AS tag
		FROM items WHERE item_type IN ('potion', 'scroll', 'poison')))
WHERE short_stats IS NULL
	OR (instr(short_stats, tag || ' ') > 0
		AND (instr(rest, '*') < 2
			OR substr(rest, 1, instr(rest, '*') - 1)
				GLOB '*[^A-Za-z0-9: -]*'));

-- where the old form doesn't fit (a spell of '?', say), Kegor's text goes
-- back in, and NyyLIB just doesn't label the item
UPDATE items SET short_stats = coalesce(
	(SELECT e.short_stats FROM k.eq e WHERE e.name = items.item_name),
	item_name)
WHERE item_id IN (SELECT item_id FROM unreadable);

-- two-handed is 2H in worn: WIELD 2H, HOLD 2H
INSERT OR IGNORE INTO item_flags (item_id, flag_abbr)
SELECT i.item_id, f.abbr
FROM k.eq e
JOIN items i ON i.item_name = e.name
JOIN flag_abbrs f ON ' ' || coalesce(e.item_flags, '') || ' ' ||
	coalesce(e.worn, '') || ' ' LIKE '% ' || f.word || ' %';

-- NyyLIB has to start: fail, and leave the old tables, if a row would
-- stop it
CREATE TEMP TABLE nyylib_check(stoppers INTEGER
	CONSTRAINT nyylib_would_not_start CHECK (stoppers = 0));
INSERT INTO nyylib_check SELECT count(*) FROM unreadable;

COMMIT;
