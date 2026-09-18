package vietnamese

import (
	"unicode"
)

// Tone marks in Vietnamese
const (
	ToneNone = iota
	ToneSac   // s
	ToneHuyen // f
	ToneHoi   // r
	ToneNga   // x
	ToneNang  // j
)

type charData struct {
	base      rune
	hatOrHorn rune // 'a' -> 'â', 'ă', etc.
	tone      int
}

// Table of Vietnamese vowel mappings
// Row: base vowel ('a', 'ă', 'â', 'e', 'ê', 'i', 'o', 'ô', 'ơ', 'u', 'ư', 'y')
// Col: Tone (None, Sac, Huyen, Hoi, Nga, Nang)
var vowelTones = map[rune][6]rune{
	'a': {'a', 'á', 'à', 'ả', 'ã', 'ạ'},
	'A': {'A', 'Á', 'À', 'Ả', 'Ã', 'Ạ'},
	'ă': {'ă', 'ắ', 'ằ', 'ẳ', 'ẵ', 'ặ'},
	'Ă': {'Ă', 'Ắ', 'Ằ', 'Ẳ', 'Ẵ', 'Ặ'},
	'â': {'â', 'ấ', 'ầ', 'ẩ', 'ẫ', 'ậ'},
	'Â': {'Â', 'Ấ', 'Ầ', 'Ẩ', 'Ẫ', 'Ậ'},
	'e': {'e', 'é', 'è', 'ẻ', 'ẽ', 'ẹ'},
	'E': {'E', 'É', 'È', 'Ẻ', 'Ẽ', 'Ẹ'},
	'ê': {'ê', 'ế', 'ề', 'ể', 'ễ', 'ệ'},
	'Ê': {'Ê', 'Ế', 'Ề', 'Ể', 'Ễ', 'Ệ'},
	'i': {'i', 'í', 'ì', 'ỉ', 'ĩ', 'ị'},
	'I': {'I', 'Í', 'Ì', 'Ỉ', 'Ĩ', 'Ị'},
	'o': {'o', 'ó', 'ò', 'ỏ', 'õ', 'ọ'},
	'O': {'O', 'Ó', 'Ò', 'Ỏ', 'Õ', 'Ọ'},
	'ô': {'ô', 'ố', 'ồ', 'ổ', 'ỗ', 'ộ'},
	'Ô': {'Ô', 'Ố', 'Ồ', 'Ổ', 'Ỗ', 'Ộ'},
	'ơ': {'ơ', 'ớ', 'ờ', 'ở', 'ỡ', 'ợ'},
	'Ơ': {'Ơ', 'Ớ', 'Ờ', 'Ở', 'Ỡ', 'Ợ'},
	'u': {'u', 'ú', 'ù', 'ủ', 'ũ', 'ụ'},
	'U': {'U', 'Ú', 'Ù', 'Ủ', 'Ũ', 'Ụ'},
	'ư': {'ư', 'ứ', 'ừ', 'ử', 'ữ', 'ự'},
	'Ư': {'Ư', 'Ứ', 'Ừ', 'Ử', 'Ữ', 'Ự'},
	'y': {'y', 'ý', 'ỳ', 'ỷ', 'ỹ', 'ỵ'},
	'Y': {'Y', 'Ý', 'Ỳ', 'Ỷ', 'Ỹ', 'Ỵ'},
}

// Map any accented vowel back to its (root vowel, tone)
var reverseVowelMap = make(map[rune]struct {
	root rune
	tone int
})

func init() {
	for root, row := range vowelTones {
		for t, r := range row {
			reverseVowelMap[r] = struct {
				root rune
				tone int
			}{root: root, tone: t}
		}
	}
}

// IsVowel checks if a rune is a Vietnamese vowel.
func IsVowel(r rune) bool {
	_, ok := reverseVowelMap[r]
	return ok
}

// RemoveTone removes tone mark from a vowel.
func RemoveTone(r rune) rune {
	if info, ok := reverseVowelMap[r]; ok {
		return info.root
	}
	return r
}

// GetTone returns the tone of a vowel (0-5).
func GetTone(r rune) int {
	if info, ok := reverseVowelMap[r]; ok {
		return info.tone
	}
	return ToneNone
}

// ApplyTone applies a tone (0-5) to a root vowel.
func ApplyTone(root rune, tone int) rune {
	if tone < 0 || tone > 5 {
		return root
	}
	if row, ok := vowelTones[root]; ok {
		return row[tone]
	}
	// If root itself has a tone, get base root first
	if info, ok := reverseVowelMap[root]; ok {
		if row, ok := vowelTones[info.root]; ok {
			return row[tone]
		}
	}
	return root
}

