package commands

import (
	"bufio"
	"errors"
	"fmt"
	"html/template"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/MattSwanson/burtbot/comm"
	"github.com/MattSwanson/burtbot/db"
	"github.com/MattSwanson/burtbot/web"
	"github.com/gempir/go-twitch-irc/v2"
)

// patoi wraps an atoi call and panics if
// the input can not be converted to an
// int. This way we can inline atoi on
// items read in from text files in structs
// also - for even more laziness - returns
// 0 for empty strings...
func patoi(s string) int {
	if s == "" {
		return 0
	}
	i, err := strconv.Atoi(s)
	if err != nil {
		panic(fmt.Sprintf("Failed Atoi: %s", err))
	}
	return i
}

type Item interface {
	getRarity() int
	parseTableRecord(string) error
}

type BaseItem struct {
	Name    string
	Version int
	Rarity  int
	Level   int
}

func (base *BaseItem) getRarity() int {
	return base.Rarity
}

func AmIAnItem(i Item) bool {
	return true
}

type ArmorBase struct {
	BaseItem
}

func (ab *ArmorBase) parseTableRecord(record string) error {
	ln := strings.Split(record, "\t")
	if len(ln) < 14 {
		return errors.New("Not enough columns in input for ArmorBase")
	}
	ab.Name = ln[0]
	ab.Version = patoi(ln[1])
	ab.Rarity = patoi(ln[3])
	ab.Level = patoi(ln[13])
	return nil
}

type WeaponBase struct {
	BaseItem
}

func (wb *WeaponBase) parseTableRecord(record string) error {
	ln := strings.Split(record, "\t")
	if len(ln) < 14 {
		return errors.New("Not enough columns in input for WeaponBase")
	}
	wb.Name = ln[0]
	wb.Version = patoi(ln[6])
	wb.Rarity = patoi(ln[8])
	wb.Level = patoi(ln[30])
	return nil
}

type d2 struct{}
type ItemProbability struct {
	Name string
	Prob int
}
type TreasureClass struct {
	Name   string
	Group  int
	Level  int
	Picks  int
	Unique int
	Set    int
	Rare   int
	Magic  int
	NoDrop int
	Items  []ItemProbability
}

/*
type ArmorBase struct {
	Name    string
	Version int
	Rarity  int
	Level   int
}
type WeaponBase struct {
	Name    string
	Version int
	Rarity  int
	Level   int
}*/
type MiscItem struct {
	Name    string
	Version int
	Rarity  int
	Level   int
	Code    string
}

type ItemRatio struct {
	Quality    string
	BaseChance int
	Divisor    int
	MinChance  int
}

var itemRatios = []ItemRatio{
	ItemRatio{
		Quality:    "Unique",
		BaseChance: 400,
		Divisor:    1,
		MinChance:  6400,
	},
	ItemRatio{
		Quality:    "Set",
		BaseChance: 160,
		Divisor:    2,
		MinChance:  5600,
	},
	ItemRatio{
		Quality:    "Rare",
		BaseChance: 100,
		Divisor:    2,
		MinChance:  3200,
	},
	ItemRatio{
		Quality:    "Magic",
		BaseChance: 34,
		Divisor:    3,
		MinChance:  192,
	},
	ItemRatio{
		Quality:    "HiQuality",
		BaseChance: 12,
		Divisor:    8,
	},
	ItemRatio{
		Quality:    "Normal",
		BaseChance: 2,
		Divisor:    2,
	},
}

var dt = &d2{}
var grailTpl *template.Template

var itemsLoaded bool
var uniqueItems = [][]string{}
var setItems = [][]string{}
var treasureClasses = []TreasureClass{}
var armorBases = []ArmorBase{}
var armorClasses = make(map[string][]ArmorBase)
var weaponBases = []WeaponBase{}
var weaponClasses = make(map[string][]WeaponBase)
var miscItems = []MiscItem{}
var farming bool

