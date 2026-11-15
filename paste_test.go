package main

import (
	"io"
	"log"
	"path/filepath"
	"strings"
	"testing"
)

// Test_ListNames checks that each kind of list the game prints, pasted as
// it comes, gives the names of its items and nothing else. The lines are
// the game's own, from players' logs, with made-up players in them.
func Test_ListNames(t *testing.T) {
	for _, c := range []struct{ what, line, want string }{
		// inventory, a bag, storage
		{"inventory", "You are carrying:", ""},
		{"inventory", "a smoking bracer forged from Baatorian steel (magic) ",
			"a smoking bracer forged from Baatorian steel"},
		{"inventory", "a small spell component pouch ",
			"a small spell component pouch"},
		{"bag", "A fur-lined bag of writhing tentacles (carried) : ", ""},
		{"bag", "[4] a feather ", "a feather"},
		{"storage", "Ann's storage cache contains:", ""},
		{"storage", "[4] a white-hot fiery mace (magic) (illuminating)",
			"a white-hot fiery mace"},

		// equipment
		{"equipment", "You are using:", ""},
		{"equipment", "<worn on head>       a star-metal chainmail hood of " +
			"boundless thought (magic)",
			"a star-metal chainmail hood of boundless thought"},
		{"equipment", "<held>               an ancient staff of the arch-magi",
			"an ancient staff of the arch-magi"},
		{"equipment", "<worn around wrist>  a bracelet of brainstealer scales " +
			"(magic)", "a bracelet of brainstealer scales"},
		// what torilmud.tin passes -glist from someone's equipment
		{"seen worn", "a necklace of mental prowess (magic)",
			"a necklace of mental prowess"},

		// glist: who left an item in the group cache, in a column before it
		{"glist", "Ann                  a black longsword of destruction",
			"a black longsword of destruction"},
		{"glist", "                                the mark of the dragonhunter",
			"the mark of the dragonhunter"},

		// a shop's list, which the old reader looked up whole, price and all
		{"shop", "You can buy:", ""},
		{"shop", " 1) A thick black iron key for 10 platinum coins. ",
			"A thick black iron key"},
		{"shop", " 3) a plate of chicken tarta for 1 silver, and 5 copper " +
			"coins. ", "a plate of chicken tarta"},
		{"shop", " 6) a light fruit plate for 1 silver coin. ",
			"a light fruit plate"},
		{"shop", " 1) A double-bladed two-handed battleaxe for 5 platinum " +
			"coins. (*)", "A double-bladed two-handed battleaxe"},
		{"shop", "5) A broadsword of dancing shadows  for 18 platinum, and 3 " +
			"gold coins.", "A broadsword of dancing shadows"},
		{"shop", " 4) White wine in a cup of Jhalessa's spiced wine for 3 " +
			"silver coins.", "White wine in a cup of Jhalessa's spiced wine"},

		// auction list, whose items the old reader cut at a fixed column
		{"auction", "Auction list", ""},
		{"auction", "item#)    item id                                item " +
			"description            bid      bidinc        top bidder time  " +
			"status seller", ""},
		{"auction", "====== ========== ======================================" +
			"========= ============== =========== ================= ==== " +
			"======= ======", ""},
		{"auction", "    1)     100114               a pouch of rubbery " +
			"deepspawn hide            100          50               Ann  N/A" +
			"   BUYER Bob", "a pouch of rubbery deepspawn hide"},
		{"auction", "    4)     100226 a cloak emblazoned with the seal of " +
			"Caer Corwell            200          10        No Bid Yet  N/A  " +
			"SELLER Bob", "a cloak emblazoned with the seal of Caer Corwell"},
		{"auction", "    8)     100364                     a necklace of " +
			"viper's teeth          5,000          50               Ann  N/A" +
			"   BUYER Bob", "a necklace of viper's teeth"},
		{"auction", "   16)     100521                      a potion labeled " +
			"'Healing'             50          10        No Bid Yet  N/A  " +
			"SELLER Bob", "a potion labeled 'Healing'"},
		// a line wrapped after the item, and what it leaves on lines of its own
		{"auction, wrapped", "   12)     100423             a pair of shining " +
			"emerald gauntlets", "a pair of shining emerald gauntlets"},
		{"auction, wrapped", "2,000         100        No Bid Yet 27 d    " +
			"OPEN Bob", ""},
		{"auction, wrapped", "OPEN Bob", ""},
		{"auction, wrapped", "BUYER Bob", ""},
		{"auction, wrapped", "Ann  N/A  SELLER Bob", ""},
		{"auction, wrapped", "15          10", ""},

		// what comes along in a paste and names no item
		{"prompt", "< 751h/751H 991p/991P 140v/140V >", ""},
		{"prompt", "< P: sit > lib", ""},
		{"pager", "[RETURN for more]", ""},
		{"border", "|   --------", ""},
		{"blank", "   ", ""},
		{"blank", "", ""},
		// and what doesn't, which is looked up and reported missing
		{"other", "something unknown", "something unknown"},
	} {
		if got := lineName(c.line); got != c.want {
			t.Errorf("lineName() of %s line %q = %q, want %q", c.what, c.line,
				got, c.want)
		}
	}

	// a paste as the page gets it, with Windows line ends
	chk(t, "listNames()", []string{"a feather", "a black longsword of " +
		"destruction", "a necklace of viper's teeth"},
		listNames("You are carrying:\r\n[4] a feather\r\n\r\n<wielded>   a "+
			"black longsword of destruction (magic)\r\n    8)     100364     "+
			"a necklace of viper's teeth          5,000          50          "+
			"No Bid Yet  N/A  SELLER Bob\r\nOPEN Bob\r\n"))
}

