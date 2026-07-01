package normalize

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

var latinMinorWords = map[string]bool{
	"a": true, "an": true, "the": true,
	"and": true, "or": true, "but": true, "nor": true,
	"so": true, "yet": true,
	"in": true, "on": true, "at": true, "to": true,
	"for": true, "of": true, "with": true, "by": true,
	"from": true, "as": true, "into": true, "like": true,
	"near": true, "over": true, "upon": true, "via": true,
}

var cyrillicProperNouns = map[string]bool{
	"александр": true, "алексей": true, "андрей": true, "анна": true,
	"антон": true, "артём": true, "борис": true, "вадим": true,
	"валерий": true, "василий": true, "виктор": true, "виталий": true,
	"владимир": true, "вячеслав": true, "геннадий": true, "георгий": true,
	"григорий": true, "дарья": true, "денис": true, "дмитрий": true,
	"евгений": true, "екатерина": true, "елена": true, "иван": true,
	"игорь": true, "ирина": true, "кирилл": true, "константин": true,
	"лариса": true, "леонид": true, "максим": true, "мария": true,
	"марина": true, "михаил": true, "наталья": true, "никита": true,
	"николай": true, "олег": true, "ольга": true, "павел": true,
	"пётр": true, "роман": true, "руслан": true, "светлана": true,
	"сергей": true, "станислав": true, "степан": true, "татьяна": true,
	"тимур": true, "фёдор": true, "юлия": true, "юрий": true,
	"ярослав": true,
}

var cyrillicAbbreviations = map[string]string{
	"ддт": "ДДТ", "би-2": "Би-2", "аукцыон": "АукцЫон",
}

func collapseSpaces(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	prev := ' '
	for _, r := range s {
		if unicode.IsSpace(r) {
			if prev != ' ' {
				b.WriteRune(' ')
			}
			prev = ' '
		} else {
			b.WriteRune(r)
			prev = r
		}
	}
	return strings.TrimSpace(b.String())
}

func isCyrillic(s string) bool {
	cyrillic := 0
	latin := 0
	for _, r := range s {
		if unicode.Is(unicode.Cyrillic, r) {
			cyrillic++
		} else if unicode.Is(unicode.Latin, r) {
			latin++
		}
	}
	return cyrillic > latin
}

func capitalizeCyrillicWord(word string, isFirst bool) string {
	lower := strings.ToLower(word)

	if repl, ok := cyrillicAbbreviations[lower]; ok {
		return repl
	}

	if cyrillicProperNouns[lower] || isFirst {
		r, size := utf8.DecodeRuneInString(lower)
		if r == utf8.RuneError {
			return lower
		}
		return string(unicode.ToUpper(r)) + lower[size:]
	}

	return lower
}

func capitalizeLatinWord(word string, isFirst bool) string {
	lower := strings.ToLower(word)
	if !isFirst && latinMinorWords[lower] {
		return lower
	}
	r, size := utf8.DecodeRuneInString(lower)
	if r == utf8.RuneError {
		return lower
	}
	return string(unicode.ToUpper(r)) + lower[size:]
}

func SongName(raw string) string {
	s := collapseSpaces(raw)
	if s == "" {
		return s
	}

	words := strings.Split(s, " ")
	cyrillic := isCyrillic(s)

	for i, w := range words {
		if cyrillic {
			words[i] = capitalizeCyrillicWord(w, i == 0)
		} else {
			words[i] = capitalizeLatinWord(w, i == 0)
		}
	}

	return strings.Join(words, " ")
}