func init() {
	rand.Seed(time.Now().UnixNano())
	// load data from files until we migrate to a DB if we do
	f, err := os.Open("./uniqueitems.txt")
	if err != nil {
		log.Println("Couldn't open uniqueitems.txt", err.Error())
	}
	scanner := bufio.NewScanner(f)
	// dump the first line
	scanner.Scan()
	for scanner.Scan() {
		ln := strings.Split(scanner.Text(), "\t")
		if ln[3] != "0" || ln[3] != "" {
			uniqueItems = append(uniqueItems, ln)
		}
	}
	fmt.Printf("%d unique items loaded.\n", len(uniqueItems))
	f.Close()

	f, err = os.Open("./setitems.txt")
	if err != nil {
		log.Println("Couldn't open setitems.txt", err.Error())
	}
	scanner = bufio.NewScanner(f)
	//dump the header line
	scanner.Scan()
	for scanner.Scan() {
		ln := strings.Split(scanner.Text(), "\t")
		if ln[1] != "" {
			setItems = append(setItems, ln)
		}
	}
	f.Close()

	itemsLoaded = true
	loadTreasureClasses()
	loadArmorBases()
	loadWeaponBases()
	loadMiscItems()
	generateAtomicTreasureClasses()
	RegisterCommand("d2", dt)
	grailTpl = template.Must(template.ParseFiles("./templates/grail.gohtml"))
	web.AuthHandleFunc("/grail", unfoundItems)
	web.AuthHandleFunc("/found", foundItem)
}

func loadTreasureClasses() {
	// Load treasure classes from data files
	f, err := os.Open("./d2/data/excel/treasureclassex.txt")
	if err != nil {
		log.Fatal("Couldn't open treasure class file")
	}
	defer f.Close()
	tcs := []TreasureClass{}
	scanner := bufio.NewScanner(f)
	scanner.Scan() // dump header line
	for scanner.Scan() {
		ln := strings.Split(scanner.Text(), "\t")
		ips := []ItemProbability{}
		for i := 0; i < 10; i++ {
			if ln[9+i*2] == "" {
				break
			}
			prob, _ := strconv.Atoi(ln[10+i*2])
			ip := ItemProbability{
				Name: ln[9+i*2],
				Prob: prob,
			}
			ips = append(ips, ip)
		}
		tc := TreasureClass{
			Name:   ln[0],
			Group:  patoi(ln[1]),
			Level:  patoi(ln[2]),
			Picks:  patoi(ln[3]),
			Unique: patoi(ln[4]),
			Set:    patoi(ln[5]),
			Rare:   patoi(ln[6]),
			Magic:  patoi(ln[7]),
			NoDrop: patoi(ln[8]),
			Items:  ips,
		}
		tcs = append(tcs, tc)
	}
	treasureClasses = tcs
	fmt.Println("Finished loading treasure classes")
	// Generate treasue classes needed from armor and weapons files etc.
}

func loadArmorBases() {
	f, err := os.Open("./d2/data/excel/armor.txt")
	if err != nil {
		log.Fatal("Couldn't open armor bases file")
	}
	defer f.Close()
	armorBases = []ArmorBase{}
	scanner := bufio.NewScanner(f)
	scanner.Scan()
	for scanner.Scan() {
		ab := ArmorBase{}
		err := ab.parseTableRecord(scanner.Text())
		if err != nil {
			log.Fatal(err)
		}
		armorBases = append(armorBases, ab)
	}
	fmt.Println("Finished loading armor bases")
}

func loadWeaponBases() {
	f, err := os.Open("./d2/data/excel/weapons.txt")
	if err != nil {
		log.Fatal("Couldn't open weapon bases file")
	}
	defer f.Close()
	weaponBases = []WeaponBase{}
	scanner := bufio.NewScanner(f)
	scanner.Scan()
	for scanner.Scan() {
		wb := WeaponBase{}
		err := wb.parseTableRecord(scanner.Text())
		if err != nil {
			log.Fatal(err)
		}
		weaponBases = append(weaponBases, wb)
	}
	fmt.Println("Finished loading weapon bases")
}

