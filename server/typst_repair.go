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
	"render-puzzle-collection",
	"csv-to-puzzles",
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

var embeddedFenPattern = regexp.MustCompile(`(?s)` + "```" + `(?:chessboard|chess-board|chess_board|fen|chess-fen|chess_fen|diagram|teaching-diagram|board)\s*\r?\n(.*?)\s*` + "```")

func repairEmbeddedChessBlocks(doc string) string {
	return embeddedFenPattern.ReplaceAllStringFunc(doc, func(match string) string {
		sub := embeddedFenPattern.FindStringSubmatch(match)
		if len(sub) < 2 {
			return match
		}
		content := sub[1]
		lines := strings.Split(content, "\n")
		fen := ""
		title := "Thế cờ"
		turn := ""
		caption := ""

		for _, rawLine := range lines {
			line := strings.TrimSpace(rawLine)
			if line == "" {
				continue
			}
			lower := strings.ToLower(line)
			if strings.HasPrefix(lower, "fen:") || strings.HasPrefix(lower, "thế cờ:") {
				idx := strings.Index(line, ":")
				fen = strings.Trim(strings.TrimSpace(line[idx+1:]), `"'`)
				continue
			}
			parts := strings.Fields(line)
			if len(parts) > 0 {
				rows := strings.Split(parts[0], "/")
				if len(rows) == 8 {
					fen = strings.Trim(line, `"'`)
					continue
				}
			}
			if idx := strings.Index(line, ":"); idx > 0 {
				key := strings.ToLower(strings.TrimSpace(line[:idx]))
				val := strings.Trim(strings.TrimSpace(line[idx+1:]), `"'`)
				switch key {
				case "title":
					title = val
				case "turn":
					if val == "b" || val == "black" || val == "đen" {
						turn = "b"
					} else {
						turn = "w"
					}
				case "caption":
					caption = val
				}
			}
		}

		if fen == "" && len(lines) > 0 {
			fen = strings.TrimSpace(lines[0])
		}
		if turn == "" && fen != "" {
			parts := strings.Fields(fen)
			if len(parts) >= 2 && (parts[1] == "w" || parts[1] == "b") {
				turn = parts[1]
			}
		}
		if turn == "" {
			turn = "w"
		}
		if fen == "" {
			fen = "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"
		}

		escapedFen := strings.ReplaceAll(strings.ReplaceAll(fen, `\`, `\\`), `"`, `\"`)
		escapedTitle := strings.ReplaceAll(strings.ReplaceAll(title, `\`, `\\`), `"`, `\"`)
		escapedCap := strings.ReplaceAll(strings.ReplaceAll(caption, `\`, `\\`), `"`, `\"`)

		return `#teaching-diagram(
  "` + escapedFen + `",
  title: "` + escapedTitle + `",
  turn: "` + turn + `",
  size: 16pt,
  caption: "` + escapedCap + `"
)`
	})
}

// repairTypstContent ensures that a .typ document using chessbook functions:
// 1. Has no inline `#let` mock definitions shadowing the library.
// 2. Converts embedded raw markdown chess blocks (```chessboard, ```fen) to #teaching-diagram.
// 3. Has exactly one canonical `#import "@local/chessbook:0.1.0": *` at the very top (line 1).
func repairTypstContent(data []byte) []byte {
	content := string(data)
	hasImport := strings.Contains(content, "@local/chessbook:") || strings.Contains(content, "lib/lib.typ")
	hasFunc := chessFuncCallPattern.MatchString(content)
	hasEmbedded := embeddedFenPattern.MatchString(content)

	if !hasImport && !hasFunc && !hasEmbedded {
		return data
	}

	// 1. Strip mock definitions.
	cleaned := stripChessbookMocks(content)

	// 2. Convert embedded markdown chess blocks.
	cleaned = repairEmbeddedChessBlocks(cleaned)

	// 3. Remove all existing misplaced/duplicate chessbook import lines.
	cleaned = chessImportLinePattern.ReplaceAllString(cleaned, "")
	cleaned = strings.TrimLeft(cleaned, " \t\r\n")

	// 4. Prepend the canonical import at line 1.
	repaired := "#import \"@local/chessbook:0.1.0\": *\n\n" + cleaned
	if repaired != content {
		return []byte(repaired)
	}
	return data
}
