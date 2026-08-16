package solstice

import (
	"encoding/xml"
	"fmt"
	"image"
	"math"
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
	PartySprite    string `json:"party_sprite"`
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

// MapTimer represents a scheduled timer on a map.
type MapTimer struct {
	RemainingTurns int
	ScriptPath     string
	Globals        map[string]interface{}
}

// Map represents a 2D tile map loaded from a Tiled .tmx file.
type Map struct {
	Name       string
	Width      int
	Height     int
	TileWidth  int
	TileHeight int
	FirstGID   int
	Tiles      []int       // 0-indexed tile indices
	Timers     []*MapTimer // Scheduled map timers
	Actors     []*Actor    // Active actors on this map
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
	if defaultGame != nil {
		defaultGame.currentMap = m
	}
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
	XMLName      xml.Name         `xml:"map"`
	Version      string           `xml:"version,attr"`
	Width        int              `xml:"width,attr"`
	Height       int              `xml:"height,attr"`
	TileWidth    int              `xml:"tilewidth,attr"`
	TileHeight   int              `xml:"tileheight,attr"`
	Tilesets     []tmxTileset     `xml:"tileset"`
	Layers       []tmxLayer       `xml:"layer"`
	ObjectGroups []tmxObjectGroup `xml:"objectgroup"`
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

type tmxObjectGroup struct {
	ID      int         `xml:"id,attr"`
	Name    string      `xml:"name,attr"`
	Objects []tmxObject `xml:"object"`
}

type tmxObject struct {
	ID     int     `xml:"id,attr"`
	Name   string  `xml:"name,attr"`
	Type   string  `xml:"type,attr"`
	GID    int     `xml:"gid,attr"`
	X      float64 `xml:"x,attr"`
	Y      float64 `xml:"y,attr"`
	Width  float64 `xml:"width,attr"`
	Height float64 `xml:"height,attr"`
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
			case "party_sprite":
				tp.PartySprite = p.Value
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
		Timers:     make([]*MapTimer, 0),
		Actors:     make([]*Actor, 0),
	}

	for _, og := range raw.ObjectGroups {
		for _, obj := range og.Objects {
			if obj.Name == "" {
				continue
			}
			tileX := int(math.Round(obj.X / float64(raw.TileWidth)))

			objY := obj.Y
			if obj.GID > 0 {
				// In Tiled TMX format, tile objects (with GID) specify the bottom-left coordinate of the tile.
				// Adjust Y by subtracting object height (or TileHeight) to obtain top-left Y coordinate.
				h := obj.Height
				if h == 0 {
					h = float64(raw.TileHeight)
				}
				objY -= h
			}
			tileY := int(math.Round(objY / float64(raw.TileHeight)))

			actorID := fmt.Sprintf("%s-%d", obj.Name, obj.ID)
			actor, err := NewActorFromDef(actorID, obj.Name, tileX, tileY)
			if err != nil {
				actor = NewActor(actorID, tileX, tileY, obj.Name)
			}
			m.AddActor(actor)
		}
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

// AddActor adds an actor to the map.
func (m *Map) AddActor(a *Actor) {
	if m == nil || a == nil {
		return
	}
	m.Actors = append(m.Actors, a)
}

// RemoveActor removes an actor from the map.
func (m *Map) RemoveActor(a *Actor) bool {
	if m == nil || a == nil || len(m.Actors) == 0 {
		return false
	}
	idx := -1
	for i, actor := range m.Actors {
		if actor == a || (actor != nil && actor.ID != "" && actor.ID == a.ID) {
			idx = i
			break
		}
	}
	if idx >= 0 {
		m.Actors = append(m.Actors[:idx], m.Actors[idx+1:]...)
		return true
	}
	return false
}

// GetActorsInArea returns all actors on the map whose tile coordinates lie inside area.
func (m *Map) GetActorsInArea(area image.Rectangle) []*Actor {
	if m == nil || len(m.Actors) == 0 {
		return nil
	}
	var res []*Actor
	for _, a := range m.Actors {
		if a != nil {
			pt := image.Pt(a.X, a.Y)
			if pt.In(area) {
				res = append(res, a)
			}
		}
	}
	return res
}

// AddTimer schedules a new timer on the map with a delay expressed in turns,
// the name of the script to execute upon expiry, and global variables to inject into the script context.
func (m *Map) AddTimer(delayTurns int, scriptPath string, globals map[string]interface{}) {
	if m == nil {
		return
	}
	timer := &MapTimer{
		RemainingTurns: delayTurns,
		ScriptPath:     scriptPath,
		Globals:        globals,
	}
	m.Timers = append(m.Timers, timer)
}

// AdvanceTurn simulates one game turn within the current map.
// Decrements all active map timers and executes any timers that expire.
func (m *Map) AdvanceTurn() {
	if m == nil || len(m.Timers) == 0 {
		return
	}

	activeTimers := make([]*MapTimer, 0, len(m.Timers))
	expiredTimers := make([]*MapTimer, 0)

	for _, timer := range m.Timers {
		timer.RemainingTurns--
		if timer.RemainingTurns <= 0 {
			expiredTimers = append(expiredTimers, timer)
		} else {
			activeTimers = append(activeTimers, timer)
		}
	}

	m.Timers = activeTimers

	for _, timer := range expiredTimers {
		_ = ExecuteScriptWithGlobals(timer.ScriptPath, timer.Globals)
	}
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

// GetActorAt returns the actor occupying tile coordinates (x, y) if present, or nil if unoccupied.
func (m *Map) GetActorAt(x, y int) *Actor {
	if m == nil || len(m.Actors) == 0 {
		return nil
	}
	for _, a := range m.Actors {
		if a != nil && a.X == x && a.Y == y {
			return a
		}
	}
	return nil
}

// HasActorAt returns true if an actor occupies tile coordinates (x, y).
func (m *Map) HasActorAt(x, y int) bool {
	return m.GetActorAt(x, y) != nil
}

// MoveParty handles relative party movement on the map.
// The party can move onto a tile if the tile is "walkable" (or "spirit_passable" in spirit mode)
// AND the tile is not occupied by an Actor.
// Simulates one game turn within the current map upon a successful move.
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

	// Prevent party from moving onto a tile occupied by an Actor
	if m.HasActorAt(targetX, targetY) {
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

	// Advance map turn after successful party movement
	m.AdvanceTurn()
	return true
}

// DrawCentered renders the map centered on the party's position into the map view area using assets at scale 1 or 2.
// Out-of-bounds map tiles are drawn as a black void, actors are rendered on top of map tiles, and the party sprite is rendered at the center cell.
func (m *Map) DrawCentered(dst *ebiten.Image, assets *Assets, p *Party, scale int) {
	if assets == nil {
		return
	}

	if p == nil {
		p = GetParty()
	}

	centerX := 16
	centerY := 16
	if p != nil {
		centerX = p.X
		centerY = p.Y
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

	// 1. Render map tiles
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

	// 2. Render actors present on the map
	if m != nil && len(m.Actors) > 0 {
		for _, actor := range m.Actors {
			if actor == nil {
				continue
			}
			stx := centerStx + (actor.X - centerX)
			sty := centerSty + (actor.Y - centerY)

			if stx >= 0 && stx < cols && sty >= 0 && sty < rows {
				assets.DrawSpriteDef(dst, actor.SpriteDef, stx, sty, scale)
			}
		}
	}

	// 3. Render party sprite at the center cell
	if p != nil {
		assets.DrawSpriteDef(dst, p.GetSpriteDef(), centerStx, centerSty, scale)
	}
}

// Draw renders the map centered on the current party into the map view display area.
func (m *Map) Draw(dst *ebiten.Image, assets *Assets, scale int) {
	m.DrawCentered(dst, assets, nil, scale)
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
