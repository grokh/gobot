package main

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
)

// itemDBFile is TorilDB.db, Kegor's item stats DB. torildb.sh fetches it
// daily into the bot's folder, next to toril.db, and into the web server's.
// Every item lookup reads its eq table: one row per item, keyed by name, with
// a column per stat and Kegor's own short_stats and long_stats. mstat, and
// astat's Loads on line, read its mobs table (mobs.go). Everything else the
// bot keeps (chars, accounts, boots, loads) is in toril.db.
const itemDBFile = "TorilDB.db"

// itemsDown is the reply when TorilDB.db is missing or can't be read.
const itemsDown = "Item stats are unavailable right now. Please try again later."

// OpenItemDB opens TorilDB.db read-only. It checks the file is there first,
// since opening a missing file would create an empty one.
func OpenItemDB() (*sql.DB, error) {
	return openItemDB(itemDBFile)
}

func openItemDB(path string) (*sql.DB, error) {
	if _, err := os.Stat(path); err != nil {
		return nil, err
	}
	return sql.Open("sqlite3", "file:"+path+"?mode=ro")
}

// col quotes a TorilDB.db column name for SQL. Column names can't be ?
// parameters, so only names from the tables below, and mobColumns, ever go
// through here. It uses brackets, not double quotes: SQLite reads a
// double-quoted name that matches no column as a string, so a column Kegor
// renamed would quietly turn "max_agi" > 0 into a test every row passes,
// instead of an error.
func col(name string) string {
	return "[" + name + "]"
}

// hasToken is the SQL test for one of tokens in a space-separated eq column,
// like EARRING in worn or NO-MAGE in item_flags, with the arguments it takes.
func hasToken(column string, tokens ...string) (string, []interface{}) {
	var tests []string
	var args []interface{}
	for _, t := range tokens {
		tests = append(tests,
			"instr(' ' || coalesce("+col(column)+", '') || ' ', ?) > 0")
		args = append(args, " "+t+" ")
	}
	return "(" + strings.Join(tests, " OR ") + ")", args
}

// stat is a number fstat and the TorilEQ form search on. abbr is the name
// players type (fstat maxagi > 0) and the form posts; col is the eq column.
type stat struct{ abbr, display, col string }

var stats = []stat{
	{"ac", "Armor Class", "ac"},
	{"armor", "Armor", "armor"},
	{"hit", "Hitroll", "hit"},
	{"dam", "Damroll", "dam"},
	{"hp", "Hitpoints", "hp"},
	{"mv", "Movement", "move"},
	{"mana", "Mana", "mana"},
	{"svsp", "Save Spell", "sv_spell"},
	{"svbr", "Save Breath", "sv_bre"},
	{"svpar", "Save Paralysis", "sv_para"},
	{"svpet", "Save Petrification", "sv_petri"},
	{"svrod", "Save Rod", "sv_rod"},
	{"maxstr", "Max Strength", "max_str"},
	{"maxagi", "Max Agility", "max_agi"},
	{"maxdex", "Max Dexterity", "max_dex"},
	{"maxcon", "Max Constitution", "max_con"},
	{"maxpow", "Max Power", "max_pow"},
	{"maxint", "Max Intelligence", "max_int"},
	{"maxwis", "Max Wisdom", "max_wis"},
	{"maxcha", "Max Charisma", "max_cha"},
	{"str", "Strength", "str"},
	{"agi", "Agility", "agi"},
	{"dex", "Dexterity", "dex"},
	{"con", "Constitution", "con"},
	{"pow", "Power", "pow"},
	{"int", "Intelligence", "int"},
	{"wis", "Wisdom", "wis"},
	{"cha", "Charisma", "cha"},
	{"MR", "Magic Resistance", "mr"},
	{"age", "Age", "age"},
	{"wt", "Weight", "weight"}, // the WEIGHT affect; the item's own is wt
	{"ht", "Height", "height"},
	{"luck", "Luck", "luck"},
	{"karma", "Karma", "karma"},
	{"sf_elem", "Spell Focus Elemental", "sf_ele"},
	{"sf_ench", "Spell Focus Enchantment", "sf_enc"},
	{"sf_heal", "Spell Focus Healing", "sf_heal"},
	{"sf_illu", "Spell Focus Illusion", "sf_ill"},
	{"sf_invo", "Spell Focus Invocation", "sf_inv"},
	{"sf_nat", "Spell Focus Nature", "sf_nat"},
	{"sf_nec", "Spell Focus Necromancy", "sf_nec"},
	{"sf_prot", "Spell Focus Protection", "sf_prot"},
	{"sf_spir", "Spell Focus Spirit", "sf_spi"},
	{"sf_sum", "Spell Focus Summoning", "sf_sum"},
}

