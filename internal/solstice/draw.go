package solstice

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	_ "image/png"
	"strings"
	"solstice/data"

	"github.com/bits-and-blooms/bitset"
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
		if defaultAssets != nil {
			UpdateAnimatedWaterTiles(defaultAssets, defaultTileSet, globalAnimFrame)
		}
	}
}

// GetAnimFrame returns the current state of the global animation frame ticker.
func GetAnimFrame() int {
	return globalAnimFrame
}

// SetAnimFrame sets the global animation frame ticker value (useful for tests).
func SetAnimFrame(frame int) {
	globalAnimFrame = frame
	if defaultAssets != nil {
		UpdateAnimatedWaterTiles(defaultAssets, defaultTileSet, globalAnimFrame)
	}
}

// Assets holds graphical assets for Solstice.
type Assets struct {
	FontIBM8x8     *ebiten.Image
	FontIBM16x12   *ebiten.Image
	FontRune8x8    *ebiten.Image
	FontRune16x12  *ebiten.Image
	Tiles16        *ebiten.Image
	OriginalTiles  *image.RGBA
	CurrentTiles   *image.RGBA
	awtLastFrame   int

	blackTile8x8   *ebiten.Image
	blackTile16x16 *ebiten.Image
}

var defaultAssets *Assets

// LoadAssets loads all required images from the data.FS embedded filesystem.
func LoadAssets() (*Assets, error) {
	ibm8x8, err := loadImage("gfx/IBM.CH.png")
	if err != nil {
		return nil, err
	}

	ibm16x12, err := loadImage("gfx/IBM.HCS.png")
	if err != nil {
		return nil, err
	}

	rune8x8, err := loadImage("gfx/RUNES.CH.png")
	if err != nil {
		return nil, err
	}

	rune16x12, err := loadImage("gfx/RUNES.HCS.png")
	if err != nil {
		return nil, err
	}

	rawTiles, err := loadRGBA("gfx/TILES.16.png")
	if err != nil {
		return nil, err
	}
	currentTiles := image.NewRGBA(rawTiles.Bounds())
	copy(currentTiles.Pix, rawTiles.Pix)

	tiles16 := ebiten.NewImageFromImage(currentTiles)

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
		OriginalTiles:  rawTiles,
		CurrentTiles:   currentTiles,
		awtLastFrame:   -1,
		blackTile8x8:   blackTile8,
		blackTile16x16: blackTile16,
	}

	defaultAssets = assets
	UpdateAnimatedWaterTiles(assets, defaultTileSet, globalAnimFrame)
	return assets, nil
}

func loadRGBA(name string) (*image.RGBA, error) {
	cleanPath := strings.TrimPrefix(name, "data/")
	if !strings.HasPrefix(cleanPath, "gfx/") {
		cleanPath = "gfx/" + cleanPath
	}

	b, err := data.FS.ReadFile(cleanPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read embedded asset %s: %w", name, err)
	}
	img, _, err := image.Decode(bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("failed to decode image %s: %w", name, err)
	}
	bounds := img.Bounds()
	rgba := image.NewRGBA(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			rgba.Set(x, y, img.At(x, y))
		}
	}
	return rgba, nil
}

func loadImage(name string) (*ebiten.Image, error) {
	cleanPath := strings.TrimPrefix(name, "data/")
	if !strings.HasPrefix(cleanPath, "gfx/") {
		cleanPath = "gfx/" + cleanPath
	}

	b, err := data.FS.ReadFile(cleanPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read embedded asset %s: %w", name, err)
	}
	img, _, err := image.Decode(bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("failed to decode image %s: %w", name, err)
	}
	return ebiten.NewImageFromImage(img), nil
}

