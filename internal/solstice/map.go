package solstice

import (
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"

	"solstice/data"

	"github.com/hajimehoshi/ebiten/v2"
)

// TileProperties holds physical, gameplay, and scripting properties for a tile.
type TileProperties struct {
	Walkable       bool   `json:"walkable"`
	BlocksVis      bool   `json:"blocks_vis"`
	DeepWater      bool   `json:"deep_water"`
	Water          bool   `json:"water"`
	Door           bool   `json:"door"`
	SpiritPassable bool   `json:"spirit_passable"`
	UseScript      string `json:"use_script"`
}

// TileSet holds tileset information loaded from a Tiled .tsx file.
type TileSet struct {
	Name       string
	TileWidth  int
	TileHeight int
	TileCount  int
	Columns    int
	Properties map[int]TileProperties // Tile ID -> TileProperties
}

// Map represents a 2D tile map loaded from a Tiled .tmx file.
type Map struct {
	Name       string
	Width      int
	Height     int
	TileWidth  int
	TileHeight int
	FirstGID   int
	Tiles      []int // 0-indexed tile indices
}

var defaultTileSet *TileSet
var defaultMap *Map

// GetMap returns the current default map instance.
func GetMap() *Map {
	return defaultMap
}

// SetMap sets the current default map instance.
func SetMap(m *Map) {
	defaultMap = m
}

// XML structures for deserializing TSX tilesets
type tsxTileset struct {
	XMLName    xml.Name   `xml:"tileset"`
	Name       string     `xml:"name,attr"`
	TileWidth  int        `xml:"tilewidth,attr"`
	TileHeight int        `xml:"tileheight,attr"`
	TileCount  int        `xml:"tilecount,attr"`
	Columns    int        `xml:"columns,attr"`
	Tiles      []tsxTile  `xml:"tile"`
}

type tsxTile struct {
	ID         int           `xml:"id,attr"`
	Properties []tsxProperty `xml:"properties>property"`
}

type tsxProperty struct {
	Name  string `xml:"name,attr"`
	Type  string `xml:"type,attr"`
	Value string `xml:"value,attr"`
}

// XML structures for deserializing TMX maps
type tmxMap struct {
	XMLName    xml.Name     `xml:"map"`
	Version    string       `xml:"version,attr"`
	Width      int          `xml:"width,attr"`
	Height     int          `xml:"height,attr"`
	TileWidth  int          `xml:"tilewidth,attr"`
	TileHeight int          `xml:"tileheight,attr"`
	Tilesets   []tmxTileset `xml:"tileset"`
	Layers     []tmxLayer   `xml:"layer"`
}

type tmxTileset struct {
	FirstGID int    `xml:"firstgid,attr"`
	Source   string `xml:"source,attr"`
}

type tmxLayer struct {
	ID     int     `xml:"id,attr"`
	Name   string  `xml:"name,attr"`
	Width  int     `xml:"width,attr"`
	Height int     `xml:"height,attr"`
	Data   tmxData `xml:"data"`
}

type tmxData struct {
	Encoding string `xml:"encoding,attr"`
	RawData  string `xml:",chardata"`
}

// PreloadTileSet pre-loads the default tile set from data/maps/tileset.tsx at program start.
func PreloadTileSet() (*TileSet, error) {
	return LoadTileSet("maps/tileset.tsx")
}

// LoadTileSet loads a .tsx tileset file from data.FS into a TileSet struct.
func LoadTileSet(path string) (*TileSet, error) {
	cleanPath := strings.TrimPrefix(path, "data/")
	if !strings.HasPrefix(cleanPath, "maps/") {
		cleanPath = "maps/" + cleanPath
	}

	b, err := data.FS.ReadFile(cleanPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read tileset file %s: %w", path, err)
	}

	var raw tsxTileset
	if err := xml.Unmarshal(b, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse tileset XML %s: %w", path, err)
	}

	ts := &TileSet{
		Name:       raw.Name,
		TileWidth:  raw.TileWidth,
		TileHeight: raw.TileHeight,
		TileCount:  raw.TileCount,
		Columns:    raw.Columns,
		Properties: make(map[int]TileProperties),
	}

	for _, t := range raw.Tiles {
		var tp TileProperties
		for _, p := range t.Properties {
			val, _ := strconv.ParseBool(p.Value)
			switch p.Name {
			case "walkable":
				tp.Walkable = val
			case "blocks_vis":
				tp.BlocksVis = val
			case "deep_water":
				tp.DeepWater = val
			case "water":
				tp.Water = val
			case "door":
				tp.Door = val
			case "spirit_passable":
				tp.SpiritPassable = val
			case "use_script":
				tp.UseScript = p.Value
			}
		}
		ts.Properties[t.ID] = tp
	}

	defaultTileSet = ts
	return ts, nil
}

// GetTileProperties returns the TileProperties for the given tile ID.
func (ts *TileSet) GetTileProperties(tileID int) TileProperties {
	if ts == nil || ts.Properties == nil {
		return TileProperties{}
	}
	return ts.Properties[tileID]
}

// GetTileProperties returns the TileProperties for the given tile ID from defaultTileSet.
func GetTileProperties(tileID int) TileProperties {
	if defaultTileSet != nil {
		return defaultTileSet.GetTileProperties(tileID)
	}
	return TileProperties{}
}