func loadMiscItems() {
	f, err := os.Open("./d2/data/excel/misc.txt")
	if err != nil {
		log.Fatal("Couldn't open misc file")
	}
	defer f.Close()
	miscItems = []MiscItem{}
	scanner := bufio.NewScanner(f)
	scanner.Scan()
	for scanner.Scan() {
		ln := strings.Split(scanner.Text(), "\t")
		mi := MiscItem{
			Name:    ln[0],
			Version: patoi(ln[2]),
			Rarity:  patoi(ln[8]),
			Level:   patoi(ln[3]),
			Code:    ln[14],
		}
		miscItems = append(miscItems, mi)
	}
	fmt.Println("Finished loading misc items")
}

// Generate the TreasureClasses which hold the actual
// item bases that drop
func generateAtomicTreasureClasses() {
	// armor
	for i := 3; i <= 87; i += 3 {
		armorClasses[fmt.Sprintf("armo%d", i)] = []ArmorBase{}
		weaponClasses[fmt.Sprintf("weap%d", i)] = []WeaponBase{}
	}
	// go through the armor bases and put them in the correct armorclass
	for _, base := range armorBases {
		var ac int
		rem := base.Level % 3
		if rem == 0 {
			ac = base.Level
		} else {
			ac = base.Level + (3 - rem)
		}
		acName := fmt.Sprintf("armo%d", ac)
		armorClasses[acName] = append(armorClasses[acName], base)
	}
	// weapons
	for _, base := range weaponBases {
		var ac int
		rem := base.Level % 3
		if rem == 0 {
			ac = base.Level
		} else {
			ac = base.Level + (3 - rem)
		}
		acName := fmt.Sprintf("weap%d", ac)
		weaponClasses[acName] = append(weaponClasses[acName], base)
	}
}

func (d *d2) PostInit() {

}

func (d *d2) Run(msg twitch.PrivateMessage) {

	args := strings.Fields(strings.TrimPrefix(msg.Message, "!"))

	if args[1] == "add" && IsMod(msg.User) {
		loadItemsIntoDB()
	}

	if args[1] == "left" && IsMod(msg.User) {
		getUnfoundItems()
	}

	if args[1] == "found" && IsMod(msg.User) {
		if len(args) < 3 {
			return
		}
		id, err := strconv.Atoi(args[2])
		if err != nil {
			return
		}
		markItemFound(id)
	}

	if args[1] == "unfound" && IsMod(msg.User) {
		if len(args) < 3 {
			return
		}
		id, err := strconv.Atoi(args[2])
		if err != nil {
			return
		}
		markItemUnfound(id)
	}

	if args[1] == "search" {
		searchStr := strings.ToLower(strings.Join(args[2:], " "))
		// some bases have multiple uniques (amu, ring, phase blade others?)
		responses := itemSearch(searchStr)
		if len(responses) == 0 {
			comm.ToChat(msg.Channel, "Couldn't find that item, sorry.")
			return
		}
		for _, resp := range responses {
			comm.ToChat(msg.Channel, resp)
		}
	}

	if args[1] == "findzod" {
		if farming {
			return
		}
		farming = true
		comm.ToChat(msg.Channel, "brb - running Baal until I find a Zod rune...")
		go func() {
			n := farm()
			comm.ToChat(msg.Channel, fmt.Sprintf("I'm back, found Zod after only %d Baal runs.", n))
			farming = false
		}()
	}

	if args[1] == "findtalammy" && !farming {
		farming = true
		comm.ToChat(msg.Channel, "brb - running Baal until I get Tal's Ammy...")
		go func() {
			var runs int
			var found bool
			str := "Set Tal Rasha's Adjudication"
			for {
				runs++
				drops := killMonster()
				for _, item := range drops {
					if strings.EqualFold(str, item) {
						found = true
						break
					}
				}
				if found {
					break
				}
			}
			comm.ToChat(msg.Channel, fmt.Sprintf("Done. Found Tal's Ammy after only %d Baal Runs.", runs))
			farming = false
		}()
	}

	if args[1] == "farm" {
		drops := killMonster()
		comm.ToChat(msg.Channel, "Baal dropped: ")
		for _, drop := range drops {
			comm.ToChat(msg.Channel, drop)
			comm.ToOverlay(fmt.Sprintf("itemdrop %s", drop))
		}
	}

}

