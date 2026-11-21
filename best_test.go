package main

import (
	"database/sql"
	"maps"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// bestItem is a test item for a fill: its name, its worn and item_flags,
// its load code, what it weighs, and its stats, by eq column.
func bestItem(name, worn, flags, load string, wt int, stats dbRow) dbRow {
	row := dbRow{"name": name, "zone": "Testville", "load": load,
		"type": "WORN", "worn": worn, "wt": wt, "affect_flags": "NOBITS",
		"item_flags": flags, "date": "2024-01-01",
		"short_stats": name + " (" + worn + ")", "long_stats": name}
	maps.Copy(row, stats)
	return row
}

// bestEq are items testBuild adds for fills, each kind of check with a stat
// of its own, which no other test item has.
var bestEq = []dbRow{
	// luck: a whole-body robe, and a whole-head circlet, against what they
	// cover
	bestItem("a quilted robe", "WHOLE-BODY", "NOBITS", "N", 4, dbRow{"luck": 5}),
	bestItem("a padded jerkin", "BODY", "NOBITS", "N", 3, dbRow{"luck": 2}),
	bestItem("some padded sleeves", "ARMS", "NOBITS", "N", 2, dbRow{"luck": 2}),
	bestItem("some padded breeches", "LEGS", "NOBITS", "N", 2, dbRow{"luck": 2}),
	bestItem("a lucky circlet", "WHOLE-HEAD", "NOBITS", "N", 1, dbRow{"luck": 4}),
	bestItem("a leather cap", "HEAD", "NOBITS", "N", 1, dbRow{"luck": 1}),
	bestItem("a bone mask", "FACE", "NOBITS", "N", 1, dbRow{"luck": 1}),
	// sf_enc: a whole-head hood that ties with what it covers
	bestItem("a motley hood", "WHOLE-HEAD", "NOBITS", "N", 2, dbRow{"sf_enc": 3}),
	bestItem("a motley coif", "HEAD", "NOBITS", "N", 1, dbRow{"sf_enc": 2}),
	bestItem("a pair of motley lenses", "EYES", "NOBITS", "N", 1,
		dbRow{"sf_enc": 1}),
	// int: the hands
	bestItem("a willow wand", "WIELD", "NOBITS", "N", 1, dbRow{"int": 3}),
	bestItem("an oak cudgel", "WIELD 2H", "NOBITS", "N", 8, dbRow{"int": 5}),
	bestItem("a crystal orb", "HOLD", "NOBITS", "N", 1, dbRow{"int": 2}),
	bestItem("a scrying bowl", "HOLD 2H", "NOBITS", "N", 3, dbRow{"int": 4}),
	bestItem("a warded buckler", "SHIELD", "NOBITS", "N", 5, dbRow{"int": 1}),
	// cha: a class's item, a sex's, and one barred to good characters and
	// good races
	bestItem("a cleric's pendant", "NECK", "NO-WARRIOR NO-THIEF NO-MAGE", "N",
		1, dbRow{"cha": 4}),
	bestItem("a dancer's anklet", "FEET", "ANTI-MALE", "N", 0, dbRow{"cha": 3}),
	bestItem("a sash of the evil eye", "WAIST", "ANTI-GOOD ANTI-GOODRACE",
		"N", 1, dbRow{"cha": 2}),
	// con: kinds of item to leave out, and items no player can have: out of
	// the game, no take, and barred to every class or to players
	bestItem("a granite bracer", "WRIST", "NOBITS", "R", 2, dbRow{"con": 4}),
	bestItem("a jade bracer", "WRIST", "NOBITS", "Q", 2, dbRow{"con": 3}),
	bestItem("a copper bracer", "WRIST", "NOBITS", "N", 2, dbRow{"con": 2}),
	bestItem("a cracked crown", "HEAD", "NOBITS", "!", 1, dbRow{"con": 9}),
	bestItem("a stone pillar", "HEAD", "NOTAKE", "X", 900, dbRow{"con": 9}),
	bestItem("a cursed collar", "NECK", "NO-WARRIOR NO-CLERIC NO-THIEF NO-MAGE",
		"N", 1, dbRow{"con": 9}),
	bestItem("an idol's crown", "HEAD", "ANTI-PLAYER", "N", 1, dbRow{"con": 9}),
	// maxcon: a no-rent item to leave out, and a no-take item with a load
	// code other than X, as some have, which never goes in
	bestItem("a gravecloth veil", "FACE", "NORENT", "N", 1, dbRow{"max_con": 4}),
	bestItem("a griffon-feather veil", "FACE", "NOBITS", "R", 1,
		dbRow{"max_con": 3}),
	bestItem("a linen veil", "FACE", "NOBITS", "N", 1, dbRow{"max_con": 2}),
	bestItem("a carved stone visage", "FACE", "NOTAKE", "R", 50,
		dbRow{"max_con": 9}),
	// wis: a tail
	bestItem("a scaled tail band", "TAIL", "NOBITS", "N", 0, dbRow{"wis": 3}),
	bestItem("a sage's ring", "FINGER", "NOBITS", "N", 0, dbRow{"wis": 1}),
	// agi, then pow: a tie that the second stat breaks, or else lightness,
	// and slots only the second stat fills
	bestItem("a silver hoop", "EARRING", "NOBITS", "N", 0, dbRow{"agi": 2}),
	bestItem("a gold hoop", "EARRING", "NOBITS", "N", 1,
		dbRow{"agi": 2, "pow": 1}),
	bestItem("a moonstone pendant", "NECK", "NOBITS", "N", 1, dbRow{"pow": 2}),
	// svbr: a save, better less
	bestItem("a scorched cloak", "ABOUT", "NOBITS", "N", 2, dbRow{"sv_bre": -3}),
	bestItem("a smoky cloak", "ABOUT", "NOBITS", "N", 2, dbRow{"sv_bre": 2}),
	// maxwis, with the essential resists: necks with and without cold, the
	// best cloak, which takes cold away, and a held item, a waist and a badge
	// with them, while testEq's ruby stud, a quest reward, has fire
	bestItem("an owl-feather amulet", "NECK", "NOBITS", "N", 1,
		dbRow{"max_wis": 3}),
	bestItem("a rimed amulet", "NECK", "NOBITS", "N", 1,
		dbRow{"max_wis": 2, "r_cold": 5}),
	bestItem("a shadow-woven mantle", "ABOUT", "NOBITS", "N", 1,
		dbRow{"max_wis": 4, "r_cold": -5}),
	bestItem("a hermit's cloak", "ABOUT", "NOBITS", "N", 2, dbRow{"max_wis": 2}),
	bestItem("a scholar's sash", "WAIST", "NOBITS", "N", 1, dbRow{"max_wis": 2}),
	bestItem("a slimy girdle", "WAIST", "NOBITS", "N", 3,
		dbRow{"max_wis": 1, "r_acid": 5}),
	bestItem("an acid-etched badge", "INSIGNIA", "NOBITS", "N", 1,
		dbRow{"r_acid": 5}),
	bestItem("a frost crystal", "HOLD", "NOBITS", "N", 1, dbRow{"r_cold": 5}),
	// maxdex: an item barred to yuan-ti alone, which a race left blank, any
	// race but yuan-ti, can use
	bestItem("a snakeskin cincture", "WAIST", "ANTI-YUANTI", "N", 1,
		dbRow{"max_dex": 2}),
	// sf_spir capped, then mv: necks and rings with the focus or movement,
	// and bracers with a save or movement
	bestItem("a prayer-bead necklace", "NECK", "NOBITS", "N", 1,
		dbRow{"sf_spi": 5}),
	bestItem("a pilgrim's amulet", "NECK", "NOBITS", "N", 1,
		dbRow{"sf_spi": 2, "move": 20}),
	bestItem("a spirit-bound ring", "FINGER", "NOBITS", "N", 0,
		dbRow{"sf_spi": 3}),
	bestItem("a wanderer's ring", "FINGER", "NOBITS", "N", 0,
		dbRow{"move": 15}),
	bestItem("a ward of stillness", "WRIST", "NOBITS", "N", 1,
		dbRow{"sv_para": -3}),
	bestItem("a calming charm", "WRIST", "NOBITS", "N", 1,
		dbRow{"sv_para": -1, "move": 10}),
}

// fillPicks fills a set for c as o asks, around the items keep names by
// slot, and lists what the fill put where, as "slot=name", then each of the
// essentials the set lacks, as "lacks=fire", and "capout" if it left the cap
// out. It checks that nothing the fill put in shows a warning for c, nor for
// c with each value a part it leaves blank stands for (standsFor).
func fillPicks(t *testing.T, db *sql.DB, c buildChar, o bestOpts,
	keep map[string]string) []string {
	kept := make([]bool, len(buildSlots))
	keptItems := make([]*buildItem, len(buildSlots))
	set := buildSet{names: map[string]string{}}
	for key, name := range keep {
		n := slotIndex(key)
		it, err := itemNamed(db, name)
		chkErr(t, err)
		kept[n], keptItems[n], set.names[key] = true, it, name
	}
	made, err := bestSet(db, c, o, kept, keptItems)
	chkErr(t, err)
	picks := made.picks
	var got []string
	for n, p := range picks {
		if p != nil {
			got = append(got, buildSlots[n].key+"="+p.it.name)
			set.names[buildSlots[n].key] = p.it.name
		}
	}
	for _, e := range made.lacks {
		got = append(got, "lacks="+e)
	}
	if made.capOut {
		got = append(got, "capout")
	}
	chars := []buildChar{c}
	for p, part := range charParts {
		if *part.get(&c) == "" {
			for _, v := range standsFor(p, "") {
				w := c
				*part.get(&w) = v
				chars = append(chars, w)
			}
		}
	}
	for _, w := range chars {
		e, err := build{char: w}.eval(newFinder(db), set)
		chkErr(t, err)
		for n, ws := range e.warns {
			if picks[n] != nil && len(ws) > 0 {
				t.Errorf("bestSet() check failed: %+v %+v put %s in %s, "+
					"which for %+v shows %+v", c, o, picks[n].it.name,
					buildSlots[n].key, w, ws)
			}
		}
	}
	return got
}

func testBuildBest(t *testing.T, db *sql.DB) {
	opts := func(stat, then, hands string, leave ...string) bestOpts {
		return bestOpts{stat: stat, then: then, hands: hands, leave: leave}
	}
	resists := func(o bestOpts) bestOpts {
		o.resists = true
		return o
	}
	capAt := func(n int, o bestOpts) bestOpts {
		o.capped, o.capAt = true, n
		return o
	}
	luck := opts("luck", "", "any")
	intel := func(hands string) bestOpts { return opts("int", "", hands) }
	cha, con, wis := opts("cha", "", "any"), opts("con", "", "any"),
		opts("wis", "", "any")
	wands := []string{"primary=a willow wand", "secondary=a willow wand"}
	pendants := []string{"neck1=a cleric's pendant", "neck2=a cleric's pendant"}
	sash, anklet := "waist=a sash of the evil eye", "feet=a dancer's anklet"
	rings := []string{"finger1=a sage's ring", "finger2=a sage's ring"}
	cleric := buildChar{"Cleric", "Drow Elf", "neutral", "female"}
	wisdom := resists(opts("maxwis", "", "any"))
	badge, ruby := "badge=an acid-etched badge",
		"ear1=a tiny mithril stud set with a ruby"
	owl, rimed := "an owl-feather amulet", "a rimed amulet"
	mantle, crystal := "about=a shadow-woven mantle", "held=a frost crystal"
	crystal2 := "held2=a frost crystal"
	scholar := "waist=a scholar's sash"
	studs := []string{"ear1=a tiny mithril stud set with a ruby",
		"ear2=a tiny mithril stud set with a ruby"}
	human := buildChar{race: "Human"}
	spirit := opts("sf_spir", "mv", "any")
	bead, pilgrim := "a prayer-bead necklace", "a pilgrim's amulet"
	beads := []string{"neck1=" + bead, "neck2=" + bead}
	charms := []string{"wrist1=a calming charm", "wrist2=a calming charm"}
	spiritRings := []string{"finger1=a spirit-bound ring",
		"finger2=a spirit-bound ring"}
	checks := []struct {
		what string
		char buildChar
		opts bestOpts
		keep map[string]string
		want []string
	}{
		{"whole head, and what whole body covers", buildChar{}, luck, nil,
			[]string{"head=a lucky circlet", "body=a padded jerkin",
				"arms=some padded sleeves", "legs=some padded breeches"}},
		{"no whole head over a kept face", buildChar{}, luck,
			map[string]string{"face": "a bone mask"},
			[]string{"head=a leather cap", "body=a padded jerkin",
				"arms=some padded sleeves", "legs=some padded breeches"}},
		{"no whole body over kept arms", buildChar{}, luck,
			map[string]string{"arms": "some padded sleeves"},
			[]string{"head=a lucky circlet", "body=a padded jerkin",
				"legs=some padded breeches"}},
		{"nothing under a kept whole body", buildChar{}, luck,
			map[string]string{"body": "a quilted robe"},
			[]string{"head=a lucky circlet"}},
		{"a tie, to what a whole head would cover", buildChar{},
			opts("sf_ench", "", "any"), nil,
			[]string{"head=a motley coif", "eyes=a pair of motley lenses"}},

		{"hands, any way: two of a weapon, for a warrior",
			buildChar{class: "Warrior"}, intel("any"), nil, wands},
		{"two weapons, for a ranger", buildChar{class: "Ranger"}, intel("two"),
			nil, wands},
		{"hands, any way, with no class chosen: no second weapon", buildChar{},
			intel("any"), nil,
			[]string{"primary=a willow wand", "held=a crystal orb"}},
		{"two weapons, with no class chosen: one", buildChar{}, intel("two"),
			nil, []string{"primary=a willow wand"}},
		{"two weapons, for a psionicist, without dual wield: one",
			buildChar{class: "Psionicist"}, intel("two"), nil,
			[]string{"primary=a willow wand"}},
		{"a weapon and a shield", buildChar{}, intel("shield"), nil,
			[]string{"shield=a warded buckler", "primary=a willow wand"}},
		{"a two-handed weapon", buildChar{}, intel("twohanded"), nil,
			[]string{"primary=an oak cudgel"}},
		{"a weapon and a held item", buildChar{}, intel("held"), nil,
			[]string{"primary=a willow wand", "held=a crystal orb"}},
		{"a shield and a held item", buildChar{}, intel("shieldheld"), nil,
			[]string{"shield=a warded buckler", "held=a crystal orb"}},
		{"two held items", buildChar{}, intel("heldheld"), nil,
			[]string{"held=a crystal orb", "held2=a crystal orb"}},
		{"a two-handed held item", buildChar{}, intel("twohandedheld"), nil,
			[]string{"held=a scrying bowl"}},
		{"a second held item with a kept first", buildChar{},
			intel("heldheld"), map[string]string{"held": "a crystal orb"},
			[]string{"held2=a crystal orb"}},
		{"a first held item with a kept second", buildChar{},
			intel("heldheld"), map[string]string{"held2": "a crystal orb"},
			[]string{"held=a crystal orb"}},
		{"hands left empty", buildChar{}, intel("none"), nil, nil},
		{"no hand left by a kept two-handed weapon", buildChar{}, intel("any"),
			map[string]string{"primary": "an oak cudgel"}, nil},
		{"one hand left by a kept shield", buildChar{}, intel("any"),
			map[string]string{"shield": "a warded buckler"},
			[]string{"primary=a willow wand"}},
		{"a primary weapon with a kept secondary", buildChar{}, intel("any"),
			map[string]string{"secondary": "a willow wand"},
			[]string{"primary=a willow wand"}},

		{"a character the items suit", cleric, cha, nil,
			append(pendants, sash, anklet)},
		{"nothing that doesn't suit everyone", buildChar{}, cha, nil, nil},
		{"no sex chosen", buildChar{"Cleric", "Drow Elf", "neutral", ""}, cha,
			nil, append(pendants, sash)},
		{"illithids have no sex", buildChar{"Cleric", "Illithid", "neutral",
			""}, cha, nil, append(pendants, sash, anklet)},
		{"another class", buildChar{"Warrior", "Drow Elf", "neutral",
			"female"}, cha, nil, []string{sash, anklet}},
		{"a good race and alignment", buildChar{"Cleric", "Human", "good",
			"female"}, cha, nil, append(pendants, anklet)},
		{"nothing a bard may not use", buildChar{"Bard", "Drow Elf",
			"neutral", "female"}, cha, nil, []string{sash, anklet}},

		{"nothing no player can have", buildChar{}, con, nil,
			[]string{"wrist1=a granite bracer", "wrist2=a granite bracer"}},
		{"nothing no player can have, for a bard", buildChar{class: "Bard"},
			con, nil,
			[]string{"wrist1=a granite bracer", "wrist2=a granite bracer"}},
		{"rare items left out", buildChar{}, opts("con", "", "any", "rare"),
			nil, []string{"wrist1=a jade bracer", "wrist2=a jade bracer"}},
		{"rare items and quest rewards left out", buildChar{},
			opts("con", "", "any", "rare", "quest"), nil,
			[]string{"wrist1=a copper bracer", "wrist2=a copper bracer"}},
		{"a no-rent item", buildChar{}, opts("maxcon", "", "any"), nil,
			[]string{"face=a gravecloth veil"}},
		{"no-rent items left out", buildChar{},
			opts("maxcon", "", "any", "norent"), nil,
			[]string{"face=a griffon-feather veil"}},
		{"no-rent and rare items left out", buildChar{},
			opts("maxcon", "", "any", "rare", "norent"), nil,
			[]string{"face=a linen veil"}},

		{"a tail for a yuan-ti", buildChar{race: "Yuan-Ti"}, wis, nil,
			append(rings, "tail=a scaled tail band")},
		{"no tail for a human", buildChar{race: "Human"}, wis, nil, rings},
		{"no tail with no race chosen", buildChar{}, wis, nil, rings},
		{"a kept tail", buildChar{}, wis,
			map[string]string{"tail": "a jeweled tail ring"}, rings},
		{"no whole body, nor legs, for a yuan-ti", buildChar{race: "Yuan-Ti"},
			luck, nil, []string{"head=a lucky circlet", "body=a padded jerkin",
				"arms=some padded sleeves"}},
		{"no earrings for a yuan-ti", buildChar{race: "Yuan-Ti"},
			opts("agi", "pow", "any"), nil,
			[]string{"neck1=a moonstone pendant", "neck2=a moonstone pendant"}},
		{"no anklet for a yuan-ti", buildChar{"Cleric", "Yuan-Ti", "neutral",
			"female"}, cha, nil, append(pendants, sash)},
		{"the essential resists for a yuan-ti, with no ear for fire",
			buildChar{race: "Yuan-Ti"}, wisdom, nil,
			[]string{badge, "neck1=" + owl, "neck2=" + owl, mantle, scholar,
				crystal, crystal2, "lacks=fire"}},
		{"an item barred to yuan-ti, with no race chosen", buildChar{},
			opts("maxdex", "", "any"), nil,
			[]string{"waist=a snakeskin cincture"}},
		{"an item barred to yuan-ti, for a yuan-ti", buildChar{race: "Yuan-Ti"},
			opts("maxdex", "", "any"), nil, nil},
		{"an illusionist, with a Wizard:Illusionist weapon",
			buildChar{class: "Illusionist"}, opts("dam", "", "any"), nil,
			[]string{studs[0], studs[1], "primary=a phantom blade"}},
		{"a sorcerer, without it", buildChar{class: "Sorcerer"},
			opts("dam", "", "any"), nil, []string{studs[0], studs[1],
				"primary=a short sword"}},

		{"a focus, then movement", buildChar{}, spirit, nil,
			append(append(beads, charms...), spiritRings...)},
		{"a focus capped, then movement", buildChar{}, capAt(12, spirit), nil,
			append(append([]string{"neck1=" + bead, "neck2=" + pilgrim},
				charms...), spiritRings...)},
		{"a focus capped at what it gets", buildChar{}, capAt(16, spirit), nil,
			append(append(beads, charms...), spiritRings...)},
		{"a focus capped above what it can get", buildChar{},
			capAt(30, spirit), nil,
			append(append(beads, charms...), spiritRings...)},
		{"a focus capped at nothing: movement alone", buildChar{},
			capAt(0, spirit), nil,
			append(append([]string{"neck1=" + pilgrim, "neck2=" + pilgrim},
				charms...), "finger1=a wanderer's ring",
				"finger2=a wanderer's ring")},
		{"a focus capped, with nothing after it: the lightest", buildChar{},
			capAt(12, opts("sf_spir", "", "any")), nil,
			append(beads, "finger1=a spirit-bound ring")},
		{"a focus capped around a kept ring", buildChar{}, capAt(12, spirit),
			map[string]string{"finger1": "a spirit-bound ring"},
			append(append([]string{"neck1=" + bead, "neck2=" + pilgrim},
				charms...), "finger2=a spirit-bound ring")},
		{"a save capped, then movement", buildChar{},
			capAt(-4, opts("svpar", "mv", "any")), nil,
			[]string{"neck1=" + pilgrim, "neck2=" + pilgrim,
				"wrist1=a ward of stillness", "wrist2=a calming charm",
				"finger1=a wanderer's ring", "finger2=a wanderer's ring"}},
		{"a focus capped, with the essential resists", human,
			resists(capAt(12, spirit)), nil,
			append(append(append([]string{badge, ruby, "neck1=" + bead,
				"neck2=" + pilgrim}, charms...), spiritRings...), crystal)},

		{"a tie, to the lighter", buildChar{}, opts("agi", "", "any"), nil,
			[]string{"ear1=a silver hoop", "ear2=a silver hoop"}},
		{"a tie, to the second stat, which fills more", buildChar{},
			opts("agi", "pow", "any"), nil,
			[]string{"ear1=a gold hoop", "ear2=a gold hoop",
				"neck1=a moonstone pendant", "neck2=a moonstone pendant"}},
		{"a save, better less", buildChar{}, opts("svbr", "", "any"), nil,
			[]string{"about=a scorched cloak"}},
		{"nothing for the stat", buildChar{class: "Lich"},
			opts("sf_sum", "", "any"), nil, nil},

		{"without the essential resists", buildChar{},
			opts("maxwis", "", "any"), nil,
			[]string{"neck1=" + owl, "neck2=" + owl, mantle, scholar}},
		{"the essential resists: two held items for their cold", buildChar{},
			wisdom, nil,
			[]string{badge, ruby, "neck1=" + owl, "neck2=" + owl, mantle,
				scholar, crystal, crystal2}},
		{"the essential resists: a held item for its cold, and an amulet",
			buildChar{}, resists(opts("maxwis", "", "held")), nil,
			[]string{badge, ruby, "neck1=" + rimed, "neck2=" + owl, mantle,
				scholar, crystal}},
		{"the essential resists, with no hands: both amulets", buildChar{},
			resists(opts("maxwis", "", "none")), nil,
			[]string{badge, ruby, "neck1=" + rimed, "neck2=" + rimed, mantle,
				scholar}},
		{"as many of the essential resists as a set can have", buildChar{},
			resists(opts("maxwis", "", "any", "quest")), nil,
			[]string{badge, "neck1=" + owl, "neck2=" + owl, mantle, scholar,
				crystal, crystal2, "lacks=fire"}},
		{"the essential resists, around a kept cloak without cold",
			buildChar{}, wisdom, map[string]string{"about": "a hermit's cloak"},
			[]string{badge, ruby, "neck1=" + owl, "neck2=" + owl, scholar,
				crystal}},
		{"the essential resists, around a kept cloak that takes cold away",
			buildChar{}, wisdom,
			map[string]string{"about": "a shadow-woven mantle"},
			[]string{badge, ruby, "neck1=" + owl, "neck2=" + owl, scholar,
				crystal, crystal2}},
		{"the essential resists, around a kept girdle with acid", buildChar{},
			wisdom, map[string]string{"waist": "a slimy girdle"},
			[]string{ruby, "neck1=" + owl, "neck2=" + owl, mantle, crystal,
				crystal2}},
		{"the essential resists, all kept: the set it would be without them",
			buildChar{}, resists(luck), map[string]string{
				"ear1": "a tiny mithril stud set with a ruby", "neck1": rimed,
				"badge": "an acid-etched badge"},
			[]string{"head=a lucky circlet", "body=a padded jerkin",
				"arms=some padded sleeves", "legs=some padded breeches"}},
	}
	for _, c := range checks {
		chk(t, "bestSet() "+c.what, c.want,
			fillPicks(t, db, c.char, c.opts, c.keep))
	}

	// the essentials are resists, which a fill counts; a plan that would
	// count too many sums of them fails
	for _, e := range essentials {
		if r, ok := findBestStat(e); !ok || r.key != "r:"+e {
			t.Errorf("essentials check failed: %s is no resist", e)
		}
	}
	was := maxWays
	maxWays = 2
	if _, err := bestSet(db, buildChar{}, wisdom,
		make([]bool, len(buildSlots)), make([]*buildItem, len(buildSlots))); err == nil {
		t.Errorf("bestSet() check failed: no error counting more than %d "+
			"sums", maxWays)
	}
	// a cap the essentials would make too much to count is left out
	maxWays = 50
	chk(t, "bestSet() a cap left out", append(append(append(
		[]string{badge, ruby}, beads...), charms...), append(spiritRings,
		crystal, "capout")...),
		fillPicks(t, db, human, resists(capAt(12, spirit)), nil))
	maxWays = was

	for words, want := range map[string]string{"": "", "fire": "fire",
		"fire acid": "fire and acid", "fire cold acid": "fire, cold and acid"} {
		chk(t, "andList() "+words, []string{want},
			[]string{andList(strings.Fields(words))})
	}

	// what a fill is asked for, from a link or form
	for _, c := range []struct {
		v    url.Values
		want bestOpts
	}{
		{url.Values{}, opts("hit", "", "any")},
		{url.Values{"best": {"LUCK"}, "then": {"pow"}, "hands": {"two"},
			"leave": {"quest", "junk", "rare"}},
			opts("luck", "pow", "two", "rare", "quest")},
		{url.Values{"leave": {"norent", "invasion"}},
			opts("hit", "", "any", "invasion", "norent")},
		{url.Values{"best": {"fire"}, "then": {"fire"}}, opts("fire", "", "any")},
		{url.Values{"best": {"wt"}, "then": {"karma"}, "hands": {"juggle"}},
			opts("hit", "", "any")},
		{url.Values{"hands": {"heldheld"}}, opts("hit", "", "heldheld")},
		{url.Values{"hands": {"twohandedheld"}},
			opts("hit", "", "twohandedheld")},
		{url.Values{"resists": {"1"}}, resists(opts("hit", "", "any"))},
		{url.Values{"cap": {" 35 "}}, capAt(35, opts("hit", "", "any"))},
		{url.Values{"cap": {"-10"}}, capAt(-10, opts("hit", "", "any"))},
		{url.Values{"cap": {"0"}}, capAt(0, opts("hit", "", "any"))},
		{url.Values{"cap": {""}}, opts("hit", "", "any")},
		{url.Values{"cap": {"lots"}}, opts("hit", "", "any")},
		{url.Values{"cap": {"10000"}}, opts("hit", "", "any")},
		{url.Values{"cap": {"3.5"}}, opts("hit", "", "any")},
	} {
		got := readBest(c.v)
		if got.stat != c.want.stat || got.then != c.want.then ||
			got.hands != c.want.hands || got.resists != c.want.resists ||
			got.capped != c.want.capped || got.capAt != c.want.capAt ||
			strings.Join(got.leave, " ") != strings.Join(c.want.leave, " ") {
			t.Errorf("readBest() check failed: %v gave %+v, not %+v", c.v, got,
				c.want)
		}
	}

	testBuildFill(t)
}

// testBuildFill checks the page's Fill button, and the page's fill choices.
func testBuildFill(t *testing.T) {
	stud := "a.ear1=a+tiny+mithril+stud"
	luck := func(set string) string {
		return set + ".arms=some+padded+sleeves&" + set +
			".body=a+padded+jerkin&" + set + ".head=a+lucky+circlet&" + set +
			".label=Best+Luck&" + set + ".legs=some+padded+breeches"
	}
	wisdom := "a.about=a+shadow-woven+mantle&a.badge=an+acid-etched+badge" +
		"&a.ear1=a+tiny+mithril+stud+set+with+a+ruby&a.held=a+frost+crystal" +
		"&a.held2=a+frost+crystal&a.label=Best+Max+Wisdom" +
		"&a.neck1=an+owl-feather+amulet&a.neck2=an+owl-feather+amulet" +
		"&a.waist=a+scholar%27s+sash"
	checks := []struct {
		what string
		form url.Values
		want string
	}{
		{"the first empty set", url.Values{"best": {"luck"}, "fill": {"1"}},
			"build?" + luck("a") + "&best=luck"},
		{"a set, in place of what it had", url.Values{"a.ear1": {"stud"},
			"a.label": {"Mine"}, "best": {"luck"}, "fillTo": {"a"},
			"fill": {"1"}}, "build?" + luck("a") + "&best=luck"},
		{"a set, around what it has", url.Values{"a.ear1": {"stud"},
			"b.face": {"bone mask"}, "b.label": {"Mine"}, "best": {"luck"},
			"fillTo": {"b"}, "keep": {"1"}, "fill": {"1"}},
			"build?" + stud + "&b.arms=some+padded+sleeves" +
				"&b.body=a+padded+jerkin&b.face=a+bone+mask" +
				"&b.head=a+leather+cap&b.label=Mine" +
				"&b.legs=some+padded+breeches&best=luck"},
		{"a new set", url.Values{"a.ear1": {"stud"}, "b.ear1": {"stud"},
			"best": {"luck"}, "fillTo": {"new"}, "fill": {"1"}},
			"build?" + stud + "&b.ear1=a+tiny+mithril+stud&best=luck&" +
				luck("c")},
		{"for the character", url.Values{"class": {"Cleric"},
			"race": {"Drow Elf"}, "align": {"neutral"}, "sex": {"female"},
			"best": {"cha"}, "fill": {"1"}},
			"build?a.feet=a+dancer%27s+anklet&a.label=Best+Charisma" +
				"&a.neck1=a+cleric%27s+pendant&a.neck2=a+cleric%27s+pendant" +
				"&a.waist=a+sash+of+the+evil+eye&align=neutral&best=cha" +
				"&class=Cleric&race=Drow+Elf&sex=female"},
		{"nothing that suits everyone", url.Values{"best": {"cha"},
			"fill": {"1"}}, "build?best=cha"},
		{"nothing found", url.Values{"best": {"sf_sum"}, "fill": {"1"}},
			"build?best=sf_sum"},
		{"with the essential resists", url.Values{"best": {"maxwis"},
			"resists": {"1"}, "fill": {"1"}}, "build?" + wisdom +
			"&best=maxwis&resists=1"},
		{"lacking an essential resist", url.Values{"best": {"maxwis"},
			"resists": {"1"}, "leave": {"quest"}, "fill": {"1"}},
			"build?" + strings.Replace(wisdom, "a.ear1=a+tiny+mithril+stud+"+
				"set+with+a+ruby&", "", 1) +
				"&best=maxwis&lacks=fire&leave=quest&resists=1"},
		{"capped, labeled with its cap", url.Values{"best": {"sf_spir"},
			"then": {"mv"}, "cap": {"12"}, "fill": {"1"}},
			"build?a.finger1=a+spirit-bound+ring&a.finger2=a+spirit-bound+ring" +
				"&a.label=Spell+Focus+Spirit+to+12" +
				"&a.neck1=a+prayer-bead+necklace&a.neck2=a+pilgrim%27s+amulet" +
				"&a.wrist1=a+calming+charm&a.wrist2=a+calming+charm" +
				"&best=sf_spir&cap=12&then=mv"},
		{"leaving out no-rent items", url.Values{"best": {"maxcon"},
			"leave": {"norent"}, "fill": {"1"}},
			"build?a.face=a+griffon-feather+veil" +
				"&a.label=Best+Max+Constitution&best=maxcon&leave=norent"},
		{"two held items", url.Values{"best": {"int"}, "hands": {"heldheld"},
			"fill": {"1"}}, "build?a.held=a+crystal+orb&a.held2=a+crystal+orb" +
			"&a.label=Best+Intelligence&best=int&hands=heldheld"},
		{"two weapons, for a warrior", url.Values{"class": {"Warrior"},
			"best": {"int"}, "hands": {"two"}, "fill": {"1"}},
			"build?a.label=Best+Intelligence&a.primary=a+willow+wand" +
				"&a.secondary=a+willow+wand&best=int&class=Warrior&hands=two"},
		{"one weapon, for a psionicist", url.Values{"class": {"Psionicist"},
			"best": {"int"}, "hands": {"two"}, "fill": {"1"}},
			"build?a.label=Best+Intelligence&a.primary=a+willow+wand" +
				"&best=int&class=Psionicist&hands=two"},
		{"the choices, kept", url.Values{"best": {"luck"}, "then": {"pow"},
			"cap": {"-3"}, "hands": {"two"}, "leave": {"norent", "quest", "rare"},
			"resists": {"1"}},
			"build?best=luck&cap=-3&hands=two&leave=rare&leave=quest" +
				"&leave=norent&resists=1&then=pow"},
		{"no note of what an earlier fill lacked", url.Values{
			"best": {"maxwis"}, "resists": {"1"}, "lacks": {"fire"}},
			"build?best=maxwis&resists=1"},
		{"no choice the page doesn't offer", url.Values{"best": {"wt"},
			"then": {"hit"}, "hands": {"juggle"}, "leave": {"store"}}, "build"},
	}
	for _, c := range checks {
		code, loc := postBuild(c.form)
		if code != http.StatusSeeOther || loc != c.want {
			t.Errorf("buildHandler() POST fill %s check failed: %d %q, not %q",
				c.what, code, loc, c.want)
		}
	}

	page := func(query string) string {
		w := httptest.NewRecorder()
		buildHandler(w, httptest.NewRequest("GET", "/build?"+query, nil))
		if w.Code != http.StatusOK {
			t.Errorf("buildHandler() check failed: %s gave %d", query, w.Code)
		}
		return w.Body.String()
	}
	for query, wants := range map[string][]string{
		"": {`<option value="hit" selected="selected">Hitroll</option>`,
			`<option value="any" selected="selected">any way</option>`,
			`<option value="heldheld">two held items</option>`,
			`<option value="twohandedheld">a two-handed held item</option>`,
			`<input type="checkbox" name="leave" value="rare" id="leave-rare" />`,
			`<input type="checkbox" name="leave" value="norent" id="leave-norent" /><label for="leave-norent">no-rent items</label>`,
			`<input type="checkbox" name="resists" value="1" id="resists" />`,
			`<input type="number" name="cap" id="cap" class="cap" value="" step="1" min="-9999" max="9999" />`,
			`<option value="fire">Fire resist</option>`,
			`<button type="submit" name="fill" value="1" id="fill">Fill</button>`,
			"Only bards, battlechanters, hexblades, rangers, rogues and " +
				"warriors can use two weapons"},
		"best=sf_spir&cap=12": {
			`<option value="sf_spir" selected="selected">Spell Focus Spirit</option>`,
			`<input type="number" name="cap" id="cap" class="cap" value="12" step="1" min="-9999" max="9999" />`,
			`best=sf_spir&amp;cap=12" rel="nofollow">`},
		"best=maxwis&resists=1&lacks=cold&lacks=fire&a.head=cap": {
			`<input type="checkbox" name="resists" value="1" id="resists" checked="checked" />`,
			"so the one it made lacks fire and cold resist.",
			`cap&amp;best=maxwis&amp;resists=1" rel="nofollow">`},
		"best=svsp&then=fire&hands=twohanded&leave=quest&a.head=cap": {
			`<option value="svsp" selected="selected">Save Spell</option>`,
			`<option value="fire" selected="selected">Fire resist</option>`,
			`<option value="twohanded" selected="selected">a two-handed weapon</option>`,
			`<input type="checkbox" name="leave" value="quest" id="leave-quest" checked="checked" />`,
			`<input type="checkbox" name="leave" value="rare" id="leave-rare" />`,
			`<option value="b" selected="selected">Set B</option>`,
			"best=svsp&amp;hands=twohanded&amp;leave=quest&amp;then=fire"},
		"leave=norent&a.head=cap": {
			`<input type="checkbox" name="leave" value="norent" id="leave-norent" checked="checked" />`,
			`<input type="checkbox" name="leave" value="rare" id="leave-rare" />`,
			`cap&amp;leave=norent" rel="nofollow">`},
	} {
		body := page(query)
		for _, want := range wants {
			if !strings.Contains(body, want) {
				t.Errorf("buildHandler() check failed: %q has no %s:\n%s",
					query, want, body)
			}
		}
	}
	if body := page("best=maxwis&lacks=fire"); strings.Contains(body,
		"lacks fire") {
		t.Errorf("buildHandler() check failed: a fill that didn't need the " +
			"essential resists noted one it lacked")
	}

	// a cap left out, as the link from that fill says, and only then
	was := maxWays
	maxWays = 50
	code, loc := postBuild(url.Values{"race": {"Human"}, "best": {"sf_spir"},
		"then": {"mv"}, "cap": {"12"}, "resists": {"1"}, "fill": {"1"}})
	maxWays = was
	if code != http.StatusSeeOther || !strings.Contains(loc,
		"a.label=Best+Spell+Focus+Spirit&") || !strings.HasSuffix(loc,
		"&best=sf_spir&cap=12&capout=1&race=Human&resists=1&then=mv") {
		t.Errorf("buildHandler() POST fill a cap left out check failed: %d %q",
			code, loc)
	}
	note := "so it left the cap out."
	if body := page("best=sf_spir&cap=12&resists=1&capout=1"); !strings.Contains(
		body, note) {
		t.Errorf("buildHandler() check failed: no note of a cap left out")
	}
	if body := page("best=sf_spir&capout=1"); strings.Contains(body, note) {
		t.Errorf("buildHandler() check failed: a note of a cap left out, " +
			"with no cap")
	}
}
