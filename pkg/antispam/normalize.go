package antispam

import "strings"

// confusableGroups maps each ASCII letter to look-alike runes spammers use to
// dodge keyword matching: Latin diacritics, Cyrillic/Greek homoglyphs and the
// IPA script g. Uppercase forms are covered because input is lowercased first.
var confusableGroups = map[rune]string{
	'a': "àáâãäåāăąаα",
	'b': "ьв",
	'c': "çćĉċčс",
	'e': "èéêëēĕėęěеε",
	'g': "ĝğġģɡ",
	'h': "н",
	'i': "ìíîïĩīĭįıіι",
	'j': "ј",
	'k': "кκ",
	'm': "м",
	'n': "ñńņň",
	'o': "òóôõöøōŏőоο",
	'p': "рρ",
	's': "śŝşšѕ",
	't': "тτ",
	'u': "ùúûüũūŭůűųυ",
	'v': "ν",
	'x': "х",
	'y': "ýÿŷу",
	'z': "źżž",
}

var confusables = buildConfusables()

func buildConfusables() map[rune]rune {
	m := make(map[rune]rune)
	for ascii, lookalikes := range confusableGroups {
		for _, r := range lookalikes {
			m[r] = ascii
		}
	}
	return m
}

// leetDigits undoes digit-for-letter swaps ("g4c0r" -> "gacor"). It is only
// applied to tokens that also contain a letter, so plain numbers such as
// "2026" or "98" are left alone.
var leetDigits = map[byte]byte{'0': 'o', '1': 'i', '3': 'e', '4': 'a', '5': 's', '7': 't'}

// normalize reduces text to lowercase ASCII words separated by single spaces
// so obfuscated spam matches the plain keyword list:
//
//   - fullwidth, math-styled and circled letters fold to ASCII ("𝐠𝐚𝐜𝐨𝐫");
//   - homoglyphs and diacritics fold to ASCII ("gаcor" with Cyrillic а);
//   - invisible characters are dropped ("ga​cor");
//   - runs of single characters merge ("g a c o r", "g.a.c.o.r");
//   - leetspeak digits are undone inside words ("g4c0r").
func normalize(text string) string {
	var tokens []string
	var current strings.Builder
	flush := func() {
		if current.Len() > 0 {
			tokens = append(tokens, current.String())
			current.Reset()
		}
	}

	for _, r := range strings.ToLower(text) {
		if isInvisible(r) {
			continue
		}
		if folded := foldRune(r); isASCIIAlnum(folded) {
			current.WriteRune(folded)
		} else {
			flush()
		}
	}
	flush()

	tokens = mergeSingleCharRuns(tokens)
	for i, token := range tokens {
		tokens[i] = undoLeet(token)
	}
	return strings.Join(tokens, " ")
}

// foldRune maps a rune to its plain ASCII look-alike, or returns it unchanged.
// '@' and '$' fold too because they stand in for letters ("g@cor", "$lot").
func foldRune(r rune) rune {
	switch {
	case r == '@':
		return 'a'
	case r == '$':
		return 's'
	case r >= 0xFF10 && r <= 0xFF5A: // fullwidth digits and letters
		return unicodeLower(r - 0xFEE0)
	case r >= 0x1D400 && r <= 0x1D6A3: // mathematical alphanumeric letters
		return 'a' + (r-0x1D400)%52%26
	case r >= 0x1D7CE && r <= 0x1D7FF: // mathematical digits
		return '0' + (r-0x1D7CE)%10
	case r >= 0x24B6 && r <= 0x24E9: // circled letters
		return 'a' + (r-0x24B6)%26
	}
	if ascii, ok := confusables[r]; ok {
		return ascii
	}
	return r
}

func unicodeLower(r rune) rune {
	if r >= 'A' && r <= 'Z' {
		return r + ('a' - 'A')
	}
	return r
}

// isInvisible reports characters that render as nothing (soft hyphen,
// zero-width and bidi controls, BOM) or only decorate the previous letter
// (combining marks). Dropping them glues the surrounding letters back
// together instead of splitting the word.
func isInvisible(r rune) bool {
	switch {
	case r == 0x00AD, r == 0xFEFF:
		return true
	case r >= 0x0300 && r <= 0x036F:
		return true
	case r >= 0x200B && r <= 0x200F, r >= 0x202A && r <= 0x202E, r >= 0x2060 && r <= 0x2064:
		return true
	}
	return false
}

func isASCIIAlnum(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
}

// mergeSingleCharRuns joins consecutive one-character tokens, undoing
// letter-spacing ("g a c o r" -> "gacor") without gluing real words together
// ("mega corp" stays two tokens).
func mergeSingleCharRuns(tokens []string) []string {
	merged := make([]string, 0, len(tokens))
	var run strings.Builder
	flush := func() {
		if run.Len() > 0 {
			merged = append(merged, run.String())
			run.Reset()
		}
	}

	for _, token := range tokens {
		if len(token) == 1 {
			run.WriteString(token)
			continue
		}
		flush()
		merged = append(merged, token)
	}
	flush()
	return merged
}

func undoLeet(token string) string {
	if !containsLetter(token) {
		return token
	}
	b := []byte(token)
	for i, c := range b {
		if letter, ok := leetDigits[c]; ok {
			b[i] = letter
		}
	}
	return string(b)
}

func containsLetter(token string) bool {
	for i := 0; i < len(token); i++ {
		if token[i] >= 'a' && token[i] <= 'z' {
			return true
		}
	}
	return false
}
