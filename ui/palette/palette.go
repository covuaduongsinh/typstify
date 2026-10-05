package palette

import (
	"fmt"
	"image/color"
	"slices"

	"github.com/oligo/gioview/misc"
	th "github.com/oligo/gioview/theme"
)

type SemanticType int

const (
	SemanticError SemanticType = iota
	SemanticWarning
	SemanticSuccess
	SemanticInfo
)

type SemanticPalette struct {
	Error   color.NRGBA
	Warning color.NRGBA
	Success color.NRGBA
	Info    color.NRGBA
}

type UIPalette struct {
	th.Palette
	// chroma style name, see https://xyproto.github.io/splash/docs/ for the full list
	CodeColorScheme string
	Semantic        SemanticPalette
}

var themeMap = map[string]UIPalette{
	"Dương Sinh Light": {
		Palette: th.Palette{
			Fg:            misc.HexColor(0x1C2140), // ds-ink
			Bg:            misc.HexColor(0xFAFAFC),
			ContrastFg:    misc.HexColor(0xFFFFFF),
			ContrastBg:    misc.HexColor(0x2B3990), // ds-brand
			Bg2:           misc.HexColor(0xEEF0F8), // ds-brand-soft
			HoverAlpha:    30,
			SelectedAlpha: 50,
		},
		CodeColorScheme: "emacs",
		Semantic: SemanticPalette{
			Error:   misc.HexColor(0xC53030),
			Warning: misc.HexColor(0xC05621), // ds-warn
			Success: misc.HexColor(0x2F855A),
			Info:    misc.HexColor(0x2B3990), // ds-brand
		},
	},

	"Dương Sinh Dark": {
		Palette: th.Palette{
			Fg:            misc.HexColor(0xE2E8F0),
			Bg:            misc.HexColor(0x121733),
			ContrastFg:    misc.HexColor(0xFFFFFF),
			ContrastBg:    misc.HexColor(0x5A67A8), // ds-brand-light
			Bg2:           misc.HexColor(0x1F264D),
			HoverAlpha:    30,
			SelectedAlpha: 55,
		},
		CodeColorScheme: "doom-one",
		Semantic: SemanticPalette{
			Error:   misc.HexColor(0xFC8181),
			Warning: misc.HexColor(0xED8936),
			Success: misc.HexColor(0x68D391),
			Info:    misc.HexColor(0x63B3ED),
		},
	},

	"Default Light": {
		Palette: th.Palette{
			Fg:            misc.HexColor(0x383A42),
			Bg:            misc.HexColor(0xFAFAFA),
			ContrastFg:    misc.HexColor(0xFAFAFA),
			ContrastBg:    misc.HexColor(0x5A5A5A),
			Bg2:           misc.HexColor(0xF0F0F0),
			HoverAlpha:    30,
			SelectedAlpha: 50,
		},
		CodeColorScheme: "emacs",
		Semantic: SemanticPalette{
			Error:   misc.HexColor(0xD32F2F),
			Warning: misc.HexColor(0xF57C00),
			Success: misc.HexColor(0x388E3C),
			Info:    misc.HexColor(0x1976D2),
		},
	},

	"Default Dark": {
		Palette: th.Palette{
			Fg:            misc.HexColor(0xABB2BF),
			Bg:            misc.HexColor(0x282C34),
			ContrastFg:    misc.HexColor(0xABB2BF),
			ContrastBg:    misc.HexColor(0x5C6370), // Muted gray
			Bg2:           misc.HexColor(0x21252B),
			HoverAlpha:    36,
			SelectedAlpha: 60,
		},
		CodeColorScheme: "doom-one",
		Semantic: SemanticPalette{
			Error:   misc.HexColor(0xE57373),
			Warning: misc.HexColor(0xFFB74D),
			Success: misc.HexColor(0x81C784),
			Info:    misc.HexColor(0x64B5F6),
		},
	},

	"Solarized Light": {
		Palette: th.Palette{
			Fg:            misc.HexColor(0x657b83),
			Bg:            misc.HexColor(0xfdf6e3),
			ContrastFg:    misc.HexColor(0xfdf6e3),
			ContrastBg:    misc.HexColor(0xCB4B16),
			Bg2:           misc.HexColor(0xeee8d5),
			HoverAlpha:    30,
			SelectedAlpha: 40,
		},
		CodeColorScheme: "solarized-light",
		Semantic: SemanticPalette{
			Error:   misc.HexColor(0xDC322F),
			Warning: misc.HexColor(0xCB4B16),
			Success: misc.HexColor(0x859900),
			Info:    misc.HexColor(0x268BD2),
		},
	},

	"Solarized Dark": {
		Palette: th.Palette{
			Fg:            misc.HexColor(0x839496), // Softer foreground
			Bg:            misc.HexColor(0x002b36), // Deep navy
			ContrastFg:    misc.HexColor(0x073642), // Slightly lighter
			ContrastBg:    misc.HexColor(0x268BD2), // Solarized blue for accent
			Bg2:           misc.HexColor(0x073642), // Lighter navy for surfaces
			HoverAlpha:    25,
			SelectedAlpha: 40,
		},
		CodeColorScheme: "solarized-dark",
		Semantic: SemanticPalette{
			Error:   misc.HexColor(0xDC322F),
			Warning: misc.HexColor(0xCB4B16),
			Success: misc.HexColor(0x859900),
			Info:    misc.HexColor(0x268BD2),
		},
	},

	// Nord - Nordic frost palette
	"Nord Light": {
		Palette: th.Palette{
			Fg:            misc.HexColor(0x2E3440),
			Bg:            misc.HexColor(0xECEFF4),
			ContrastFg:    misc.HexColor(0xD8DEE9),
			ContrastBg:    misc.HexColor(0x5E81AC),
			Bg2:           misc.HexColor(0xE5E9F0),
			HoverAlpha:    30,
			SelectedAlpha: 50,
		},
		CodeColorScheme: "monokailight",
		Semantic: SemanticPalette{
			Error:   misc.HexColor(0xBF616A),
			Warning: misc.HexColor(0xD08770),
			Success: misc.HexColor(0xA3BE8C),
			Info:    misc.HexColor(0x5E81AC),
		},
	},

	"Nord Dark": {
		Palette: th.Palette{
			Fg:            misc.HexColor(0xD8DEE9),
			Bg:            misc.HexColor(0x2E3440),
			ContrastFg:    misc.HexColor(0xECEFF4),
			ContrastBg:    misc.HexColor(0x88C0D0), // Aurora cyan
			Bg2:           misc.HexColor(0x3B4252),
			HoverAlpha:    25,
			SelectedAlpha: 60,
		},
		CodeColorScheme: "nord",
		Semantic: SemanticPalette{
			Error:   misc.HexColor(0xBF616A),
			Warning: misc.HexColor(0xD08770),
			Success: misc.HexColor(0xA3BE8C),
			Info:    misc.HexColor(0x88C0D0),
		},
	},

	// Dracula - Vibrant purple/pink
	"Dracula": {
		Palette: th.Palette{
			Fg:            misc.HexColor(0xF8F8F2),
			Bg:            misc.HexColor(0x282A36),
			ContrastFg:    misc.HexColor(0x282A36),
			ContrastBg:    misc.HexColor(0xBD93F9),
			Bg2:           misc.HexColor(0x343746),
			HoverAlpha:    35,
			SelectedAlpha: 55,
		},
		CodeColorScheme: "dracula",
		Semantic: SemanticPalette{
			Error:   misc.HexColor(0xFF5555),
			Warning: misc.HexColor(0xFFB86C),
			Success: misc.HexColor(0x50FA7B),
			Info:    misc.HexColor(0x8BE9FD),
		},
	},

	// Gruvbox - Retro warm palette
	"Gruvbox Light": {
		Palette: th.Palette{
			Fg:            misc.HexColor(0x3C3C3C),
			Bg:            misc.HexColor(0xFBF1C7),
			ContrastFg:    misc.HexColor(0xF9F5D7),
			ContrastBg:    misc.HexColor(0x9D0000),
			Bg2:           misc.HexColor(0xEBDBB2),
			HoverAlpha:    35,
			SelectedAlpha: 55,
		},
		CodeColorScheme: "gruvbox-light",
		Semantic: SemanticPalette{
			Error:   misc.HexColor(0xCC241D),
			Warning: misc.HexColor(0xD65D0E),
			Success: misc.HexColor(0x98971A),
			Info:    misc.HexColor(0x458588),
		},
	},

	"Gruvbox Dark": {
		Palette: th.Palette{
			Fg:            misc.HexColor(0xEBDBB2),
			Bg:            misc.HexColor(0x282828),
			ContrastFg:    misc.HexColor(0x1D2021),
			ContrastBg:    misc.HexColor(0xCC241D),
			Bg2:           misc.HexColor(0x32302F),
			HoverAlpha:    30,
			SelectedAlpha: 45,
		},
		CodeColorScheme: "gruvbox-dark",
		Semantic: SemanticPalette{
			Error:   misc.HexColor(0xFB4934),
			Warning: misc.HexColor(0xFE8019),
			Success: misc.HexColor(0xB8BB26),
			Info:    misc.HexColor(0x83A598),
		},
	},
}