func loadItemsIntoDB() {

	for _, unique := range uniqueItems {
		item := db.GrailItem{
			Name:      unique[0],
			SetName:   "",
			BaseItem:  unique[10],
			BaseLevel: 0,
		}
		err := db.AddGrailItem(item)
		if err != nil {
			log.Fatal("couldn't load uniques", err)
		}
	}

	for _, set := range setItems {
		item := db.GrailItem{
			Name:      set[0],
			SetName:   set[2],
			BaseItem:  set[4],
			BaseLevel: 0,
		}
		err := db.AddGrailItem(item)
		if err != nil {
			log.Fatal("Couldn't load sets", err)
		}
	}

	fmt.Println("loading into db complete")
}

func markItemFound(itemID int) error {
	t := time.Now()
	err := db.MarkItemFound(itemID, t)
	if err != nil {
		return err
	}
	// get the item name based off ID
	// send it to the overlay so it can
	// show a notification

	return nil
}

func markItemUnfound(itemID int) error {
	t := time.Date(1, time.January, 1, 1, 1, 1, 1, time.UTC)
	return db.MarkItemFound(itemID, t)
}

func unfoundItems(w http.ResponseWriter, r *http.Request) {
	items, err := db.GetUnfoundGrailItems()
	if err != nil {
		http.Error(w, "Couldn't load items", http.StatusInternalServerError)
		return
	}
	recent, err := db.GetLastFoundItems(5)
	if err != nil {
		http.Error(w, "Couldn't get recent finds", http.StatusInternalServerError)
		return
	}
	d := struct {
		Unfound []db.GrailItem
		Recent  []db.GrailItem
	}{
		Unfound: items,
		Recent:  recent,
	}
	grailTpl.ExecuteTemplate(w, "grail.gohtml", d)
}

func foundItem(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.FormValue("id"))
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	err = markItemFound(id)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "https://burtbot.app/grail", http.StatusSeeOther)
}

func getUnfoundItems() []db.GrailItem {
	items, err := db.GetUnfoundGrailItems()
	if err != nil {
		log.Fatal("coulndn't get em", err)
	}
	return items
}

func itemSearch(input string) []string {
	responses := []string{}
	for _, v := range uniqueItems {
		if input == strings.ToLower(v[0]) || input == strings.ToLower(v[10]) {
			res := fmt.Sprintf("Unique: %s - %s", v[0], v[10])
			responses = append(responses, res)
		}
	}
	for _, v := range setItems {
		if input == strings.ToLower(v[0]) || input == strings.ToLower(v[4]) {
			res := fmt.Sprintf("Set: %s (%s) - %s", v[0], v[2], v[4])
			responses = append(responses, res)
		}
	}
	return responses
}

func getSetItemsForBase(base string) []db.GrailItem {
	items := []db.GrailItem{}
	for _, item := range setItems {
		if strings.ToLower(item[4]) == strings.ToLower(base) {
			match := db.GrailItem{
				Name:      item[0],
				SetName:   item[2],
				BaseItem:  item[4],
				BaseLevel: 0,
				Rarity:    patoi(item[5]),
			}
			items = append(items, match)
		}
	}
	return items
}