// TransformWordTelex attempts to apply a Telex key (such as s, f, r, x, j, a, e, o, w, d, z)
// to a given word. If a transformation happens, it returns (newWord, true).
// If no rule matches, it returns (word, false).
func TransformWordTelex(word string, keyChar rune) (string, bool) {
	if len(word) == 0 {
		// Single key press at start of word
		if keyChar == 'w' {
			return "ư", true
		}
		if keyChar == 'W' {
			return "Ư", true
		}
		return "", false
	}

	runes := []rune(word)
	kLower := unicode.ToLower(keyChar)
	isUpperKey := unicode.IsUpper(keyChar)

	// 1. Check for 'd' + 'd' -> 'đ' / 'D' + 'd' -> 'Đ'
	if kLower == 'd' {
		lastRune := runes[len(runes)-1]
		if lastRune == 'd' {
			runes[len(runes)-1] = 'đ'
			return string(runes), true
		} else if lastRune == 'D' {
			runes[len(runes)-1] = 'Đ'
			return string(runes), true
		} else if lastRune == 'đ' {
			// Revert: đ + d -> dd
			if isUpperKey {
				runes[len(runes)-1] = 'd'
				runes = append(runes, 'D')
			} else {
				runes[len(runes)-1] = 'd'
				runes = append(runes, 'd')
			}
			return string(runes), true
		} else if lastRune == 'Đ' {
			// Revert: Đ + d -> Dd or DD
			if isUpperKey {
				runes[len(runes)-1] = 'D'
				runes = append(runes, 'D')
			} else {
				runes[len(runes)-1] = 'D'
				runes = append(runes, 'd')
			}
			return string(runes), true
		}
	}

	// 2. Check for vowel modifier: 'a', 'e', 'o', 'w'
	// 'a' + 'a' -> 'â', 'e' + 'e' -> 'ê', 'o' + 'o' -> 'ô'
	// 'a' + 'w' -> 'ă', 'o' + 'w' -> 'ơ', 'u' + 'w' -> 'ư', 'uo' + 'w' -> 'ươ'
	switch kLower {
	case 'a':
		// Search for 'a' to turn into 'â'
		if modIdx := findLastVowelForMod(runes, 'a', 'A'); modIdx >= 0 {
			r := runes[modIdx]
			tone := GetTone(r)
			base := RemoveTone(r)
			if base == 'a' {
				runes[modIdx] = ApplyTone('â', tone)
				return string(runes), true
			} else if base == 'A' {
				runes[modIdx] = ApplyTone('Â', tone)
				return string(runes), true
			} else if base == 'â' || base == 'Â' {
				// Revert: â + a -> aa
				orig := 'a'
				if base == 'Â' {
					orig = 'A'
				}
				runes[modIdx] = ApplyTone(orig, tone)
				if isUpperKey {
					runes = append(runes, 'A')
				} else {
					runes = append(runes, 'a')
				}
				return string(runes), true
			}
		}
	case 'e':
		if modIdx := findLastVowelForMod(runes, 'e', 'E'); modIdx >= 0 {
			r := runes[modIdx]
			tone := GetTone(r)
			base := RemoveTone(r)
			if base == 'e' {
				runes[modIdx] = ApplyTone('ê', tone)
				return string(runes), true
			} else if base == 'E' {
				runes[modIdx] = ApplyTone('Ê', tone)
				return string(runes), true
			} else if base == 'ê' || base == 'Ê' {
				// Revert: ê + e -> ee
				orig := 'e'
				if base == 'Ê' {
					orig = 'E'
				}
				runes[modIdx] = ApplyTone(orig, tone)
				if isUpperKey {
					runes = append(runes, 'E')
				} else {
					runes = append(runes, 'e')
				}
				return string(runes), true
			}
		}
	case 'o':
		if modIdx := findLastVowelForMod(runes, 'o', 'O'); modIdx >= 0 {
			r := runes[modIdx]
			tone := GetTone(r)
			base := RemoveTone(r)
			if base == 'o' {
				runes[modIdx] = ApplyTone('ô', tone)
				return string(runes), true
			} else if base == 'O' {
				runes[modIdx] = ApplyTone('Ô', tone)
				return string(runes), true
			} else if base == 'ô' || base == 'Ô' {
				// Revert: ô + o -> oo
				orig := 'o'
				if base == 'Ô' {
					orig = 'O'
				}
				runes[modIdx] = ApplyTone(orig, tone)
				if isUpperKey {
					runes = append(runes, 'O')
				} else {
					runes = append(runes, 'o')
				}
				return string(runes), true
			}
		}
	case 'w':
		// Check for 'uo' -> 'ươ' / 'uO' -> 'ưƠ'
		if uIdx, oIdx := findUOPair(runes); uIdx >= 0 && oIdx >= 0 {
			uR := runes[uIdx]
			oR := runes[oIdx]
			uTone := GetTone(uR)
			oTone := GetTone(oR)
			uBase := RemoveTone(uR)
			oBase := RemoveTone(oR)
			if uBase == 'u' && oBase == 'o' {
				runes[uIdx] = ApplyTone('ư', uTone)
				runes[oIdx] = ApplyTone('ơ', oTone)
				return string(runes), true
			} else if uBase == 'U' && oBase == 'O' {
				runes[uIdx] = ApplyTone('Ư', uTone)
				runes[oIdx] = ApplyTone('Ơ', oTone)
				return string(runes), true
			} else if uBase == 'U' && oBase == 'o' {
				runes[uIdx] = ApplyTone('Ư', uTone)
				runes[oIdx] = ApplyTone('ơ', oTone)
				return string(runes), true
			}
		}

		// Check for 'a' -> 'ă'
		if aIdx := findLastVowelForMod(runes, 'a', 'A'); aIdx >= 0 {
			r := runes[aIdx]
			tone := GetTone(r)
			base := RemoveTone(r)
			if base == 'a' {
				runes[aIdx] = ApplyTone('ă', tone)
				return string(runes), true
			} else if base == 'A' {
				runes[aIdx] = ApplyTone('Ă', tone)
				return string(runes), true
			} else if base == 'ă' || base == 'Ă' {
				// Revert: ă + w -> aw
				orig := 'a'
				if base == 'Ă' {
					orig = 'A'
				}
				runes[aIdx] = ApplyTone(orig, tone)
				if isUpperKey {
					runes = append(runes, 'W')
				} else {
					runes = append(runes, 'w')
				}
				return string(runes), true
			}
		}

		// Check for 'o' -> 'ơ'
		if oIdx := findLastVowelForMod(runes, 'o', 'O'); oIdx >= 0 {
			r := runes[oIdx]
			tone := GetTone(r)
			base := RemoveTone(r)
			if base == 'o' {
				runes[oIdx] = ApplyTone('ơ', tone)
				return string(runes), true
			} else if base == 'O' {
				runes[oIdx] = ApplyTone('Ơ', tone)
				return string(runes), true
			} else if base == 'ơ' || base == 'Ơ' {
				// Revert: ơ + w -> ow
				orig := 'o'
				if base == 'Ơ' {
					orig = 'O'
				}
				runes[oIdx] = ApplyTone(orig, tone)
				if isUpperKey {
					runes = append(runes, 'W')
				} else {
					runes = append(runes, 'w')
				}
				return string(runes), true
			}
		}

		// Check for 'u' -> 'ư'
		if uIdx := findLastVowelForMod(runes, 'u', 'U'); uIdx >= 0 {
			r := runes[uIdx]
			tone := GetTone(r)
			base := RemoveTone(r)
			if base == 'u' {
				runes[uIdx] = ApplyTone('ư', tone)
				return string(runes), true
			} else if base == 'U' {
				runes[uIdx] = ApplyTone('Ư', tone)
				return string(runes), true
			} else if base == 'ư' || base == 'Ư' {
				// Revert: ư + w -> uw
				orig := 'u'
				if base == 'Ư' {
					orig = 'U'
				}
				runes[uIdx] = ApplyTone(orig, tone)
				if isUpperKey {
					runes = append(runes, 'W')
				} else {
					runes = append(runes, 'w')
				}
				return string(runes), true
			}
		}

		// If standalone 'w' at end of word or after consonant (e.g. "thw" -> "thư")
		if len(runes) > 0 {
			last := runes[len(runes)-1]
			if !IsVowel(last) {
				if isUpperKey {
					runes = append(runes, 'Ư')
				} else {
					runes = append(runes, 'ư')
				}
				return string(runes), true
			}
		}
	}

	// 3. Check for Tone keys: s, f, r, x, j, z
	targetTone := ToneNone
	switch kLower {
	case 's':
		targetTone = ToneSac
	case 'f':
		targetTone = ToneHuyen
	case 'r':
		targetTone = ToneHoi
	case 'x':
		targetTone = ToneNga
	case 'j':
		targetTone = ToneNang
	case 'z':
		targetTone = ToneNone
	default:
		return "", false
	}

	// Before applying tone, normalize common diphthongs if ending with consonant
	// e.g. "ie" -> "iê" in "tiengs" -> "tiếng", "vietj" -> "việt"
	// "ươ" in "thuwocs" -> "thước"
	normalizeDiphthongs(runes)

	// Find the best vowel index to place tone
	toneIdx, curTone := findToneTarget(runes)
	if toneIdx >= 0 {
		// If pressing same tone key again -> revert tone and append raw letter
		if curTone == targetTone && targetTone != ToneNone {
			runes[toneIdx] = ApplyTone(runes[toneIdx], ToneNone)
			if isUpperKey {
				runes = append(runes, unicode.ToUpper(keyChar))
			} else {
				runes = append(runes, unicode.ToLower(keyChar))
			}
			return string(runes), true
		}

		// Apply target tone
		runes[toneIdx] = ApplyTone(runes[toneIdx], targetTone)
		// Clear tone on any other vowel if present
		for i, r := range runes {
			if i != toneIdx && IsVowel(r) {
				runes[i] = ApplyTone(r, ToneNone)
			}
		}
		return string(runes), true
	}

	return "", false
}