// UpdateAnimatedWaterTiles modifies the tileset image directly according to Ultima 5 AWT rules for the given animation frame.
func UpdateAnimatedWaterTiles(assets *Assets, ts *TileSet, animFrame int) {
	if assets == nil || assets.OriginalTiles == nil || assets.CurrentTiles == nil || assets.Tiles16 == nil {
		return
	}

	if ts == nil {
		ts = defaultTileSet
	}
	if ts == nil {
		return
	}

	// Step 0: Copy original unaltered tileset to current working copy
	copy(assets.CurrentTiles.Pix, assets.OriginalTiles.Pix)

	// Step 1: Update awt_basic tiles first. Every pixel row is moved down, last row moved to top.
	shift := (animFrame % 16 + 16) % 16
	for tileID, props := range ts.Properties {
		if props.AWTBasic {
			tileCol := tileID % 16
			tileTRow := tileID / 16
			for dstRow := 0; dstRow < 16; dstRow++ {
				srcRow := (dstRow - shift + 16) % 16
				dstOffset := ((tileTRow*16 + dstRow) * assets.CurrentTiles.Stride) + (tileCol * 16 * 4)
				srcOffset := ((tileTRow*16 + srcRow) * assets.OriginalTiles.Stride) + (tileCol * 16 * 4)
				copy(assets.CurrentTiles.Pix[dstOffset:dstOffset+16*4], assets.OriginalTiles.Pix[srcOffset:srcOffset+16*4])
			}
		}
	}

	// Step 2: Update masked tiles.
	for tileID, props := range ts.Properties {
		var maskTileID int
		var waterTileID int
		var isWhiteMask bool
		var isMasked bool

		if props.AWTMaskTL {
			maskTileID = 210
			waterTileID = 3
			isWhiteMask = true
			isMasked = true
		} else if props.AWTMaskTR {
			maskTileID = 211
			waterTileID = 3
			isWhiteMask = true
			isMasked = true
		} else if props.AWTMaskBL {
			maskTileID = 209
			waterTileID = 3
			isWhiteMask = true
			isMasked = true
		} else if props.AWTMaskBR {
			maskTileID = 208
			waterTileID = 3
			isWhiteMask = true
			isMasked = true
		} else if props.AWTMaskRiver {
			maskTileID = tileID + 16
			waterTileID = 2
			isWhiteMask = false // black mask mode
			isMasked = true
		}

		if !isMasked {
			continue
		}

		tileCol := tileID % 16
		tileTRow := tileID / 16
		maskCol := maskTileID % 16
		maskTRow := maskTileID / 16
		waterCol := waterTileID % 16
		waterTRow := waterTileID / 16

		for y := 0; y < 16; y++ {
			for x := 0; x < 16; x++ {
				maskOffset := (((maskTRow*16 + y) * assets.OriginalTiles.Stride) + ((maskCol*16 + x) * 4))
				mr := assets.OriginalTiles.Pix[maskOffset]
				mg := assets.OriginalTiles.Pix[maskOffset+1]
				mb := assets.OriginalTiles.Pix[maskOffset+2]

				srcOffset := (((tileTRow*16 + y) * assets.OriginalTiles.Stride) + ((tileCol*16 + x) * 4))
				waterOffset := (((waterTRow*16 + y) * assets.CurrentTiles.Stride) + ((waterCol*16 + x) * 4))
				dstOffset := (((tileTRow*16 + y) * assets.CurrentTiles.Stride) + ((tileCol*16 + x) * 4))

				if isWhiteMask {
					if mr == 255 && mg == 255 && mb == 255 {
						copy(assets.CurrentTiles.Pix[dstOffset:dstOffset+4], assets.OriginalTiles.Pix[srcOffset:srcOffset+4])
					} else {
						copy(assets.CurrentTiles.Pix[dstOffset:dstOffset+4], assets.CurrentTiles.Pix[waterOffset:waterOffset+4])
					}
				} else { // black mask
					if mr == 0 && mg == 0 && mb == 0 {
						copy(assets.CurrentTiles.Pix[dstOffset:dstOffset+4], assets.OriginalTiles.Pix[srcOffset:srcOffset+4])
					} else {
						copy(assets.CurrentTiles.Pix[dstOffset:dstOffset+4], assets.CurrentTiles.Pix[waterOffset:waterOffset+4])
					}
				}
			}
		}
	}

	assets.Tiles16.WritePixels(assets.CurrentTiles.Pix)
	assets.awtLastFrame = animFrame
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

// DrawGlyph8x8Colored draws a single 8x8 font glyph onto dst at cell coordinates (cellX, cellY) tinted with color c.
func (a *Assets) DrawGlyph8x8Colored(dst *ebiten.Image, glyph int, cellX, cellY int, c color.Color) {
	px := float64(cellX * 8)
	py := float64(cellY * 8)

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(px, py)

	if c != nil {
		r, g, b, aCol := c.RGBA()
		if aCol > 0 {
			op.ColorScale.Scale(float32(r)/65535.0, float32(g)/65535.0, float32(b)/65535.0, float32(aCol)/65535.0)
		}
	}

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

// DrawGlyph8x8Colored draws a single 8x8 font glyph tinted with color c using defaultAssets.
func DrawGlyph8x8Colored(dst *ebiten.Image, glyph int, cellX, cellY int, c color.Color) {
	if defaultAssets != nil {
		defaultAssets.DrawGlyph8x8Colored(dst, glyph, cellX, cellY, c)
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

// DrawString8x8Colored draws a string using the 8x8 font glyph function tinted with color c.
func (a *Assets) DrawString8x8Colored(dst *ebiten.Image, str string, cellX, cellY int, c color.Color) {
	for i, r := range []rune(str) {
		a.DrawGlyph8x8Colored(dst, int(r), cellX+i, cellY, c)
	}
}

// DrawString8x8Colored draws a string tinted with color c using defaultAssets.
func DrawString8x8Colored(dst *ebiten.Image, str string, cellX, cellY int, c color.Color) {
	if defaultAssets != nil {
		defaultAssets.DrawString8x8Colored(dst, str, cellX, cellY, c)
	}
}

// DrawMapTile draws a tile into the map view area (0,0 to 352x352) of dst.
// Output coordinates (tileX, tileY) are screen-relative tile locations:
//   - Scale 2: 11x11 matrix, top-left at (0, 0), top-left corner at (tileX * 32, tileY * 32)
//   - Scale 1: 23x23 matrix, top-left at (-8, -8), top-left corner at (-8 + tileX * 16, -8 + tileY * 16)
// All draw operations are clipped to the sub-image 0,0-351,351 (352x352).
func (a *Assets) DrawMapTile(dst *ebiten.Image, tileIdx int, tileX, tileY int, scale int) {
	a.DrawMapTileHalf(dst, tileIdx, tileX, tileY, scale, false)
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
	a.DrawSpriteDefHalf(dst, sd, screenTileX, screenTileY, scale, false)
}

// DrawSpriteDef draws a SpriteDef onto dst using defaultAssets.
func DrawSpriteDef(dst *ebiten.Image, sd SpriteDef, screenTileX, screenTileY int, scale int) {
	if defaultAssets != nil {
		defaultAssets.DrawSpriteDef(dst, sd, screenTileX, screenTileY, scale)
	}
}

// DrawMapTileHalf draws a tile (or top 8 source pixels if half is true) into the map view area of dst.
func (a *Assets) DrawMapTileHalf(dst *ebiten.Image, tileIdx int, tileX, tileY int, scale int, half bool) {
	if a == nil || a.Tiles16 == nil {
		return
	}

	tileIdx = ResolveAnimatedTile(tileIdx)

	mapArea := dst.SubImage(image.Rect(0, 0, 352, 352)).(*ebiten.Image)

	gx := (tileIdx % 16) * 16
	gy := (tileIdx / 16) * 16

	h := 16
	if half {
		h = 8
	}

	sub := a.Tiles16.SubImage(image.Rect(gx, gy, gx+16, gy+h)).(*ebiten.Image)

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

// DrawMapTileHalf draws a tile (or top 8 source pixels if half is true) using defaultAssets.
func DrawMapTileHalf(dst *ebiten.Image, tileIdx int, tileX, tileY int, scale int, half bool) {
	if defaultAssets != nil {
		defaultAssets.DrawMapTileHalf(dst, tileIdx, tileX, tileY, scale, half)
	}
}

// DrawSpriteDefHalf draws a SpriteDef (or top 8 source pixels if half is true) onto dst at screen tile coordinates.
func (a *Assets) DrawSpriteDefHalf(dst *ebiten.Image, sd SpriteDef, screenTileX, screenTileY int, scale int, half bool) {
	tileIdx := sd.Tile
	if sd.Animated && sd.Frames > 1 {
		tileIdx += (GetAnimFrame() % sd.Frames)
	}
	a.DrawMapTileHalf(dst, tileIdx, screenTileX, screenTileY, scale, half)
}

// DrawSpriteDefHalf draws a SpriteDef (or top 8 source pixels if half is true) onto dst using defaultAssets.
func DrawSpriteDefHalf(dst *ebiten.Image, sd SpriteDef, screenTileX, screenTileY int, scale int, half bool) {
	if defaultAssets != nil {
		defaultAssets.DrawSpriteDefHalf(dst, sd, screenTileX, screenTileY, scale, half)
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

// DrawTileScaled draws a single 16x16 tile from Tiles16 scaled by scale at exact pixel position (px, py) onto dst.
func (a *Assets) DrawTileScaled(dst *ebiten.Image, tileIdx int, px, py float64, scale float64) {
	if a == nil || a.Tiles16 == nil {
		return
	}
	gx := (tileIdx % 16) * 16
	gy := (tileIdx / 16) * 16
	sub := a.Tiles16.SubImage(image.Rect(gx, gy, gx+16, gy+16)).(*ebiten.Image)

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(px, py)
	dst.DrawImage(sub, op)
}

// DrawMiniMap renders a 9x9 tile minimap of the world map centered on the party's world position
// into dst, clipped to the 128x128 pixel region with top-left at (512, 0).
// Visibility is calculated with radius 4. Non-visible tiles are drawn as black void.
func (a *Assets) DrawMiniMap(dst *ebiten.Image, worldMap *Map, p *Party) {
	if a == nil || dst == nil {
		return
	}

	if worldMap == nil {
		worldMap = GetWorldMap()
	}
	if p == nil {
		p = GetParty()
	}

	worldX := 38
	worldY := 103
	if p != nil {
		worldX = p.WorldX
		worldY = p.WorldY
	}

	miniMapArea := dst.SubImage(image.Rect(512, 0, 640, 128)).(*ebiten.Image)

	centerStx := 4
	centerSty := 4

	// Calculate visibility for 9x9 area with radius 4
	var vis *bitset.BitSet
	if worldMap != nil {
		vis = worldMap.CalculateVisibility(worldX, worldY, 4)
	}

	for sty := 0; sty < 9; sty++ {
		for stx := 0; stx < 9; stx++ {
			px := float64(504 + stx*16)
			py := float64(-8 + sty*16)

			isTileVisible := vis != nil && vis.Test(uint(sty*9+stx))
			if isTileVisible {
				mx := worldX + (stx - centerStx)
				my := worldY + (sty - centerSty)

				if worldMap != nil && mx >= 0 && mx < worldMap.Width && my >= 0 && my < worldMap.Height {
					tileIdx := worldMap.GetTile(mx, my)
					a.DrawTileScaled(miniMapArea, tileIdx, px, py, 1.0)
				} else {
					op := &ebiten.DrawImageOptions{}
					op.GeoM.Translate(px, py)
					miniMapArea.DrawImage(a.blackTile16x16, op)
				}
			} else {
				op := &ebiten.DrawImageOptions{}
				op.GeoM.Translate(px, py)
				miniMapArea.DrawImage(a.blackTile16x16, op)
			}
		}
	}
}

// DrawMiniMap renders a mini-map of the world using defaultAssets.
func DrawMiniMap(dst *ebiten.Image, worldMap *Map, party *Party) {
	if defaultAssets != nil {
		defaultAssets.DrawMiniMap(dst, worldMap, party)
	}
}

// DrawCommonUI renders common UI elements visible in all game modes (Party Roster, Mini-Map, and Terminal UI).
func DrawCommonUI(dst *ebiten.Image, assets *Assets, party *Party, terminal *Terminal) {
	if assets != nil && party != nil {
		assets.DrawPartyRoster(dst, party)
	}
	if assets != nil {
		assets.DrawMiniMap(dst, GetWorldMap(), party)
	}
	if terminal != nil && assets != nil {
		terminal.Draw(dst, assets)
	}
}
