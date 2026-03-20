package layout

import "strings"

// Hyphenator implements Liang-Knuth hyphenation.
type Hyphenator struct {
	patterns map[string][]int
}

// NewHyphenator creates a hyphenator with US English patterns.
func NewHyphenator() *Hyphenator {
	h := &Hyphenator{patterns: make(map[string][]int)}
	h.loadDefaultPatterns()
	return h
}

var defaultHyphenator = NewHyphenator()

// Hyphenate returns the syllables of a word.
func (h *Hyphenator) Hyphenate(word string) []string {
	if len(word) <= 4 {
		return []string{word}
	}

	w := strings.ToLower(word)
	padded := "." + w + "."
	n := len(padded)

	// Array of hyphenation values
	values := make([]int, n+1)

	// Apply patterns
	for i := 0; i < n; i++ {
		for j := i + 1; j <= n; j++ {
			sub := padded[i:j]
			if pat, ok := h.patterns[sub]; ok {
				for k, v := range pat {
					pos := i + k
					if pos < len(values) && v > values[pos] {
						values[pos] = v
					}
				}
			}
		}
	}

	// Build syllables: odd values indicate hyphenation points
	// values[0] corresponds to before first char, values[1] between 1st and 2nd, etc.
	// Offset by 1 for the leading dot
	var syllables []string
	start := 0
	for i := 1; i < len(w); i++ {
		// Don't break within first 2 or last 2 chars
		if i < 2 || i > len(w)-2 {
			continue
		}
		if values[i+1]%2 == 1 { // +1 for leading dot offset
			syllables = append(syllables, word[start:i])
			start = i
		}
	}
	syllables = append(syllables, word[start:])

	if len(syllables) == 0 {
		return []string{word}
	}
	return syllables
}

func (h *Hyphenator) loadDefaultPatterns() {
	// US English hyphenation patterns (subset)
	// Format: letters with digits indicating hyphenation priority
	rawPatterns := []string{
		".ach4", ".ad4der", ".af1t", ".al3t", ".am5at", ".an5c", ".ang4",
		".ani5m", ".ant4", ".ap5at", ".ar5s", ".as3c", ".at5en", ".au5d",
		"a]1a", "a## 2b", "ab5erd", "abet3", "abi5a", "ab5it", "ab3l",
		"ab5ol", "ab3om", "ab5st", "ac1et", "a1ci", "ac5id", "aci4er",
		"ac5ob", "act5if", "acu4b", "ad4din", "ad5er.", "adi4er", "a3dit",
		"ad5l", "ad5ran", "aeacr", "aeri4e", "ag5el", "agi4n", "ag1o",
		"agru5", "ai2", "al5ab", "al3ad", "al1b", "al5d", "al3ed",
		"al3ia", "ali4e", "al5lev", "al1o", "al4th", "a5lu", "am5ab",
		"am3ag", "am5an", "am5az", "am5in", "am5o", "an3ag", "an5arc",
		"an3ar", "an1at", "an3c", "ancy5", "and5it", "an5diz", "an3el",
		"an5est", "an5et", "ang5ie", "an1gl", "an3i", "a5nim", "an5is",
		"an3it", "an4kli", "an3na", "an5o", "an5ot", "ant5ab", "an3th",
		"anti5s", "an4tra", "an5ul", "an5y", "ap5at", "ap5er", "ap3i",
		"ap5ol", "ap5os", "ap3t", "ar5ap", "ar3b", "ar4ce", "ar5d",
		"ar5eas", "a5rea", "ar5eg", "ar3en", "ar5et", "ar5g", "ar3i4a",
		"ar5id", "ar4il", "ar5io", "ar5is", "ar3l", "ar5o4d", "ar5ol",
		"a5ros", "ar3p", "ar5q", "ar5r", "ar4sa", "ar5sh", "4as.",
		"as4ab", "as3ant", "ashi4", "a5sia", "a3sib", "a3sic", "5. ask",
		"as3l", "as4s", "as3ten", "as1tr", "a2th", "ath5em", "a5then",
		"at5omi", "at5on", "at5oo", "a4top", "at5rop", "at5te", "at5ti",
		"au5d", "au3g", "au4l", "aur4", "au5sib", "aw3i", "ax5i",
		"ay5al", "ays4", "1ba", "bad5ger", "bal1a", "ban5dag", "4bar",
		"bas4e", "1bat", "ba4z", "2b1b", "4b1d", "1be", "beas4",
		"4be2d", "be3da", "be3de", "be3di", "be3gi", "be5la", "be3li",
		"be3lo", "be1m", "be5nig", "be5nu", "be3r", "ber4i", "bes4",
		"be5sm", "be5st", "be3tr", "be5wh", "bi4b", "bi4d", "1bil",
		"bi3liz", "bio5l", "bi5ou", "3bis", "bi3t", "3ble.", "2b3li",
		"3bling", "3bly", "4b1m", "bo4e", "bol3i", "bon4a", "bon5at",
		"3boo", "bor5d", "bor5n", "bos4", "5. bot", "b5ot.", "boun4d",
	}

	for _, raw := range rawPatterns {
		h.addPattern(raw)
	}

	// Additional common patterns
	simplePatterns := map[string][]int{
		"tion":  {0, 0, 1, 0},
		"sion":  {0, 0, 1, 0},
		"ing":   {0, 1, 0},
		"ment":  {0, 1, 0, 0},
		"ness":  {0, 1, 0, 0},
		"able":  {0, 1, 0, 0},
		"ible":  {0, 1, 0, 0},
		"ful":   {0, 1, 0},
		"less":  {0, 1, 0, 0},
		"ous":   {0, 1, 0},
		"ive":   {0, 1, 0},
		"ence":  {0, 1, 0, 0},
		"ance":  {0, 1, 0, 0},
		"ment.": {0, 1, 0, 0, 0},
		"pre":   {0, 1, 0},
		"mis":   {0, 1, 0},
		"dis":   {0, 1, 0},
		"over":  {0, 1, 0, 0},
		"under": {0, 1, 0, 0, 0},
		"inter": {0, 0, 1, 0, 0},
		"com":   {0, 1, 0},
		"con":   {0, 1, 0},
	}
	for k, v := range simplePatterns {
		h.patterns[k] = v
	}
}

func (h *Hyphenator) addPattern(raw string) {
	// Parse pattern: letters with embedded digits
	var letters []byte
	var values []int

	values = append(values, 0)
	for i := 0; i < len(raw); i++ {
		ch := raw[i]
		if ch >= '0' && ch <= '9' {
			if len(values) > 0 {
				values[len(values)-1] = int(ch - '0')
			}
		} else if ch == '#' || ch == ']' || ch == '[' || ch == ' ' {
			// Skip special chars from the raw data
			continue
		} else {
			letters = append(letters, ch)
			values = append(values, 0)
		}
	}

	key := string(letters)
	if key != "" {
		h.patterns[key] = values
	}
}