// Test_PastedItem checks that a name as a shop shows it finds the item:
// with a capital first letter, or a drink by its container, but only when
// the name as given finds nothing.
func Test_PastedItem(t *testing.T) {
	defer log.SetOutput(log.Writer())
	log.SetOutput(io.Discard) // the 404 at the end logs as on the site

	var items []dbRow
	for _, name := range []string{
		"a thick black iron key", "a cup of Jhalessa's spiced wine",
		"a nebulous earring of water", "the Blade of Blessings", "Sting",
		"a box", "a basket", "a bracelet of mages (blue) 1",
		"a ship in a bottle", "a bottle", "Ale in a tankard", "a tankard",
	} {
		items = append(items, dbRow{"name": name, "long_stats": "long " + name})
	}
	path := filepath.Join(t.TempDir(), itemDBFile)
	writeItemDB(t, path, items)
	db, err := openItemDB(path)
	chkErr(t, err)
	defer db.Close()

	for name, want := range map[string]string{
		"A thick black iron key": "a thick black iron key",
		"The Blade of Blessings": "the Blade of Blessings",
		"Sting":                  "Sting",
		"White wine in a cup of Jhalessa's spiced wine": "a cup of Jhalessa's spiced wine",
		"Water in a nebulous earring of water":          "a nebulous earring of water",
		// an item named for what's in it comes as itself, not its container
		"A ship in a bottle": "a ship in a bottle",
		"Ale in a tankard":   "Ale in a tankard",
		// and an item Kegor lacks isn't taken for its container
		"Some boots in a box":   "",
		"An apple in a basket":  "",
		"A bracelet of mages":   "a bracelet of mages (blue) 1",
		"a thick black iron ke": "",
	} {
		got, err := pastedItem(db, name)
		chkErr(t, err)
		if got != want {
			t.Errorf("pastedItem(%q) = %q, want %q", name, got, want)
		}
	}

	// a whole paste, as ParseList looks it up
	var stats []string
	for _, name := range listNames(strings.Join([]string{
		"You can buy:",
		" 1) A thick black iron key for 10 platinum coins.",
		" 2) White wine in a cup of Jhalessa's spiced wine for 3 silver coins.",
		" 3) A gnomish mechanical canary for 5 gold coins.",
	}, "\n")) {
		stats = append(stats, exactItem(db, name))
	}
	chk(t, "exactItem() of a shop's list", []string{
		"long a thick black iron key", "long a cup of Jhalessa's spiced wine",
		"404 item not found: A gnomish mechanical canary"}, stats)
}