// normalizeDiphthongs automatically upgrades diphthongs like ie->iê, ưo->ươ before tone application
func normalizeDiphthongs(runes []rune) {
	for i := 0; i < len(runes)-1; i++ {
		r1 := RemoveTone(runes[i])
		r2 := RemoveTone(runes[i+1])
		r1Lower := unicode.ToLower(r1)
		r2Lower := unicode.ToLower(r2)

		hasEndConsonant := i+1 < len(runes)-1 && !IsVowel(runes[len(runes)-1])

		// ie + consonant -> iê
		if r1Lower == 'i' && r2Lower == 'e' && hasEndConsonant {
			if unicode.IsUpper(runes[i+1]) {
				runes[i+1] = ApplyTone('Ê', GetTone(runes[i+1]))
			} else {
				runes[i+1] = ApplyTone('ê', GetTone(runes[i+1]))
			}
		}

		// ye + consonant -> yê
		if r1Lower == 'y' && r2Lower == 'e' && hasEndConsonant {
			if unicode.IsUpper(runes[i+1]) {
				runes[i+1] = ApplyTone('Ê', GetTone(runes[i+1]))
			} else {
				runes[i+1] = ApplyTone('ê', GetTone(runes[i+1]))
			}
		}

		// ư + o -> ư + ơ
		if (r1 == 'ư' || r1 == 'Ư') && r2Lower == 'o' {
			if unicode.IsUpper(runes[i+1]) {
				runes[i+1] = ApplyTone('Ơ', GetTone(runes[i+1]))
			} else {
				runes[i+1] = ApplyTone('ơ', GetTone(runes[i+1]))
			}
		}

		// u + ơ -> ư + ơ
		if r1Lower == 'u' && (r2 == 'ơ' || r2 == 'Ơ') {
			if unicode.IsUpper(runes[i]) {
				runes[i] = ApplyTone('Ư', GetTone(runes[i]))
			} else {
				runes[i] = ApplyTone('ư', GetTone(runes[i]))
			}
		}
	}
}