func getUniqueItemsForBase(base string) []db.GrailItem {
	items := []db.GrailItem{}
	for _, item := range uniqueItems {
		if strings.ToLower(item[10]) == strings.ToLower(base) {
			match := db.GrailItem{
				Name:      item[0],
				SetName:   "",
				BaseItem:  item[10],
				BaseLevel: 0,
				Rarity:    patoi(item[5]),
			}
			items = append(items, match)
		}
	}
	return items
}

func farm() int {
	// Essentially this is a slot machine
	// which will use the bosses loot info
	// to drop items for the user who "killed"
	// them.  Maybe in the future we can make
	// it like the raids where up to 8 people can
	// join to scale the drop amount and redeem
	// currency for magicfind increases

	// should drop unid items maybe? start without it

	// get monsters treasure class
	tc := getTreasureClass("Baal (H)")
	runs := 0
	for {
		runs++
		nDrops := 0
		var found bool
		for i := 0; i < tc.Picks; i++ {
			drop := pickTreasureClass("Baal (H)")
			if drop != "NoDrop" {
				nDrops++
			}
			if drop == "r33" {
				found = true
			}
			if nDrops >= 5 {
				break
			}
		}
		if found {
			return runs
		}
	}
	return 0
}

func killMonster() []string {
	// Essentially this is a slot machine
	// which will use the bosses loot info
	// to drop items for the user who "killed"
	// them.  Maybe in the future we can make
	// it like the raids where up to 8 people can
	// join to scale the drop amount and redeem
	// currency for magicfind increases

	// should drop unid items maybe? start without it

	// get monsters treasure class
	tc := getTreasureClass("Baal (H)")
	nDrops := 0
	drops := []string{}
	for i := 0; i < tc.Picks; i++ {
		drop := pickTreasureClass("Baal (H)")
		if drop == "NoDrop" {
			continue
		}
		nDrops++
		// get the item base for the treasureClass
		// we ended at (armo87, weap33 etc)
		if strings.Contains(drop, "armo") {
			armorBases := armorClasses[drop]
			prob := 0
			for _, base := range armorBases {
				prob += base.Rarity
			}
			r := rand.Intn(prob)
			var thresh int
			for _, base := range armorBases {
				thresh += base.Rarity
				if r < thresh {
					quality := rollItem(tc, 99, base.Level)
					if quality == "Unique" {
						matches := getUniqueItemsForBase(base.Name)
						if len(matches) == 0 {
							quality = "Superior"
						}
					} else if quality == "Set" {
						matches := getSetItemsForBase(base.Name)
						if len(matches) == 0 {
							quality = "Superior"
						}
					}
					drop = fmt.Sprintf("%s %s", quality, base.Name)
					break
				}
			}
		} else if strings.Contains(drop, "weap") {
			weaponBases := weaponClasses[drop]
			prob := 0
			for _, base := range weaponBases {
				prob += base.Rarity
			}
			r := rand.Intn(prob)
			var thresh int
			for _, base := range weaponBases {
				thresh += base.Rarity
				if r >= thresh {
					continue
				}
				quality := rollItem(tc, 99, base.Level)
				if quality == "Unique" {
					matches := getUniqueItemsForBase(base.Name)
					if len(matches) == 0 {
						quality = "Superior"
					}
				} else if quality == "Set" {
					matches := getSetItemsForBase(base.Name)
					if len(matches) == 0 {
						quality = "Superior"
					}
				}
				drop = fmt.Sprintf("%s %s", quality, base.Name)
				break
			}
		} else if drop == "rin" || drop == "amu" {
			quality := rollItem(tc, 99, 1)
			name := "Ring"
			if drop == "amu" {
				name = "Amulet"
			}
			if quality == "Unique" {
				// chose from avail unique jewelry
				matches := getUniqueItemsForBase(name)
				// add up the rarities, then pick based on rand
				totalRarity := 0
				for _, item := range matches {
					totalRarity += item.Rarity
				}
				r := rand.Intn(totalRarity)
				thresh := 0
				for _, item := range matches {
					thresh += item.Rarity
					if r < thresh {
						drop = fmt.Sprintf("%s %s", quality, item.Name)
						break
					}
				}
			} else if quality == "Set" {
				// same but set
				matches := getSetItemsForBase(name)
				totalRarity, thresh := 0, 0
				for _, item := range matches {
					totalRarity += item.Rarity
				}
				r := rand.Intn(totalRarity)
				for _, item := range matches {
					thresh += item.Rarity
					if r < thresh {
						drop = fmt.Sprintf("%s %s", quality, item.Name)
						break
					}
				}
			} else {
				drop = fmt.Sprintf("%s %s", quality, name)
			}
		} else if strings.HasPrefix(drop, "\"gld") {
			drop = "X Gold"
		} else {
			for _, item := range miscItems {
				if item.Code == drop {
					drop = item.Name
					break
				}
			}
		}
		drops = append(drops, drop)
		if nDrops >= 5 {
			break
		}
	}
	return drops
}

