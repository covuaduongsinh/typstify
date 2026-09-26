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
	"difficulty-stars",
	"upside-down-solutions",
	"render-puzzle-solutions",
	"turn-indicator",
	"turn-box",
	"note-num",
	"nag",
}

var chessbookMockPattern *regexp.Regexp

func init() {
	names := make([]string, len(chessbookDefinedFunctions))
	for i, n := range chessbookDefinedFunctions {
		names[i] = regexp.QuoteMeta(n)
	}
	chessbookMockPattern = regexp.MustCompile(
		`^#let\s+(?:` + strings.Join(names, "|") + `)\s*[\(]`,
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

// repairTypstContent ensures that a .typ document using chessbook functions
// has the correct import and no inline mock definitions.
func repairTypstContent(data []byte) []byte {
	content := string(data)
	hasImport := strings.Contains(content, "@local/chessbook:")

	if hasImport {
		repaired := stripChessbookMocks(content)
		if repaired != content {
			return []byte(repaired)
		}
	}
	return data
}
