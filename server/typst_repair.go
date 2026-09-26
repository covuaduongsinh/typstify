package server

import (
	"regexp"
	"strings"
)

// chessbookDefinedFunctions lists every function provided by @local/chessbook:0.1.0.
// If a .typ document re-declares any of these with `#let`, the declaration
// shadows the real library version (whose signature is different) and causes
// compile errors.  repairTypstContent strips those mock definitions.
var chessbookDefinedFunctions = []string{
	"game-header",
	"puzzle-card",
	"eco-header",
	"eco-table",
	"opening-diagram-box",
	"lesson-header",
	"concept-box",
	"teaching-diagram",
	"practice-question",
	"instructor-note",
	"column-diagram",
	"chess-quote",
	"chess-board",
	"chess-book-init",
	"chess-magazine-init",
	"chess-worksheet-init",
	"puzzle-grid-a4",
	"puzzle-grid-16x24",
	"difficulty-stars",
	"upside-down-solutions",
	"render-puzzle-solutions",
	"turn-indicator",
	"turn-box",
	"note-num",
	"nag",
}

var (
	chessbookMockPattern   *regexp.Regexp
	chessFuncCallPattern   *regexp.Regexp
	chessImportLinePattern = regexp.MustCompile(`(?m)^[ \t]*#import[ \t]+["'](?:@local/chessbook:[^"']+|lib/lib\.typ|chess_template\.typ)["'][^\r\n]*\r?\n?`)
)

func init() {
	names := make([]string, len(chessbookDefinedFunctions))
	for i, n := range chessbookDefinedFunctions {
		names[i] = regexp.QuoteMeta(n)
	}
	chessbookMockPattern = regexp.MustCompile(
		`^#let\s+(?:` + strings.Join(names, "|") + `)\s*[\(]`,
	)
	chessFuncCallPattern = regexp.MustCompile(
		`#(?:` + strings.Join(names, "|") + `|w[KQBNRP]|b[KQBNRP])\b`,
	)
}

// stripChessbookMocks removes `#let <name>(...) = { ... }` blocks for any
// function already provided by @local/chessbook.
func stripChessbookMocks(doc string) string {
	lines := strings.Split(doc, "\n")
	type span struct{ start, end int }
	var remove []span

	for i := 0; i < len(lines); {
		trimmed := strings.TrimSpace(lines[i])
		if chessbookMockPattern.MatchString(trimmed) {
			start := i
			depth := 0
			foundOpen := false
			j := i
			for ; j < len(lines); j++ {
				for _, ch := range lines[j] {
					switch ch {
					case '{':
						depth++
						foundOpen = true
					case '}':
						depth--
					}
				}
				if foundOpen && depth <= 0 {
					break
				}
			}
			end := j
			// Trim trailing blank lines.
			for end+1 < len(lines) && strings.TrimSpace(lines[end+1]) == "" {
				end++
			}
			remove = append(remove, span{start, end})
			i = end + 1
		} else {
			i++
		}
	}

	if len(remove) == 0 {
		return doc
	}

	var kept []string
	cursor := 0
	for _, s := range remove {
		kept = append(kept, lines[cursor:s.start]...)
		cursor = s.end + 1
	}
	kept = append(kept, lines[cursor:]...)
	return strings.Join(kept, "\n")
}

// repairTypstContent ensures that a .typ document using chessbook functions:
// 1. Has no inline `#let` mock definitions shadowing the library.
// 2. Has exactly one canonical `#import "@local/chessbook:0.1.0": *` at the very top (line 1).
func repairTypstContent(data []byte) []byte {
	content := string(data)
	hasImport := strings.Contains(content, "@local/chessbook:") || strings.Contains(content, "lib/lib.typ")
	hasFunc := chessFuncCallPattern.MatchString(content)

	if !hasImport && !hasFunc {
		return data
	}

	// 1. Strip mock definitions.
	cleaned := stripChessbookMocks(content)

	// 2. Remove all existing misplaced/duplicate chessbook import lines.
	cleaned = chessImportLinePattern.ReplaceAllString(cleaned, "")
	cleaned = strings.TrimLeft(cleaned, " \t\r\n")

	// 3. Prepend the canonical import at line 1.
	repaired := "#import \"@local/chessbook:0.1.0\": *\n\n" + cleaned
	if repaired != content {
		return []byte(repaired)
	}
	return data
}