// resist is a resistance, by the name players and the form use, with the eq
// column holding its percent.
type resist struct{ abbr, display, col string }

var resists = []resist{
	{"fire", "Fire", "r_fire"},
	{"cold", "Cold", "r_cold"},
	{"acid", "Acid", "r_acid"},
	{"pois", "Poison", "r_poison"},
	{"elect", "Electricity", "r_elect"},
	{"spells", "Spells", "r_spell"},
	{"rang", "Ranged", "r_ranged"},
	{"pier", "Pierce", "r_pierce"},
	{"blud", "Bludgeon", "r_bludgn"},
	{"slas", "Slash", "r_slash"},
	{"unarm", "Unarmed", "r_unarmd"},
	{"force", "Force", "r_force"},
	{"chaos", "Chaos", "r_chaos"},
	{"law", "Law", "r_law"},
	{"evil", "Evil", "r_evil"},
	{"good", "Good", "r_good"},
	{"mental", "Mental", "r_mental"},
	{"psion", "Psionic", "r_psi"},
	{"neg", "Negative", "r_neg"},
	{"pos", "Positive", "r_pos"},
	{"sonic", "Sonic", "r_sonic"},
}

// slot is a place an item is worn, by the name players and the form use,
// with the tokens in eq's worn column that mean it. Kegor writes whole-head
// and whole-body items as WHOLE-HEAD and WHOLE-BODY in place of HEAD and BODY.
type slot struct {
	abbr, display string
	worn          []string
}

var slots = []slot{
	{"head", "Head", []string{"HEAD", "WHOLE-HEAD"}},
	{"eyes", "Eyes", []string{"EYES"}},
	{"ear", "Ear", []string{"EARRING"}},
	{"face", "Face", []string{"FACE"}},
	{"neck", "Neck", []string{"NECK"}},
	{"on_body", "On Body", []string{"BODY", "WHOLE-BODY"}},
	{"about", "About Body", []string{"ABOUT"}},
	{"waist", "About Waist", []string{"WAIST"}},
	{"arms", "Arms", []string{"ARMS"}},
	{"wrist", "Wrist", []string{"WRIST"}},
	{"hands", "Hands", []string{"HANDS"}},
	{"finger", "Finger", []string{"FINGER"}},
	{"wield", "Wielded", []string{"WIELD"}},
	{"shield", "Shield", []string{"SHIELD"}},
	{"held", "Held", []string{"HOLD"}},
	{"legs", "Legs", []string{"LEGS"}},
	{"feet", "Feet", []string{"FEET"}},
	{"no_wear", "Can't Wear", []string{"NOBITS"}},
	{"tail", "Tail", []string{"TAIL"}},
	{"quiver", "Quiver", []string{"QUIVER"}},
	{"badge", "Badge", []string{"INSIGNIA"}},
	{"component", "Component Bag", []string{"COMPONENT_BAG"}},
	{"learn", "Can't Wear", []string{"LEARN"}},
	{"light", "Can't Wear", []string{"LIGHT"}},
	{"throw", "Can't Wear", []string{"THROW"}},
}

// effect is an ability an item gives, with its token in eq's affect_flags.
type effect struct{ abbr, display, token string }

var effects = []effect{
	{"pr_evil", "Protection From Evil", "PR-EVIL"},
	{"pr_good", "Protection From Good", "PR-GOOD"},
	{"dm", "Detect Magic", "DET-MAGIC"},
	{"di", "Detect Invisibility", "DET-INVIS"},
	{"det_evil", "Detect Evil", "DET-EVIL"},
	{"det_good", "Detect Good", "DET-GOOD"},
	{"infra", "Infravision", "INFRA"},
	{"sense", "Sense Life", "SENSE-LIFE"},
	{"fly", "Fly", "FLY"},
	{"lev", "Levitate", "LEVITATE"},
	{"farsee", "Farsee", "FARSEE"},
	{"haste", "Haste", "HASTE"},
	{"sneak", "Sneak", "SNEAK"},
	{"wb", "Water Breathing", "WATERBREATH"},
	{"aware", "Awareness", "AWARE"},
	{"min_globe", "Minor Globe", "MIN-GLOBE"},
	{"mis_shield", "Missile Shield", "MSL-SHLD"},
	{"blind", "Blindness", "BLINDNESS"},
	{"slow", "Slowness", "SLOW"},
	{"ultra", "Ultravision", "ULTRA"},
	{"slow_poi", "Slow Poison", "SLOW-POISON"},
	{"no_charm", "No Charm", "NOCHARM"},
	{"no_sleep", "No Sleep", "NOSLEEP"},
	{"no_sum", "No Summon", "NOSUMMON"},
}