// findLastVowelForMod finds the index of a vowel eligible for modifier ('a', 'e', 'o', 'u')
func findLastVowelForMod(runes []rune, targets ...rune) int {
	for i := len(runes) - 1; i >= 0; i-- {
		base := RemoveTone(runes[i])
		for _, t := range targets {
			if base == t {
				return i
			}
			// Also check if already modified (for revert)
			if (t == 'a' && (base == 'â' || base == 'ă')) ||
				(t == 'A' && (base == 'Â' || base == 'Ă')) ||
				(t == 'e' && base == 'ê') || (t == 'E' && base == 'Ê') ||
				(t == 'o' && (base == 'ô' || base == 'ơ')) ||
				(t == 'O' && (base == 'Ô' || base == 'Ơ')) ||
				(t == 'u' && base == 'ư') || (t == 'U' && base == 'Ư') {
				return i
			}
		}
	}
	return -1
}

// findUOPair searches for consecutive "uo" / "ươ"
func findUOPair(runes []rune) (int, int) {
	for i := 0; i < len(runes)-1; i++ {
		uBase := unicode.ToLower(RemoveTone(runes[i]))
		oBase := unicode.ToLower(RemoveTone(runes[i+1]))
		if (uBase == 'u' || uBase == 'ư') && (oBase == 'o' || oBase == 'ơ') {
			return i, i + 1
		}
	}
	return -1, -1
}

