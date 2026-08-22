package solstice

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// DistanceMetric specifies how distance ranges are calculated for targeting.
type DistanceMetric int

const (
	DistanceSquare  DistanceMetric = iota // Chebyshev / square distance: max(|dx|, |dy|) <= maxRange
	DistanceDiamond                       // Manhattan / diamond distance: |dx| + |dy| <= maxRange
)

// VGAPalette16 contains the standard 16 VGA palette colors.
var VGAPalette16 = []color.Color{
	color.RGBA{R: 0, G: 0, B: 0, A: 255},       // 0: Black
	color.RGBA{R: 0, G: 0, B: 170, A: 255},     // 1: Blue
	color.RGBA{R: 0, G: 170, B: 0, A: 255},     // 2: Green
	color.RGBA{R: 0, G: 170, B: 170, A: 255},   // 3: Cyan
	color.RGBA{R: 170, G: 0, B: 0, A: 255},     // 4: Red
	color.RGBA{R: 170, G: 0, B: 170, A: 255},   // 5: Magenta
	color.RGBA{R: 170, G: 85, B: 0, A: 255},    // 6: Brown / Dark Yellow
	color.RGBA{R: 170, G: 170, B: 170, A: 255}, // 7: Light Gray
	color.RGBA{R: 85, G: 85, B: 85, A: 255},    // 8: Dark Gray
	color.RGBA{R: 85, G: 85, B: 255, A: 255},   // 9: Bright Blue
	color.RGBA{R: 85, G: 255, B: 85, A: 255},   // 10: Bright Green
	color.RGBA{R: 85, G: 255, B: 255, A: 255},  // 11: Bright Cyan
	color.RGBA{R: 255, G: 85, B: 85, A: 255},   // 12: Bright Red
	color.RGBA{R: 255, G: 255, B: 85, A: 255},  // 13: Bright Magenta
	color.RGBA{R: 255, G: 255, B: 85, A: 255},  // 14: Yellow
	color.RGBA{R: 255, G: 255, B: 255, A: 255}, // 15: Bright White
}

type TargetSelectedCallback func(targetX, targetY int) bool
type TargetCanceledCallback func()

// TargetMode encapsulates spatial tile targeting with range limits, distance metric options,
// keyboard controls, and color-cycling border cursor rendering.
type TargetMode struct {
	centerX        int
	centerY        int
	maxRange       int
	metric         DistanceMetric
	cursorX        int
	cursorY        int
	colorIdx       int
	highlightTiles map[image.Point]bool
	highlightColor color.Color
	onSelected     TargetSelectedCallback
	onCanceled     TargetCanceledCallback
}

// NewTargetMode creates a new TargetMode with the specified centerpoint, max range, distance metric, and callbacks.
// If maxRange <= 0, targeting mode operates with unlimited range across the entire map.
func NewTargetMode(centerX, centerY, maxRange int, metric DistanceMetric, onSelected TargetSelectedCallback, onCanceled TargetCanceledCallback) *TargetMode {
	return &TargetMode{
		centerX:    centerX,
		centerY:    centerY,
		maxRange:   maxRange,
		metric:     metric,
		cursorX:    centerX,
		cursorY:    centerY,
		onSelected: onSelected,
		onCanceled: onCanceled,
	}
}

// SetHighlightTiles assigns a set of reachable/targetable tile positions to highlight with the specified overlay color.
func (tm *TargetMode) SetHighlightTiles(tiles map[image.Point]bool, col color.Color) {
	tm.highlightTiles = tiles
	if col != nil {
		tm.highlightColor = col
	} else {
		tm.highlightColor = color.RGBA{R: 0, G: 127, B: 0, A: 15}
	}
}