// itemFlag is an item flag the form searches on, by its token in eq's
// item_flags, or in worn for the ones Kegor keeps with the slots: two-handed
// items are WIELD 2H there, and whole-head and whole-body take the slot's place.
type itemFlag struct {
	abbr, display string
	flags, worn   []string
}

var flags = []itemFlag{
	{"two_hand", "Two Handed", []string{"TWOHANDS"}, []string{"2H"}},
	{"lit", "Lit", []string{"LIT"}, nil},
	{"invis", "Invisible", []string{"INVISIBLE"}, nil},
	{"float", "Float", []string{"FLOAT"}, nil},
	{"transient", "Transient", []string{"TRANSIENT"}, nil},
	{"magic", "Magic", []string{"MAGIC"}, nil},
	{"bless", "Bless", []string{"BLESS"}, nil},
	{"hidden", "Hidden", []string{"SECRET"}, nil},
	{"glow", "Glowing", []string{"GLOW"}, nil},
	{"dark", "Dark", []string{"DARK"}, nil},
	{"whole_head", "Whole Head", []string{"WHOLE-HEAD"}, []string{"WHOLE-HEAD"}},
	{"whole_body", "Whole Body", []string{"WHOLE-BODY"}, []string{"WHOLE-BODY"}},
	{"no_burn", "No Burn", []string{"NOBURN"}, nil},
	{"no_drop", "No Drop", []string{"NODROP"}, nil},
	{"no_loc", "No Locate", []string{"NOLOCATE"}, nil},
	{"no_sell", "No Sell", []string{"NOSELL"}, nil},
	{"no_rent", "No Rent", []string{"NORENT"}, nil},
	{"no_take", "No Take", []string{"NOTAKE"}, nil},
}

// match is the SQL test for the flag, with its arguments.
func (f itemFlag) match() (string, []interface{}) {
	test, args := hasToken("item_flags", f.flags...)
	if len(f.worn) > 0 {
		w, wargs := hasToken("worn", f.worn...)
		test = "(" + test + " OR " + w + ")"
		args = append(args, wargs...)
	}
	return test, args
}

// supp is one of the form's flags for where an item comes from, with the eq
// test for it: Kegor's load code (R rare, S store, Q quest reward, I
// invasion, ! out of game), or the X he puts in quest and no_id. Most other
// items load N. He uses C, K and X to keep junk out of zone stats: C common
// (loads on several mobs or in several zones, usually zone Common), K key
// (one that isn't rare, for a quest, or worn in an odd slot), X no take.
type supp struct{ abbr, display, test string }

var supps = []supp{
	{"no_identify", "No Identify", "no_id = 'X'"},
	{"is_rare", "Is Rare", "load = 'R'"},
	{"from_store", "From Store", "load = 'S'"},
	{"from_quest", "From Quest", "load = 'Q'"},
	{"for_quest", "For Quest", "quest = 'X'"},
	{"from_invasion", "From Invasion", "load = 'I'"},
	{"out_of_game", "Out Of Game", "load = '!'"},
}

// usableBy maps the form's Usable By boxes to the restriction in item_flags
// that rules an item out.
var usableBy = []struct{ abbr, restrict string }{
	{"!fighter", "NO-WARRIOR"},
	{"!priest", "NO-CLERIC"},
	{"!thief", "NO-THIEF"},
	{"!mage", "NO-MAGE"},
	{"!male", "ANTI-MALE"},
	{"!female", "ANTI-FEMALE"},
	{"!good", "ANTI-GOOD"},
	{"!neut", "ANTI-NEUTRAL"},
	{"!evil", "ANTI-EVIL"},
	{"!goodrace", "ANTI-GOODRACE"},
	{"!evilrace", "ANTI-EVILRACE"},
}

// typeNames are display names for item types in eq's type column whose
// title-cased name wouldn't say what they are.
var typeNames = map[string]string{
	"FIRE_WEAPON": "Ranged Weapon",
	"MISSILE":     "Ranged Ammo",
	"PEN":         "Quill",
	"PICK":        "Lockpicks",
	"LIQUID_CONT": "Liquid Container",
	"PSP_CRYSTAL": "PSP Crystal",
}

