package solstice

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	_ "image/png"
	"solstice/data"

	"github.com/hajimehoshi/ebiten/v2"
)

// Global animation frame ticker state
var (
	globalAnimTicks int
	globalAnimFrame int
)

// UpdateAnimTicker advances the global animation frame ticker.
// Call this once per frame in Update().
func UpdateAnimTicker() {
	globalAnimTicks++
	// Advance animation frame every 15 ticks (approx 4 FPS animation rate at 60 FPS update rate)
	if globalAnimTicks >= 15 {
		globalAnimTicks = 0
		globalAnimFrame++
	}
}

// GetAnimFrame returns the current state of the global animation frame ticker.
func GetAnimFrame() int {
	return globalAnimFrame
}

// SetAnimFrame sets the global animation frame ticker value (useful for tests).
func SetAnimFrame(frame int) {
	globalAnimFrame = frame
}

// Assets holds graphical assets for Solstice.
type Assets struct {
	FontIBM8x8     *ebiten.Image
	FontIBM16x12   *ebiten.Image
	FontRune8x8    *ebiten.Image
	FontRune16x12  *ebiten.Image
	Tiles16        *ebiten.Image

	blackTile8x8   *ebiten.Image
	blackTile16x16 *ebiten.Image
}

var defaultAssets *Assets

// LoadAssets loads all required images from the data.FS embedded filesystem.
func LoadAssets() (*Assets, error) {
	ibm8x8, err := loadImage("IBM.CH.png")
	if err != nil {
		return nil, err
	}

	ibm16x12, err := loadImage("IBM.HCS.png")
	if err != nil {
		return nil, err
	}

	rune8x8, err := loadImage("RUNES.CH.png")
	if err != nil {
		return nil, err
	}

	rune16x12, err := loadImage("RUNES.HCS.png")
	if err != nil {
		return nil, err
	}

	tiles16, err := loadImage("TILES.16.png")
	if err != nil {
		return nil, err
	}

	blackTile8 := ebiten.NewImage(8, 8)
	blackTile8.Fill(color.Black)

	blackTile16 := ebiten.NewImage(16, 16)
	blackTile16.Fill(color.Black)

	assets := &Assets{
		FontIBM8x8:     ibm8x8,
		FontIBM16x12:   ibm16x12,
		FontRune8x8:    rune8x8,
		FontRune16x12:  rune16x12,
		Tiles16:        tiles16,
		blackTile8x8:   blackTile8,
		blackTile16x16: blackTile16,
	}

	defaultAssets = assets
	return assets, nil
}

func loadImage(name string) (*ebiten.Image, error) {
	b, err := data.FS.ReadFile(name)
	if err != nil {
		return nil, fmt.Errorf("failed to read embedded asset %s: %w", name, err)
	}
	img, _, err := image.Decode(bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("failed to decode image %s: %w", name, err)
	}
	return ebiten.NewImageFromImage(img), nil
}

// DrawGlyph8x8 draws a single 8x8 font glyph onto dst at cell coordinates (cellX, cellY).
// Cell coordinates map to pixel coordinates as (cellX * 8, cellY * 8).
// Glyphs 0-127 are output from IBM.CH.png.
// Glyphs 128-255 are output from RUNES.CH.png (index minus 128).
// Glyphs outside these ranges fill the output area with black.
func (a *Assets) DrawGlyph8x8(dst *ebiten.Image, glyph int, cellX, cellY int) {
	px := float64(cellX * 8)
	py := float64(cellY * 8)

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(px, py)

	if glyph >= 0 && glyph <= 127 {
		gx := (glyph % 16) * 8
		gy := (glyph / 16) * 8
		sub := a.FontIBM8x8.SubImage(image.Rect(gx, gy, gx+8, gy+8)).(*ebiten.Image)
		dst.DrawImage(sub, op)
	} else if glyph >= 128 && glyph <= 255 {
		idx := glyph - 128
		gx := (idx % 16) * 8
		gy := (idx / 16) * 8
		sub := a.FontRune8x8.SubImage(image.Rect(gx, gy, gx+8, gy+8)).(*ebiten.Image)
		dst.DrawImage(sub, op)
	} else {
		dst.DrawImage(a.blackTile8x8, op)
	}
}

// DrawGlyph8x8 draws a single 8x8 font glyph onto dst using defaultAssets.
func DrawGlyph8x8(dst *ebiten.Image, glyph int, cellX, cellY int) {
	if defaultAssets != nil {
		defaultAssets.DrawGlyph8x8(dst, glyph, cellX, cellY)
	}
}

// DrawString8x8 draws a string using the 8x8 font glyph function.
// Starting output coordinates (cellX, cellY) are given in 8x8 pixel cells.
func (a *Assets) DrawString8x8(dst *ebiten.Image, str string, cellX, cellY int) {
	for i, r := range []rune(str) {
		a.DrawGlyph8x8(dst, int(r), cellX+i, cellY)
	}
}

