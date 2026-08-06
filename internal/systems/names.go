package systems

// Grand Strategy Phase: deterministic NPC name generation (kills "NPC-1").
// Name is a pure function of (identity ID, culture): no RNG stream is consumed,
// so calling it never perturbs simulation determinism. Each culture carries its
// own syllable flavor; endings come from distinct masculine / feminine /
// neutral pools chosen by the identity ID.

// nameCultureCount is the number of distinct culture flavors.
const nameCultureCount = 4

// nameCulture holds the syllable tables of one culture flavor.
type nameCulture struct {
	onsets  [8]string // First syllable (capitalized)
	middles [8]string // Optional second syllable of 3-syllable names
	masc    [4]string // Masculine endings
	fem     [4]string // Feminine endings
	neutral [4]string // Neutral endings
}

var nameCultures = [nameCultureCount]nameCulture{
	{ // Culture 0: soft western flavor
		onsets:  [8]string{"Al", "Ber", "Ce", "Dor", "El", "Fen", "Ga", "Hal"},
		middles: [8]string{"an", "be", "ci", "do", "eth", "fi", "gar", "hem"},
		masc:    [4]string{"ric", "mund", "bert", "win"},
		fem:     [4]string{"lina", "wyn", "dra", "elle"},
		neutral: [4]string{"ley", "ren", "sey", "tan"},
	},
	{ // Culture 1: hard northern flavor
		onsets:  [8]string{"Bjor", "Grim", "Har", "Ing", "Kel", "Ol", "Sig", "Thor"},
		middles: [8]string{"ak", "bran", "dal", "gri", "hol", "mar", "run", "vald"},
		masc:    [4]string{"ulf", "gar", "mund", "stein"},
		fem:     [4]string{"hild", "dis", "veig", "runa"},
		neutral: [4]string{"kel", "vin", "skar", "leif"},
	},
	{ // Culture 2: flowing southern flavor
		onsets:  [8]string{"Ama", "Bel", "Cala", "Della", "Esta", "Fio", "Lu", "Mar"},
		middles: [8]string{"ra", "li", "no", "ve", "sa", "mi", "ta", "ro"},
		masc:    [4]string{"ndo", "rio", "vio", "no"},
		fem:     [4]string{"bella", "rina", "lia", "netta"},
		neutral: [4]string{"ris", "san", "lio", "mar"},
	},
	{ // Culture 3: clipped eastern flavor
		onsets:  [8]string{"Ka", "Zan", "Tem", "Or", "Bai", "Su", "Chu", "Nur"},
		middles: [8]string{"gan", "tai", "bek", "dor", "khan", "lun", "tsu", "yar"},
		masc:    [4]string{"tar", "dai", "bek", "gan"},
		fem:     [4]string{"lai", "mei", "sula", "yin"},
		neutral: [4]string{"chin", "tay", "zor", "kim"},
	},
}

// nameMix is a splitmix64 finalizer: a stateless deterministic bit mixer.
func nameMix(x uint64) uint64 {
	x += 0x9e3779b97f4a7c15
	x = (x ^ (x >> 30)) * 0xbf58476d1ce4e5b9
	x = (x ^ (x >> 27)) * 0x94d049bb133111eb
	return x ^ (x >> 31)
}

// Name deterministically builds a 2-3 syllable name for the given identity ID
// and culture. The gender pool (masculine / feminine / neutral) derives from
// the ID alone so an NPC keeps its gendered name across cultures.
func Name(id uint64, culture uint8) string {
	c := &nameCultures[culture%nameCultureCount]
	h := nameMix(id ^ (uint64(culture) << 56))

	onset := c.onsets[h%8]
	h = nameMix(h)

	middle := ""
	if h%2 == 0 { // 3-syllable names half the time
		h = nameMix(h)
		middle = c.middles[h%8]
	}
	h = nameMix(h)

	var ending string
	switch id % 3 {
	case 0:
		ending = c.masc[h%4]
	case 1:
		ending = c.fem[h%4]
	default:
		ending = c.neutral[h%4]
	}

	return onset + middle + ending
}