// typeName is the display name for an item type: COMPONENT_BAG is Component
// Bag, so new types Kegor adds show up readably without a code change.
func typeName(t string) string {
	if n, ok := typeNames[t]; ok {
		return n
	}
	words := strings.Fields(strings.ReplaceAll(strings.ToLower(t), "_", " "))
	for n, w := range words {
		words[n] = strings.ToUpper(w[:1]) + w[1:]
	}
	return strings.Join(words, " ")
}

func findStat(abbr string) (stat, bool) {
	for _, s := range stats {
		if strings.EqualFold(s.abbr, abbr) {
			return s, true
		}
	}
	return stat{}, false
}

func findResist(abbr string) (resist, bool) {
	for _, r := range resists {
		if strings.EqualFold(r.abbr, abbr) {
			return r, true
		}
	}
	return resist{}, false
}

func findSlot(abbr string) (slot, bool) {
	for _, s := range slots {
		if strings.EqualFold(s.abbr, abbr) {
			return s, true
		}
	}
	return slot{}, false
}

// zoneSep separates the zones of an item that loads in more than one, as in
// Silverymoon - Westgate. Zone names themselves only use bare hyphens.
const zoneSep = " - "

// zoneNames splits the zone column's values into single zones, sorted
// without regard to case.
func zoneNames(values []string) []string {
	seen := make(map[string]bool)
	var names []string
	for _, v := range values {
		for _, z := range strings.Split(v, zoneSep) {
			if z = strings.TrimSpace(z); z != "" && !seen[z] {
				seen[z] = true
				names = append(names, z)
			}
		}
	}
	sort.Slice(names, func(a, b int) bool {
		return strings.ToLower(names[a]) < strings.ToLower(names[b])
	})
	return names
}

// itemColumns are the eq columns gobot reads, which CheckItemDB makes sure a
// new TorilDB.db still has.
func itemColumns() []string {
	cols := []string{"name", "short_stats", "long_stats", "date", "zone",
		"type", "worn", "item_flags", "affect_flags", "load", "quest", "no_id"}
	for _, s := range stats {
		cols = append(cols, s.col)
	}
	for _, r := range resists {
		cols = append(cols, r.col)
	}
	return cols
}

// minShare is how much of the TorilDB.db in use a new copy must hold, by
// item count, to replace it: a cut-off export shouldn't.
const minShare = 0.9

// hasColumns checks that table has every one of columns, by reading them.
func hasColumns(db *sql.DB, table string, columns []string) error {
	var cols []string
	for _, c := range columns {
		cols = append(cols, col(c))
	}
	rows, err := db.Query("SELECT " + strings.Join(cols, ", ") + " FROM " +
		table + " LIMIT 1")
	if err != nil {
		return err
	}
	return rows.Close()
}

// CheckItemDB checks that the file at path is a TorilDB.db gobot can use:
// whole, with eq and mobs tables holding every column gobot reads, and not
// much smaller than the TorilDB.db in use here, if there is one, by item
// count. torildb.sh runs it on each download before putting the download
// in place.
func CheckItemDB(path string) (string, error) {
	db, err := openItemDB(path)
	if err != nil {
		return "", err
	}
	defer db.Close()

	var check string
	if err := db.QueryRow("PRAGMA integrity_check").Scan(&check); err != nil {
		return "", err
	}
	if check != "ok" {
		return "", fmt.Errorf("integrity check: %s", check)
	}
	if err := hasColumns(db, "eq", itemColumns()); err != nil {
		return "", err
	}
	if err := hasColumns(db, "mobs", mobColumns); err != nil {
		return "", err
	}

	var n, mobs int
	var newest sql.NullString
	err = db.QueryRow("SELECT COUNT(*), MAX(date) FROM eq").Scan(&n, &newest)
	if err != nil {
		return "", err
	}
	if err := db.QueryRow("SELECT COUNT(*) FROM mobs").Scan(&mobs); err != nil {
		return "", err
	}
	if n == 0 {
		return "", errors.New("no items")
	}
	if cur, err := OpenItemDB(); err == nil {
		defer cur.Close()
		var have int
		if cur.QueryRow("SELECT COUNT(*) FROM eq").Scan(&have) == nil &&
			float64(n) < minShare*float64(have) {
			return "", fmt.Errorf("%d items, but the %s in use has %d",
				n, itemDBFile, have)
		}
	}
	return fmt.Sprintf("%s ok: %d items, %d mobs, newest %s", path, n, mobs,
		newest.String), nil
}
