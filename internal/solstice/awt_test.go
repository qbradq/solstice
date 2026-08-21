package solstice

import (
	"image/color"
	"testing"
)

func TestAnimatedWaterTilesBasic(t *testing.T) {
	ts, err := PreloadTileSet()
	if err != nil {
		t.Fatalf("PreloadTileSet failed: %v", err)
	}

	assets, err := LoadAssets()
	if err != nil {
		t.Fatalf("LoadAssets failed: %v", err)
	}

	// Verify tile 1, 2, 3 have AWTBasic set
	for _, tileID := range []int{1, 2, 3} {
		props := ts.GetTileProperties(tileID)
		if !props.AWTBasic {
			t.Errorf("Expected tile %d to have AWTBasic = true", tileID)
		}
	}

	// Helper to extract a 16-pixel RGBA slice for a given tile and row from RGBA image
	getRowPixels := func(rgba *Assets, isOriginal bool, tileID, row int) []color.RGBA {
		img := rgba.CurrentTiles
		if isOriginal {
			img = rgba.OriginalTiles
		}
		col := tileID % 16
		tRow := tileID / 16
		offset := ((tRow*16 + row) * img.Stride) + (col * 16 * 4)

		pixels := make([]color.RGBA, 16)
		for i := 0; i < 16; i++ {
			pOff := offset + i*4
			pixels[i] = color.RGBA{
				R: img.Pix[pOff],
				G: img.Pix[pOff+1],
				B: img.Pix[pOff+2],
				A: img.Pix[pOff+3],
			}
		}
		return pixels
	}

	// 1. Frame 0: CurrentTiles should match OriginalTiles for awt_basic tiles
	UpdateAnimatedWaterTiles(assets, ts, 0)
	for row := 0; row < 16; row++ {
		origPix := getRowPixels(assets, true, 1, row)
		currPix := getRowPixels(assets, false, 1, row)
		for x := 0; x < 16; x++ {
			if origPix[x] != currPix[x] {
				t.Errorf("Frame 0 mismatch at tile 1 row %d col %d: expected %v, got %v", row, x, origPix[x], currPix[x])
			}
		}
	}

	// 2. Frame 1: Every pixel row moved down, last row moved to top
	// dstRow 0 gets origRow 15; dstRow 1 gets origRow 0; etc.
	UpdateAnimatedWaterTiles(assets, ts, 1)
	for dstRow := 0; dstRow < 16; dstRow++ {
		expectedOrigRow := (dstRow - 1 + 16) % 16
		origPix := getRowPixels(assets, true, 1, expectedOrigRow)
		currPix := getRowPixels(assets, false, 1, dstRow)
		for x := 0; x < 16; x++ {
			if origPix[x] != currPix[x] {
				t.Errorf("Frame 1 mismatch at tile 1 row %d (from orig row %d) col %d: expected %v, got %v",
					dstRow, expectedOrigRow, x, origPix[x], currPix[x])
			}
		}
	}

	// 3. Frame 16: Complete cycle, should match frame 0
	UpdateAnimatedWaterTiles(assets, ts, 16)
	for row := 0; row < 16; row++ {
		origPix := getRowPixels(assets, true, 1, row)
		currPix := getRowPixels(assets, false, 1, row)
		for x := 0; x < 16; x++ {
			if origPix[x] != currPix[x] {
				t.Errorf("Frame 16 mismatch at tile 1 row %d col %d: expected %v, got %v", row, x, origPix[x], currPix[x])
			}
		}
	}
}