func ThemeNames() []string {
	var names []string
	for k := range themeMap {
		names = append(names, k)
	}

	priorityOrder := map[string]int{
		"Dương Sinh Light": 0,
		"Dương Sinh Dark":  1,
		"Default Light":    2,
		"Default Dark":     3,
	}

	slices.SortFunc(names, func(a, b string) int {
		pa, hasA := priorityOrder[a]
		pb, hasB := priorityOrder[b]
		if hasA && hasB {
			return pa - pb
		}
		if hasA {
			return -1
		}
		if hasB {
			return 1
		}
		if a < b {
			return -1
		} else if a > b {
			return 1
		}
		return 0
	})

	return names
}

func ThemeConfig(themeName string) (UIPalette, error) {
	p, ok := themeMap[themeName]
	if !ok {
		return UIPalette{}, fmt.Errorf("no theme found for name: %s", themeName)
	}

	return p, nil
}

// GetSemanticColor resolves semantic color for given theme or falls back gracefully
func GetSemanticColor(t *th.Theme, st SemanticType) color.NRGBA {
	if t != nil {
		if semVal, ok := t.Get("semanticPalette").(SemanticPalette); ok {
			switch st {
			case SemanticError:
				return semVal.Error
			case SemanticWarning:
				return semVal.Warning
			case SemanticSuccess:
				return semVal.Success
			case SemanticInfo:
				return semVal.Info
			}
		}
	}

	// Fallback if not registered
	switch st {
	case SemanticError:
		return misc.HexColor(0xD32F2F)
	case SemanticWarning:
		return misc.HexColor(0xF57C00)
	case SemanticSuccess:
		return misc.HexColor(0x388E3C)
	case SemanticInfo:
		return misc.HexColor(0x1976D2)
	default:
		return misc.HexColor(0x5A5A5A)
	}
}