// pickTreasureClass will make a pick from the given
// treasure class and then return the item which is
// picked or nodrop. This could be recursive if the
// item selected is another treasure class
func pickTreasureClass(tcName string) string {
	tc := getTreasureClass(tcName)
	if tc == nil {
		// if we get a nil pointer that means
		// we got an item, so return that string
		// up the chain
		return tcName
	}
	prob := tc.NoDrop
	for _, item := range tc.Items {
		prob += item.Prob
	}
	r := rand.Intn(prob)
	thresh := tc.NoDrop
	if r < thresh {
		return "NoDrop"
	}
	for _, item := range tc.Items {
		thresh += item.Prob
		if r < thresh {
			return pickTreasureClass(item.Name)
		}
	}
	return ""
}

func getTreasureClass(name string) *TreasureClass {
	for _, tc := range treasureClasses {
		if tc.Name == name {
			return &tc
		}
	}
	return nil
}

func rollItem(tc *TreasureClass, mLvl, iLvl int) string {
	for _, ir := range itemRatios {
		// formula for determining item quality
		// p = (quality - (monsterLvl - itemLevel) / qualityDivisor) * 128
		// quality and qualityDivisor are taken from data files
		p := (float64(ir.BaseChance-(mLvl-iLvl)) / float64(ir.Divisor)) * 128.0

		// Magic find is calculated and added to the baseline of 100.
		// MF = 100 + charMF * dim / (charMF + dim)
		// dim is a value which differs based on the item quality being rolled for
		// Unique = 250, Set = 500, Rare = 600

		// We will just use 100 for the time being

		// Adjust the probability from before with the MagicFind
		// p = p * 100 / MF

		// Calculate Probability with treasure class
		// Compare the probability with the min value for the item quality to keep it from reducing any further
		// p
		p = math.Max(p, float64(ir.MinChance))

		// Then modify the probability with the value from the related treasure class
		// p = p - p * treasureClass / 1024
		quality := 1024
		switch ir.Quality {
		case "Unique":
			quality = tc.Unique
		case "Set":
			quality = tc.Set
		case "Rare":
			quality = tc.Rare
		case "Magic":
			quality = tc.Magic
		}
		p = p - (p * float64(quality) / 1024.0)

		// After all of this - roll a number between 0 and the calculated probabilty number. If the random
		// value is less between 0 and 128, then the item has successfully rolled that specific item quality
		// Otherwise, continue on checking for the next lower quality.
		// Unique -> Set -> Rare -> Magic -> Superior -> Normal -> Low Quality
		if p <= 0 {
			return ir.Quality
		}
		r := rand.Intn(int(p))
		if r < 128 {
			return ir.Quality
		}
	}
	return ""
}

func (d *d2) Help() []string {
	return []string{
		"!d2 unique [item] - search for a unique item",
	}
}
