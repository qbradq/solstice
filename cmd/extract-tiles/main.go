package main

/*
extract-tiles is a utility program that converts the file TILES.16 from the
Ultima 5 DOS release to a PNG tile set image.
*/

import (
	"compress/lzw"
	"flag"
	"image"
	"image/color"
	"image/png"
	"io"
	"log"
	"os"
)

// vgaPalette represents the first 16 colors of the standard VGA palette.
var vgaPalette = color.Palette{
	color.RGBA{R: 0x00, G: 0x00, B: 0x00, A: 0xff}, // 0: Black
	color.RGBA{R: 0x00, G: 0x00, B: 0xaa, A: 0xff}, // 1: Blue
	color.RGBA{R: 0x00, G: 0xaa, B: 0x00, A: 0xff}, // 2: Green
	color.RGBA{R: 0x00, G: 0xaa, B: 0xaa, A: 0xff}, // 3: Cyan
	color.RGBA{R: 0xaa, G: 0x00, B: 0x00, A: 0xff}, // 4: Red
	color.RGBA{R: 0xaa, G: 0x00, B: 0xaa, A: 0xff}, // 5: Magenta
	color.RGBA{R: 0xaa, G: 0x55, B: 0x00, A: 0xff}, // 6: Brown
	color.RGBA{R: 0xaa, G: 0xaa, B: 0xaa, A: 0xff}, // 7: Light Gray
	color.RGBA{R: 0x55, G: 0x55, B: 0x55, A: 0xff}, // 8: Dark Gray
	color.RGBA{R: 0x55, G: 0x55, B: 0xff, A: 0xff}, // 9: Bright Blue
	color.RGBA{R: 0x55, G: 0xff, B: 0x55, A: 0xff}, // 10: Bright Green
	color.RGBA{R: 0x55, G: 0xff, B: 0xff, A: 0xff}, // 11: Bright Cyan
	color.RGBA{R: 0xff, G: 0x55, B: 0x55, A: 0xff}, // 12: Bright Red
	color.RGBA{R: 0xff, G: 0x55, B: 0xff, A: 0xff}, // 13: Bright Magenta
	color.RGBA{R: 0xff, G: 0xff, B: 0x55, A: 0xff}, // 14: Yellow
	color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}, // 15: Bright White
}

func main() {
	inPath := flag.String("in", "TILES.16", "Path to TILES.16")
	outPath := flag.String("out", "TILES.16.png", "Path to output PNG")
	flag.Parse()

	// Open file at *inPath
	inFile, err := os.Open(*inPath)
	if err != nil {
		log.Fatalf("failed to open input file %s: %v", *inPath, err)
	}

	// Un-LZW-compress TILES.16 data, then close the file.
	// TILES.16 contains a 4-byte header followed by LZW compressed data.
	if _, err := inFile.Seek(4, io.SeekStart); err != nil {
		inFile.Close()
		log.Fatalf("failed to seek past header in %s: %v", *inPath, err)
	}

	lzwReader := lzw.NewReader(inFile, lzw.LSB, 8)
	decompressed, err := io.ReadAll(lzwReader)
	lzwReader.Close()
	inFile.Close()
	if err != nil {
		log.Fatalf("failed to decompress LZW data from %s: %v", *inPath, err)
	}

	// Create output image.
	// HINT: The output image should be 256x512 pixels in size.
	const imgWidth = 256
	const imgHeight = 512
	const tileSize = 16
	const tilesPerRow = imgWidth / tileSize     // 16 tiles per row
	const bytesPerRow = 8                       // 16 nibbles = 8 bytes
	const bytesPerTile = tileSize * bytesPerRow // 128 bytes

	img := image.NewPaletted(image.Rect(0, 0, imgWidth, imgHeight), vgaPalette)

	// Fill the output image with black.
	for i := range img.Pix {
		img.Pix[i] = 0
	}

	// Loop through each tile of the decompressed TILES.16 data
	// HINT: TILES.16 data consists of 512 16x16 pixel tiles. Each row of 16
	// pixels is encoded as the nibbles of 8 8-bit bytes.

	// For each tile, draw it to the output image using the first 16 colors of
	// the standard VGA palette. Fill the output image with tiles from left to
	// right, then top to bottom. For example, tile index 18 would have a
	// top-left corner starting at pixel coordinates 32x16 because it is on the
	// second row and third column.
	totalTiles := len(decompressed) / bytesPerTile
	for tileIdx := 0; tileIdx < totalTiles; tileIdx++ {
		tileX := (tileIdx % tilesPerRow) * tileSize
		tileY := (tileIdx / tilesPerRow) * tileSize
		tileData := decompressed[tileIdx*bytesPerTile : (tileIdx+1)*bytesPerTile]

		for row := 0; row < tileSize; row++ {
			rowBytes := tileData[row*bytesPerRow : (row+1)*bytesPerRow]
			for colByte := 0; colByte < bytesPerRow; colByte++ {
				b := rowBytes[colByte]
				c1 := (b >> 4) & 0x0F
				c2 := b & 0x0F

				px1 := tileX + colByte*2
				px2 := tileX + colByte*2 + 1
				py := tileY + row

				img.SetColorIndex(px1, py, c1)
				img.SetColorIndex(px2, py, c2)
			}
		}
	}

	// Create or truncate file at *outPath and encode our output image to it as
	// a PNG, then close the file.
	outFile, err := os.Create(*outPath)
	if err != nil {
		log.Fatalf("failed to create output file %s: %v", *outPath, err)
	}

	if err := png.Encode(outFile, img); err != nil {
		outFile.Close()
		log.Fatalf("failed to encode PNG image to %s: %v", *outPath, err)
	}

	if err := outFile.Close(); err != nil {
		log.Fatalf("failed to close output file %s: %v", *outPath, err)
	}
}
