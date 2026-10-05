package uitokens

import (
	"image/color"

	"gioui.org/unit"
	"github.com/oligo/gioview/theme"
	"looz.ws/typstify/ui/palette"
)

// Standard spacing tokens for layout consistency
const (
	SpacingXXS = unit.Dp(4)
	SpacingXS  = unit.Dp(8)
	SpacingS   = unit.Dp(12)
	SpacingM   = unit.Dp(16)
	SpacingL   = unit.Dp(20)
	SpacingXL  = unit.Dp(24)
	SpacingXXL = unit.Dp(32)
)

// Standard corner radius tokens
const (
	RadiusSmall  = unit.Dp(2)
	RadiusMedium = unit.Dp(4)
	RadiusLarge  = unit.Dp(8)
)

// Semantic colors lookup helpers based on current theme palette
func ErrorColor(th *theme.Theme) color.NRGBA {
	return palette.GetSemanticColor(th, palette.SemanticError)
}

func WarningColor(th *theme.Theme) color.NRGBA {
	return palette.GetSemanticColor(th, palette.SemanticWarning)
}

func SuccessColor(th *theme.Theme) color.NRGBA {
	return palette.GetSemanticColor(th, palette.SemanticSuccess)
}

func InfoColor(th *theme.Theme) color.NRGBA {
	return palette.GetSemanticColor(th, palette.SemanticInfo)
}