// findToneTarget chooses the correct vowel index to place the tone mark according to Vietnamese rules.
func findToneTarget(runes []rune) (int, int) {
	var vowelIndices []int
	for i, r := range runes {
		if IsVowel(r) {
			vowelIndices = append(vowelIndices, i)
		}
	}

	if len(vowelIndices) == 0 {
		return -1, ToneNone
	}

	// Filter out 'u' in 'qu' and 'i' in 'gi' if other vowels exist
	filtered := make([]int, 0, len(vowelIndices))
	for _, idx := range vowelIndices {
		if idx > 0 {
			prev := unicode.ToLower(runes[idx-1])
			curr := unicode.ToLower(RemoveTone(runes[idx]))
			if prev == 'q' && curr == 'u' && len(vowelIndices) > 1 {
				continue
			}
			if prev == 'g' && curr == 'i' && len(vowelIndices) > 1 {
				continue
			}
		}
		filtered = append(filtered, idx)
	}

	if len(filtered) == 0 {
		filtered = vowelIndices
	}

	// Check if any vowel currently has a tone
	currentTone := ToneNone
	for _, idx := range filtered {
		t := GetTone(runes[idx])
		if t != ToneNone {
			currentTone = t
			break
		}
	}

	// Rule 1: Single vowel
	if len(filtered) == 1 {
		return filtered[0], currentTone
	}

	// Check if word has ending consonant after last vowel
	lastVowelIdx := filtered[len(filtered)-1]
	hasEndingConsonant := lastVowelIdx < len(runes)-1

	// Rule 2: Multiple vowels ending with consonant (e.g. "toan", "tieng", "nghieng", "thuan", "thước")
	// -> Tone goes on the main vowel (ê, ơ, ô, â, ă first, then ư, or second vowel)
	if hasEndingConsonant {
		// Priority 1: vowels with hat/horn (ê, ơ, ô, â, ă)
		for _, idx := range filtered {
			base := unicode.ToLower(RemoveTone(runes[idx]))
			if base == 'ê' || base == 'ơ' || base == 'ô' || base == 'â' || base == 'ă' {
				return idx, currentTone
			}
		}
		// Priority 2: 'ư'
		for _, idx := range filtered {
			base := unicode.ToLower(RemoveTone(runes[idx]))
			if base == 'ư' {
				return idx, currentTone
			}
		}
		// Otherwise tone on the last vowel in the nucleus (e.g. 'a' in 'oan', 'uyn')
		if len(filtered) >= 2 {
			return filtered[len(filtered)-1], currentTone
		}
		return filtered[0], currentTone
	}

	// Rule 3: Multiple vowels without ending consonant (e.g. "hoa", "thuy", "nguoi", "tươi")
	// Priority 1: vowels with hat/horn (ê, ơ, ô, â)
	for _, idx := range filtered {
		base := unicode.ToLower(RemoveTone(runes[idx]))
		if base == 'ê' || base == 'ơ' || base == 'ô' || base == 'â' {
			return idx, currentTone
		}
	}

	// Diphthong without hat ending in open vowel (e.g. "hòa", "hóa", "thủy", "túi")
	// In standard modern orthography:
	// "oa", "oe", "uy" -> tone on the second vowel ("hóa", "hòe", "thủy")
	if len(filtered) == 2 {
		first := unicode.ToLower(RemoveTone(runes[filtered[0]]))
		second := unicode.ToLower(RemoveTone(runes[filtered[1]]))
		if (first == 'o' && (second == 'a' || second == 'e')) || (first == 'u' && second == 'y') {
			return filtered[1], currentTone
		}
		// For "ai", "oi", "ui", "ay", "âu", "eo", "ao", "iu" -> tone on the first vowel
		return filtered[0], currentTone
	}

	if len(filtered) == 3 {
		// Triphthongs like "oai", "uay" -> tone on middle vowel
		return filtered[1], currentTone
	}

	return filtered[len(filtered)-1], currentTone
}

// IsWordChar checks whether a rune is considered part of a word.
func IsWordChar(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
}

// ExtractLastWord finds the start byte/rune offset and substring of the current word before caret.
func ExtractLastWord(s string) (startRuneIdx int, word string) {
	runes := []rune(s)
	if len(runes) == 0 {
		return 0, ""
	}

	end := len(runes)
	start := end
	for start > 0 && IsWordChar(runes[start-1]) {
		start--
	}

	return start, string(runes[start:end])
}