// DrawString8x8 draws a string using defaultAssets.
func DrawString8x8(dst *ebiten.Image, str string, cellX, cellY int) {
	if defaultAssets != nil {
		defaultAssets.DrawString8x8(dst, str, cellX, cellY)
	}
}

// DrawMapTile draws a tile into the map view area (0,0 to 352x352) of dst.
// Output coordinates (tileX, tileY) are screen-relative tile locations:
//   - Scale 2: 11x11 matrix, top-left at (0, 0), top-left corner at (tileX * 32, tileY * 32)
//   - Scale 1: 23x23 matrix, top-left at (-8, -8), top-left corner at (-8 + tileX * 16, -8 + tileY * 16)
// All draw operations are clipped to the sub-image 0,0-351,351 (352x352).
func (a *Assets) DrawMapTile(dst *ebiten.Image, tileIdx int, tileX, tileY int, scale int) {
	if a == nil || a.Tiles16 == nil {
		return
	}

	mapArea := dst.SubImage(image.Rect(0, 0, 352, 352)).(*ebiten.Image)

	gx := (tileIdx % 16) * 16
	gy := (tileIdx / 16) * 16
	sub := a.Tiles16.SubImage(image.Rect(gx, gy, gx+16, gy+16)).(*ebiten.Image)

	op := &ebiten.DrawImageOptions{}

	if scale == 2 {
		op.GeoM.Scale(2, 2)
		px := float64(tileX * 32)
		py := float64(tileY * 32)
		op.GeoM.Translate(px, py)
	} else {
		px := float64(-8 + tileX*16)
		py := float64(-8 + tileY*16)
		op.GeoM.Translate(px, py)
	}

	mapArea.DrawImage(sub, op)
}

// DrawMapTile draws a tile into the map view area using defaultAssets.
func DrawMapTile(dst *ebiten.Image, tileIdx int, tileX, tileY int, scale int) {
	if defaultAssets != nil {
		defaultAssets.DrawMapTile(dst, tileIdx, tileX, tileY, scale)
	}
}

// DrawBlackMapTile draws a black tile cell into the map view area at screen tile coordinates (tileX, tileY).
func (a *Assets) DrawBlackMapTile(dst *ebiten.Image, tileX, tileY int, scale int) {
	if a == nil || a.blackTile16x16 == nil {
		return
	}

	mapArea := dst.SubImage(image.Rect(0, 0, 352, 352)).(*ebiten.Image)
	op := &ebiten.DrawImageOptions{}

	if scale == 2 {
		op.GeoM.Scale(2, 2)
		px := float64(tileX * 32)
		py := float64(tileY * 32)
		op.GeoM.Translate(px, py)
	} else {
		px := float64(-8 + tileX*16)
		py := float64(-8 + tileY*16)
		op.GeoM.Translate(px, py)
	}

	mapArea.DrawImage(a.blackTile16x16, op)
}

// DrawBlackMapTile draws a black tile cell into the map view area using defaultAssets.
func DrawBlackMapTile(dst *ebiten.Image, tileX, tileY int, scale int) {
	if defaultAssets != nil {
		defaultAssets.DrawBlackMapTile(dst, tileX, tileY, scale)
	}
}

// DrawSpriteDef draws a SpriteDef onto dst at screen tile coordinates (screenTileX, screenTileY)
// using scale and the current state of the global animation frame ticker.
// It calls DrawMapTile to draw graphics onto the map view area, overwriting what was there previously.
func (a *Assets) DrawSpriteDef(dst *ebiten.Image, sd SpriteDef, screenTileX, screenTileY int, scale int) {
	tileIdx := sd.Tile
	if sd.Animated && sd.Frames > 1 {
		tileIdx += (GetAnimFrame() % sd.Frames)
	}
	a.DrawMapTile(dst, tileIdx, screenTileX, screenTileY, scale)
}

// DrawSpriteDef draws a SpriteDef onto dst using defaultAssets.
func DrawSpriteDef(dst *ebiten.Image, sd SpriteDef, screenTileX, screenTileY int, scale int) {
	if defaultAssets != nil {
		defaultAssets.DrawSpriteDef(dst, sd, screenTileX, screenTileY, scale)
	}
}

// FillMapScreen fills the map view area with the specified tile index at 1x or 2x scale.
func (a *Assets) FillMapScreen(dst *ebiten.Image, tileIdx int, scale int) {
	if scale == 2 {
		for ty := 0; ty < 11; ty++ {
			for tx := 0; tx < 11; tx++ {
				a.DrawMapTile(dst, tileIdx, tx, ty, 2)
			}
		}
	} else {
		for ty := 0; ty < 23; ty++ {
			for tx := 0; tx < 23; tx++ {
				a.DrawMapTile(dst, tileIdx, tx, ty, 1)
			}
		}
	}
}

// FillMapScreen fills the map view area using defaultAssets.
func FillMapScreen(dst *ebiten.Image, tileIdx int, scale int) {
	if defaultAssets != nil {
		defaultAssets.FillMapScreen(dst, tileIdx, scale)
	}
}