func TestAnimatedWaterTilesWhiteMask(t *testing.T) {
	ts, err := PreloadTileSet()
	if err != nil {
		t.Fatalf("PreloadTileSet failed: %v", err)
	}

	assets, err := LoadAssets()
	if err != nil {
		t.Fatalf("LoadAssets failed: %v", err)
	}

	// Test tile 52 (awt_mask_tl, mask 210, water 3)
	// Test tile 53 (awt_mask_tr, mask 211, water 3)
	// Test tile 54 (awt_mask_br, mask 208, water 3)
	// Test tile 55 (awt_mask_bl, mask 209, water 3)
	testCases := []struct {
		tileID     int
		maskTileID int
		name       string
	}{
		{52, 210, "awt_mask_tl"},
		{53, 211, "awt_mask_tr"},
		{54, 208, "awt_mask_br"},
		{55, 209, "awt_mask_bl"},
	}

	getPixel := func(rgba *Assets, isOriginal bool, tileID, x, y int) color.RGBA {
		img := rgba.CurrentTiles
		if isOriginal {
			img = rgba.OriginalTiles
		}
		col := tileID % 16
		tRow := tileID / 16
		pOff := ((tRow*16 + y) * img.Stride) + ((col*16 + x) * 4)
		return color.RGBA{
			R: img.Pix[pOff],
			G: img.Pix[pOff+1],
			B: img.Pix[pOff+2],
			A: img.Pix[pOff+3],
		}
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Update at frame 3
			UpdateAnimatedWaterTiles(assets, ts, 3)

			for y := 0; y < 16; y++ {
				for x := 0; x < 16; x++ {
					maskPix := getPixel(assets, true, tc.maskTileID, x, y)
					srcPix := getPixel(assets, true, tc.tileID, x, y)
					waterPix := getPixel(assets, false, 3, x, y)
					dstPix := getPixel(assets, false, tc.tileID, x, y)

					if maskPix.R == 255 && maskPix.G == 255 && maskPix.B == 255 {
						// White in mask -> source tile pixel
						if dstPix != srcPix {
							t.Errorf("Tile %d at (%d, %d): expected source pixel %v on white mask, got %v",
								tc.tileID, x, y, srcPix, dstPix)
						}
					} else {
						// Not white in mask -> animated water tile 3 pixel
						if dstPix != waterPix {
							t.Errorf("Tile %d at (%d, %d): expected water tile 3 pixel %v on non-white mask, got %v",
								tc.tileID, x, y, waterPix, dstPix)
						}
					}
				}
			}
		})
	}
}

func TestAnimatedWaterTilesBlackMaskRiver(t *testing.T) {
	ts, err := PreloadTileSet()
	if err != nil {
		t.Fatalf("PreloadTileSet failed: %v", err)
	}

	assets, err := LoadAssets()
	if err != nil {
		t.Fatalf("LoadAssets failed: %v", err)
	}

	getPixel := func(rgba *Assets, isOriginal bool, tileID, x, y int) color.RGBA {
		img := rgba.CurrentTiles
		if isOriginal {
			img = rgba.OriginalTiles
		}
		col := tileID % 16
		tRow := tileID / 16
		pOff := ((tRow*16 + y) * img.Stride) + ((col*16 + x) * 4)
		return color.RGBA{
			R: img.Pix[pOff],
			G: img.Pix[pOff+1],
			B: img.Pix[pOff+2],
			A: img.Pix[pOff+3],
		}
	}

	// Test river tiles (e.g. tile 72 through 87)
	for tileID := 72; tileID <= 87; tileID++ {
		props := ts.GetTileProperties(tileID)
		if !props.AWTMaskRiver {
			continue
		}
		maskTileID := tileID + 16

		// Update at frame 5
		UpdateAnimatedWaterTiles(assets, ts, 5)

		for y := 0; y < 16; y++ {
			for x := 0; x < 16; x++ {
				maskPix := getPixel(assets, true, maskTileID, x, y)
				srcPix := getPixel(assets, true, tileID, x, y)
				waterPix := getPixel(assets, false, 2, x, y)
				dstPix := getPixel(assets, false, tileID, x, y)

				if maskPix.R == 0 && maskPix.G == 0 && maskPix.B == 0 {
					// Black in mask -> source tile pixel
					if dstPix != srcPix {
						t.Errorf("River tile %d at (%d, %d): expected source pixel %v on black mask, got %v",
							tileID, x, y, srcPix, dstPix)
					}
				} else {
					// Not black in mask -> animated water tile 2 pixel
					if dstPix != waterPix {
						t.Errorf("River tile %d at (%d, %d): expected water tile 2 pixel %v on non-black mask, got %v",
							tileID, x, y, waterPix, dstPix)
					}
				}
			}
		}
	}
}

func TestAnimTickerAWTUpdates(t *testing.T) {
	_, err := PreloadTileSet()
	if err != nil {
		t.Fatalf("PreloadTileSet failed: %v", err)
	}

	assets, err := LoadAssets()
	if err != nil {
		t.Fatalf("LoadAssets failed: %v", err)
	}

	SetAnimFrame(0)
	if assets.awtLastFrame != 0 {
		t.Errorf("Expected awtLastFrame 0 after SetAnimFrame(0), got %d", assets.awtLastFrame)
	}

	// Advance 15 ticks to trigger frame 1
	for i := 0; i < 15; i++ {
		UpdateAnimTicker()
	}

	if GetAnimFrame() != 1 {
		t.Errorf("Expected globalAnimFrame 1, got %d", GetAnimFrame())
	}
	if assets.awtLastFrame != 1 {
		t.Errorf("Expected awtLastFrame 1, got %d", assets.awtLastFrame)
	}
}