// LoadMap loads a TMX map by name from data.FS (e.g. "home" loads "data/maps/home.tmx").
func LoadMap(name string) (*Map, error) {
	filename := name
	if !strings.HasSuffix(filename, ".tmx") {
		filename += ".tmx"
	}
	filename = strings.TrimPrefix(filename, "data/")
	if !strings.HasPrefix(filename, "maps/") {
		filename = "maps/" + filename
	}

	b, err := data.FS.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read map file %s: %w", filename, err)
	}

	var raw tmxMap
	if err := xml.Unmarshal(b, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse map XML %s: %w", filename, err)
	}

	firstGID := 1
	if len(raw.Tilesets) > 0 && raw.Tilesets[0].FirstGID > 0 {
		firstGID = raw.Tilesets[0].FirstGID
	}

	var tiles []int
	if len(raw.Layers) > 0 {
		gids, err := parseCSV(raw.Layers[0].Data.RawData)
		if err != nil {
			return nil, fmt.Errorf("failed to parse map layer data: %w", err)
		}
		tiles = make([]int, len(gids))
		for i, gid := range gids {
			if gid > 0 {
				tiles[i] = gid - firstGID
			} else {
				tiles[i] = 0
			}
		}
	} else {
		tiles = make([]int, raw.Width*raw.Height)
	}

	m := &Map{
		Name:       name,
		Width:      raw.Width,
		Height:     raw.Height,
		TileWidth:  raw.TileWidth,
		TileHeight: raw.TileHeight,
		FirstGID:   firstGID,
		Tiles:      tiles,
	}

	defaultMap = m
	return m, nil
}

// GetTile returns the 0-indexed tile index at tile coordinates (x, y).
// Returns 0 if coordinates are out of bounds.
func (m *Map) GetTile(x, y int) int {
	if m == nil || x < 0 || x >= m.Width || y < 0 || y >= m.Height {
		return 0
	}
	return m.Tiles[y*m.Width+x]
}

// SetTile sets the 0-indexed tile index at tile coordinates (x, y).
// Is a no-op if coordinates are out of bounds.
func (m *Map) SetTile(x, y int, tileIdx int) {
	if m == nil || x < 0 || x >= m.Width || y < 0 || y >= m.Height {
		return
	}
	m.Tiles[y*m.Width+x] = tileIdx
}

// ExecuteTileUseScript executes the use_script for the tile at tile coordinates (x, y) if defined.
func (m *Map) ExecuteTileUseScript(x, y int) error {
	if m == nil || x < 0 || x >= m.Width || y < 0 || y >= m.Height {
		return nil
	}

	tileIdx := m.GetTile(x, y)
	props := GetTileProperties(tileIdx)

	if props.UseScript == "" {
		return nil
	}

	return ExecuteTileScript(props.UseScript, x, y, tileIdx)
}

// MoveParty handles relative party movement on the map.
// The party can move onto a tile if the tile is "walkable",
// or additionally if the party is in "spirit mode" and the tile has the "spirit_passable" property set to true.
// Returns true if movement succeeded, or false if blocked.
func (m *Map) MoveParty(p *Party, dx, dy int) bool {
	if m == nil || p == nil || (dx == 0 && dy == 0) {
		return false
	}

	targetX := p.X + dx
	targetY := p.Y + dy

	if targetX < 0 || targetX >= m.Width || targetY < 0 || targetY >= m.Height {
		return false
	}

	tileIdx := m.GetTile(targetX, targetY)
	props := GetTileProperties(tileIdx)

	passable := props.Walkable || (p.IsSpiritMode() && props.SpiritPassable)
	if !passable {
		return false
	}

	p.X = targetX
	p.Y = targetY
	return true
}

// DrawCentered renders the map centered on map coordinates (centerX, centerY) into the map view area using assets at scale 1 or 2.
// Out-of-bounds map tiles are drawn as a black void.
func (m *Map) DrawCentered(dst *ebiten.Image, assets *Assets, centerX, centerY int, scale int) {
	if assets == nil {
		return
	}

	cols := 11
	rows := 11
	centerStx := 5
	centerSty := 5

	if scale == 1 {
		cols = 23
		rows = 23
		centerStx = 11
		centerSty = 11
	}

	for sty := 0; sty < rows; sty++ {
		for stx := 0; stx < cols; stx++ {
			mx := centerX + (stx - centerStx)
			my := centerY + (sty - centerSty)

			if m != nil && mx >= 0 && mx < m.Width && my >= 0 && my < m.Height {
				tileIdx := m.GetTile(mx, my)
				assets.DrawMapTile(dst, tileIdx, stx, sty, scale)
			} else {
				assets.DrawBlackMapTile(dst, stx, sty, scale)
			}
		}
	}
}

// Draw renders the map centered on (5, 5) or (11, 11) into the map view display area.
func (m *Map) Draw(dst *ebiten.Image, assets *Assets, scale int) {
	centerX := 5
	centerY := 5
	if scale == 1 {
		centerX = 11
		centerY = 11
	}
	m.DrawCentered(dst, assets, centerX, centerY, scale)
}

func parseCSV(raw string) ([]int, error) {
	raw = strings.TrimSpace(raw)
	raw = strings.ReplaceAll(raw, "\n", "")
	raw = strings.ReplaceAll(raw, "\r", "")
	parts := strings.Split(raw, ",")
	res := make([]int, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		val, err := strconv.Atoi(p)
		if err != nil {
			return nil, fmt.Errorf("invalid tile GID %q: %w", p, err)
		}
		res = append(res, val)
	}
	return res, nil
}