func (tm *TargetMode) Update(g *Game) error {
	// Advance animation ticker
	UpdateAnimTicker()

	// Cycle cursor border color index every frame between the first 16 VGA palette colors
	tm.colorIdx = (tm.colorIdx + 1) % len(VGAPalette16)

	// Handle cursor movement using standard movement keys (WASD, Arrow keys, HJKL)
	dx, dy := 0, 0
	if inpututil.IsKeyJustPressed(ebiten.KeyW) || inpututil.IsKeyJustPressed(ebiten.KeyUp) || inpututil.IsKeyJustPressed(ebiten.KeyK) {
		dy = -1
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyS) || inpututil.IsKeyJustPressed(ebiten.KeyDown) || inpututil.IsKeyJustPressed(ebiten.KeyJ) {
		dy = 1
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyA) || inpututil.IsKeyJustPressed(ebiten.KeyLeft) || inpututil.IsKeyJustPressed(ebiten.KeyH) {
		dx = -1
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyD) || inpututil.IsKeyJustPressed(ebiten.KeyRight) || inpututil.IsKeyJustPressed(ebiten.KeyL) {
		dx = 1
	}

	if dx != 0 || dy != 0 {
		newX := tm.cursorX + dx
		newY := tm.cursorY + dy

		distX := newX - tm.centerX
		if distX < 0 {
			distX = -distX
		}
		distY := newY - tm.centerY
		if distY < 0 {
			distY = -distY
		}

		inRange := false
		if tm.maxRange <= 0 {
			// Range 0 / non-positive indicates unlimited range
			inRange = true
		} else if tm.metric == DistanceDiamond {
			// Manhattan / diamond distance: |dx| + |dy| <= maxRange
			inRange = (distX + distY) <= tm.maxRange
		} else {
			// Square / Chebyshev distance: max(|dx|, |dy|) <= maxRange
			inRange = (distX <= tm.maxRange) && (distY <= tm.maxRange)
		}

		if inRange {
			if g.currentMap != nil && newX >= 0 && newX < g.currentMap.Width && newY >= 0 && newY < g.currentMap.Height {
				tm.cursorX = newX
				tm.cursorY = newY
				UpdateTopViewCenter(image.Pt(tm.cursorX, tm.cursorY))
			}
		}
	}

	// Handle terminal scrolling during targeting mode
	if g.terminal != nil {
		g.terminal.HandleInput()
	}

	// Toggle map scale on Z key press
	if inpututil.IsKeyJustPressed(ebiten.KeyZ) {
		if g.mapScale == 2 {
			g.mapScale = 1
		} else {
			g.mapScale = 2
		}
		SetMapScale(g.mapScale)
	}

	// Confirm target selection on Enter, Keypad Enter, or Space bar
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeyKPEnter) || inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		if tm.onSelected != nil {
			cb := tm.onSelected
			g.PopMode()
			if !cb(tm.cursorX, tm.cursorY) {
				g.PushMode(tm)
			}
		} else {
			g.PopMode()
		}
		return nil
	}

	// Cancel targeting on Escape
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		g.PopMode()
		if tm.onCanceled != nil {
			tm.onCanceled()
		}
		return nil
	}

	return nil
}

func (tm *TargetMode) Draw(g *Game, screen *ebiten.Image) {
	scale := g.mapScale
	if scale == 0 {
		scale = 2
	}

	visCenterX := 16
	visCenterY := 16
	if g.party != nil {
		if IsInCombat() && len(g.party.Members) > 0 {
			curIdx := GetCombatMemberIndex()
			if curIdx >= len(g.party.Members) {
				curIdx = 0
			}
			visCenterX = g.party.Members[curIdx].X
			visCenterY = g.party.Members[curIdx].Y
		} else {
			visCenterX = g.party.X
			visCenterY = g.party.Y
		}
	}

	// 1. Draw the map view area centered on the top of the view stack (targeting cursor), with visibility field centered on party/combat member and highlight overlays
	UpdateTopViewCenter(image.Pt(tm.cursorX, tm.cursorY))
	center := GetViewCenter()
	if g.currentMap != nil {
		g.currentMap.DrawCenteredAt(screen, g.assets, g.party, scale, center.X, center.Y, visCenterX, visCenterY, tm.highlightTiles, tm.highlightColor)
	}

	// 2. Render targeting cursor border at screen center
	DrawTargetCursor(screen, scale, tm.colorIdx)

	// 3. Draw common UI (party roster area and terminal UI)
	DrawCommonUI(screen, g.assets, g.party, g.terminal)
}

// DrawTargetCursor renders the color-cycling targeting cursor border at screen center on top of the map view area.
func DrawTargetCursor(screen *ebiten.Image, scale int, colorIdx int) {
	if screen == nil {
		return
	}
	if scale == 0 {
		scale = 2
	}

	centerStx := 5
	centerSty := 5
	if scale == 1 {
		centerStx = 11
		centerSty = 11
	}

	var px, py float32
	if scale == 2 {
		px = float32(centerStx * 32)
		py = float32(centerSty * 32)
	} else {
		// Account for -8, -8 pixel offset when scale is 1
		px = float32(-8 + centerStx*16)
		py = float32(-8 + centerSty*16)
	}

	sz := float32(16 * scale)
	bw := float32(scale) // Border width: 2px at scale 2, 1px at scale 1

	mapView := screen.SubImage(image.Rect(0, 0, 352, 352)).(*ebiten.Image)
	cursorColor := VGAPalette16[colorIdx%len(VGAPalette16)]

	// Border rectangle around the target tile with thickness bw
	vector.FillRect(mapView, px, py, sz, bw, cursorColor, false)      // Top
	vector.FillRect(mapView, px, py+sz-bw, sz, bw, cursorColor, false) // Bottom
	vector.FillRect(mapView, px, py, bw, sz, cursorColor, false)      // Left
	vector.FillRect(mapView, px+sz-bw, py, bw, sz, cursorColor, false) // Right
}
