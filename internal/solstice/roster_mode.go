package solstice

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
)

// RosterMode encapsulates interactive party roster state and rendering.
type RosterMode struct{}

// NewRosterMode creates a new RosterMode instance.
func NewRosterMode() *RosterMode {
	return &RosterMode{}
}

func (rm *RosterMode) Update(g *Game) error {
	UpdateAnimTicker()
	return nil
}

func (rm *RosterMode) Draw(g *Game, screen *ebiten.Image) {
	scale := g.mapScale
	if scale == 0 {
		scale = 2
	}

	if g.currentMap != nil {
		g.currentMap.DrawCentered(screen, g.assets, g.party, scale)
	}

	DrawCommonUI(screen, g.assets, g.party, g.terminal)
}

// formatCommas formats an integer with thousands separator commas (e.g. 2000 -> "2,000", 65000 -> "65,000").
func formatCommas(n int) string {
	in := strconv.Itoa(n)
	out := ""
	for i, c := range in {
		if i > 0 && (len(in)-i)%3 == 0 {
			out += ","
		}
		out += string(c)
	}
	return out
}

// DrawPartyRoster renders the party roster area (top-left 352,0; 160x128 pixels) for up to 4 members.
// Uses bright VGA colors for text fields (VGA 9 Bright Blue, VGA 15 Bright White, VGA 12 Bright Red, VGA 11 Bright Cyan).
func (a *Assets) DrawPartyRoster(dst *ebiten.Image, p *Party) {
	if a == nil || p == nil || p.IsSpiritMode() {
		return
	}

	for i, member := range p.Members {
		if i >= MaxPartyMembers {
			break
		}

		// 1. Un-animated sprite (first frame) scaled by 2 at pixel location (352, 32*i)
		a.DrawTileScaled(dst, member.SpriteDef.Tile, 352, float64(32*i), 2.0)

		// 2. Line 1: First 16 characters of member's name in upper-case Bright Blue (VGA 9)
		nameStr := strings.ToUpper(member.Name)
		if len([]rune(nameStr)) > 16 {
			nameStr = string([]rune(nameStr)[:16])
		}
		a.DrawString8x8Colored(dst, nameStr, 48, 4*i+0, VGAPalette16[9])

		// 3. Line 2: "LVL l EXP" left-aligned; experience points right-aligned in Bright White (VGA 15)
		lvlLabel := fmt.Sprintf("LVL %d EXP", member.Level)
		expStr := formatCommas(member.Experience)
		a.DrawString8x8Colored(dst, lvlLabel, 48, 4*i+1, VGAPalette16[15])

		expCol := 48 + 16 - len([]rune(expStr))
		a.DrawString8x8Colored(dst, expStr, expCol, 4*i+1, VGAPalette16[15])

		// 4. Line 3: "hhh/HHH" in Bright Red (VGA 12) left-aligned; "mmm/MMM" in Bright Cyan (VGA 11) right-aligned
		hpStr := fmt.Sprintf("%d/%d", member.HitPoints, member.MaxHitPoints)
		mpStr := fmt.Sprintf("%d/%d", member.MagicPoints, member.MaxMagicPoints)
		a.DrawString8x8Colored(dst, hpStr, 48, 4*i+2, VGAPalette16[12])

		mpCol := 48 + 16 - len([]rune(mpStr))
		a.DrawString8x8Colored(dst, mpStr, mpCol, 4*i+2, VGAPalette16[11])

		// 5. Line 4: Placeholder "Normal" in Bright White (VGA 15) centered within 16-character line
		statusStr := "Normal"
		startCol := 48 + (16-len([]rune(statusStr)))/2
		a.DrawString8x8Colored(dst, statusStr, startCol, 4*i+3, VGAPalette16[15])
	}
}

// DrawPartyRoster renders the party roster area using defaultAssets.
func DrawPartyRoster(dst *ebiten.Image, p *Party) {
	if defaultAssets != nil {
		defaultAssets.DrawPartyRoster(dst, p)
	}
}
