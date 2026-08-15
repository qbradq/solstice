package main

/*
extract-fonts is a utility that extracts all 4 font files from the Ultima 5 DOS
release to PNG files.
*/

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"os"
	"path/filepath"
)

var fontPalette = color.Palette{
	color.RGBA{R: 0x00, G: 0x00, B: 0x00, A: 0xff}, // 0: Black
	color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}, // 1: White
}

func main() {
	inPath := flag.String("indir", ".", "Directory containing all Ultima 5 font files.")
	outPath := flag.String("outdir", ".", "Directory to output all .png files to.")
	flag.Parse()

	// Call and error check convertFont for the following files, pulling source
	// files from *inPath and placing output files in *outPath.
	// IBM.CH (small)
	// IBM.HCS (large)
	// RUNES.CH (small)
	// RUNES.HCS (large)
	fontFiles := []struct {
		filename string
		large    bool
	}{
		{"IBM.CH", false},
		{"IBM.HCS", true},
		{"RUNES.CH", false},
		{"RUNES.HCS", true},
	}

	for _, f := range fontFiles {
		src := filepath.Join(*inPath, f.filename)
		dst := filepath.Join(*outPath, f.filename+".png")
		if err := convertFont(src, dst, f.large); err != nil {
			log.Fatalf("failed to convert font %s: %v", f.filename, err)
		}
	}
}

// Converts a single font file into a PNG
func convertFont(inPath, outPath string, large bool) error {
	// On any error, return it.

	// Open *inPath
	// Every file consists of 128 glyphs stored without headers in order in the
	// file. Each glyph is a bitmap. The bytes of the bitmap encode the
	// left-most pixel as the most-significant bit, and the right-most pixel as
	// the least-significant bit.
	data, err := os.ReadFile(inPath)
	if err != nil {
		return fmt.Errorf("failed to open input file %s: %w", inPath, err)
	}

	// If a font file is not "large", each glyph is 8x8 pixels in size and 8
	// bytes long. If a font file is "large", each glyph is 16x12 pixels in size
	// and 24 bytes long.
	glyphWidth := 8
	glyphHeight := 8
	glyphBytes := 8
	imgWidth := 128
	imgHeight := 64

	if large {
		glyphWidth = 16
		glyphHeight = 12
		glyphBytes = 24
		imgWidth = 256
		imgHeight = 96
	}

	if len(data) < 128*glyphBytes {
		return fmt.Errorf("file %s is too short: expected at least %d bytes, got %d", inPath, 128*glyphBytes, len(data))
	}

	// Create the output image. If the font is not "large", the image dimensions
	// will be 128x64. Otherwise, the image will be 256x96.
	img := image.NewPaletted(image.Rect(0, 0, imgWidth, imgHeight), fontPalette)

	for i := range img.Pix {
		img.Pix[i] = 0
	}

	// For each glyph in the font file, extract it to the output image. The
	// output image will consist of a matrix of glyphs 16 wide and 8 tall. Fill
	// the matrix from left-to-right, then top-to-bottom. For example, glyph 18
	// will start at pixel coordinate 16x8 for non "large" fonts, and at 32x12
	// for "large" fonts because it is on the second row and in the third
	// column.
	const glyphsPerRow = 16

	for g := 0; g < 128; g++ {
		tileX := (g % glyphsPerRow) * glyphWidth
		tileY := (g / glyphsPerRow) * glyphHeight
		gData := data[g*glyphBytes : (g+1)*glyphBytes]

		if !large {
			for r := 0; r < 8; r++ {
				b := gData[r]
				for c := 0; c < 8; c++ {
					if (b>>(7-c))&1 != 0 {
						img.SetColorIndex(tileX+c, tileY+r, 1)
					}
				}
			}
		} else {
			for r := 0; r < 12; r++ {
				b0 := gData[r*2]
				b1 := gData[r*2+1]
				for c := 0; c < 8; c++ {
					if (b0>>(7-c))&1 != 0 {
						img.SetColorIndex(tileX+c, tileY+r, 1)
					}
				}
				for c := 0; c < 8; c++ {
					if (b1>>(7-c))&1 != 0 {
						img.SetColorIndex(tileX+8+c, tileY+r, 1)
					}
				}
			}
		}
	}

	// Encode the output image to outPath as a png image.
	outFile, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("failed to create output file %s: %w", outPath, err)
	}

	if err := png.Encode(outFile, img); err != nil {
		outFile.Close()
		return fmt.Errorf("failed to encode PNG image to %s: %w", outPath, err)
	}

	if err := outFile.Close(); err != nil {
		return fmt.Errorf("failed to close output file %s: %w", outPath, err)
	}

	return nil
}
