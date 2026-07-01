package music

import (
	"fmt"
	"strings"
	"unicode"
)

type Key struct {
	Root  string
	Minor bool
}

func (k Key) String() string {
	if k.Minor {
		return k.Root + "m"
	}
	return k.Root
}

var enharmonicMap = map[string]string{
	"D#": "E♭",
	"G#": "A♭",
	"A#": "B♭",
}

var validRoots = map[string]bool{
	"C": true, "C#": true, "D♭": true,
	"D": true, "E♭": true, "E": true,
	"F": true, "F#": true, "G♭": true,
	"G": true, "A♭": true, "A": true,
	"B♭": true, "B": true,
}

var keyboardRoots = []string{
	"C", "C#", "D♭", "D", "E♭", "E",
	"F", "F#", "G♭", "G", "A♭", "A", "B♭", "B",
}

func normalizeRoot(root string) string {
	upper := strings.ToUpper(root[:1]) + root[1:]

	normalized := strings.ReplaceAll(upper, "b", "♭")

	if repl, ok := enharmonicMap[normalized]; ok {
		return repl
	}

	return normalized
}

func Parse(s string) (Key, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return Key{}, fmt.Errorf("пустая тональность")
	}

	minor := false
	if strings.HasSuffix(strings.ToLower(s), "m") {
		minor = true
		s = s[:len(s)-1]
	}

	if len(s) == 0 {
		return Key{}, fmt.Errorf("некорректная тональность")
	}

	first := unicode.ToUpper(rune(s[0]))
	if first < 'A' || first > 'G' {
		return Key{}, fmt.Errorf("некорректная нота: %c", first)
	}

	root := string(first)
	rest := s[1:]

	rest = strings.ReplaceAll(rest, "♭", "b")

	switch rest {
	case "":
	case "#":
		root += "#"
	case "b":
		root += "♭"
	default:
		return Key{}, fmt.Errorf("некорректный модификатор: %s", rest)
	}

	if repl, ok := enharmonicMap[root]; ok {
		root = repl
	}

	if !validRoots[root] {
		return Key{}, fmt.Errorf("некорректная тональность: %s", root)
	}

	return Key{Root: root, Minor: minor}, nil
}

func KeyboardRows() [][2]string {
	rows := make([][2]string, len(keyboardRoots))
	for i, root := range keyboardRoots {
		rows[i] = [2]string{root, root + "m"}
	}
	return rows
}
