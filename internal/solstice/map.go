package solstice

import (
	"encoding/xml"
	"fmt"
	"image"
	"image/color"
	"math"
	"strconv"
	"strings"
	"sync"

	"solstice/data"

	"github.com/bits-and-blooms/bitset"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// TileProperties holds physical, gameplay, and scripting properties for a tile.
type TileProperties struct {
	Walkable        bool   `json:"walkable"`
	BlocksVis       bool   `json:"blocks_vis"`
	DeepWater       bool   `json:"deep_water"`
	Water           bool   `json:"water"`
	Climbable       bool   `json:"climbable"`
	SpiritPassable  bool   `json:"spirit_passable"`
	UseScript       string `json:"use_script"`
	PartySprite     string `json:"party_sprite"`
	ActorHalfSprite bool   `json:"actor_half_sprite"`
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

// Trigger represents an interactive or proximity trigger area on a map.
type Trigger struct {
	ID         int             `json:"id"`
	Name       string          `json:"name"`
	Area       image.Rectangle `json:"area"`        // Tile-coordinate bounding box: [Min.X, Min.Y, Max.X, Max.Y)
	ScriptPath string          `json:"script_path"` // Path to Tengo script to execute
	OnStep     bool            `json:"on_step"`     // Activates when actor or party steps on any tile in Area
	OnEnter    bool            `json:"on_enter"`    // Activates when party presses 'E' while standing on any tile in Area
}

// MapProperties holds gameplay and navigation properties for a map.
type MapProperties struct {
	ExitToWorld bool `json:"exit_to_world"`
}

// Map represents a 2D tile map loaded from a Tiled .tmx file.
type Map struct {
	Name       string
	Width      int
	Height     int
	TileWidth  int
	TileHeight int
	FirstGID   int
	Tiles      []int         // 0-indexed tile indices
	Properties MapProperties // Map-level properties
	Timers     []*MapTimer   // Scheduled map timers
	Actors     []*Actor      // Active actors on this map
	Triggers   []*Trigger    // Trigger areas on this map
	Turn       int           // Turn counter
}

var defaultTileSet *TileSet
var defaultMap *Map
var defaultWorldMap *Map
var wizardMode bool

// IsWizardMode returns true if wizard debugging mode is enabled.
func IsWizardMode() bool {
	return wizardMode
}

// SetWizardMode enables or disables wizard debugging mode.
func SetWizardMode(v bool) {
	wizardMode = v
}

// ToggleWizardMode toggles wizard debugging mode.
func ToggleWizardMode() {
	wizardMode = !wizardMode
}

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

// GetWorldMap returns the current default world map instance.
func GetWorldMap() *Map {
	return defaultWorldMap
}

// SetWorldMap sets the current default world map instance.
func SetWorldMap(m *Map) {
	defaultWorldMap = m
	if defaultGame != nil {
		defaultGame.worldMap = m
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
	Properties   []tmxProperty    `xml:"properties>property"`
	Tilesets     []tmxTileset     `xml:"tileset"`
	Layers       []tmxLayer       `xml:"layer"`
	ObjectGroups []tmxObjectGroup `xml:"objectgroup"`
}

type tmxProperty struct {
	Name  string `xml:"name,attr"`
	Type  string `xml:"type,attr"`
	Value string `xml:"value,attr"`
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
	ID         int           `xml:"id,attr"`
	Name       string        `xml:"name,attr"`
	Type       string        `xml:"type,attr"`
	GID        int           `xml:"gid,attr"`
	X          float64       `xml:"x,attr"`
	Y          float64       `xml:"y,attr"`
	Width      float64       `xml:"width,attr"`
	Height     float64       `xml:"height,attr"`
	Properties []tmxProperty `xml:"properties>property"`
}

// PreloadTileSet pre-loads the default tile set from data/maps/tileset.tsx at program start.
func PreloadTileSet() (*TileSet, error) {
	return LoadTileSet("maps/tileset.tsx")
}

// PreloadWorldMap pre-loads the default world map from data/maps/world.tmx at program start.
func PreloadWorldMap() (*Map, error) {
	m, err := LoadMap("world")
	if err != nil {
		return nil, err
	}
	SetWorldMap(m)
	return m, nil
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
			case "climbable":
				tp.Climbable = val
			case "spirit_passable":
				tp.SpiritPassable = val
			case "use_script":
				tp.UseScript = p.Value
			case "party_sprite":
				tp.PartySprite = p.Value
			case "actor_half_sprite", "party_half_sprite":
				tp.ActorHalfSprite = val
			default:
				return nil, fmt.Errorf("unknown tile property %q on tile ID %d", p.Name, t.ID)
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

var (
	loadedMapsMu sync.RWMutex
	loadedMaps   = make(map[string]*Map)
)

// NormalizeMapName strips path prefixes and extensions (e.g. "maps/home.tmx" -> "home").
func NormalizeMapName(name string) string {
	clean := name
	clean = strings.TrimPrefix(clean, "data/")
	clean = strings.TrimPrefix(clean, "maps/")
	clean = strings.TrimSuffix(clean, ".tmx")
	return clean
}

// ClearLoadedMaps resets the in-memory map cache.
func ClearLoadedMaps() {
	loadedMapsMu.Lock()
	defer loadedMapsMu.Unlock()
	loadedMaps = make(map[string]*Map)
}

// GetAllLoadedMaps returns a copy of all loaded map instances in memory.
func GetAllLoadedMaps() map[string]*Map {
	loadedMapsMu.RLock()
	defer loadedMapsMu.RUnlock()
	res := make(map[string]*Map, len(loadedMaps))
	for k, v := range loadedMaps {
		res[k] = v
	}
	return res
}

// SetLoadedMap caches or replaces a map instance in memory.
func SetLoadedMap(name string, m *Map) {
	loadedMapsMu.Lock()
	defer loadedMapsMu.Unlock()
	cleanName := NormalizeMapName(name)
	loadedMaps[cleanName] = m
}

// LoadMap loads a TMX map by name from data.FS (e.g. "home" loads "data/maps/home.tmx"),
// or returns the in-memory cached instance if already loaded.
func LoadMap(name string) (*Map, error) {
	cleanName := NormalizeMapName(name)

	loadedMapsMu.RLock()
	if cached, ok := loadedMaps[cleanName]; ok && cached != nil {
		loadedMapsMu.RUnlock()
		return cached, nil
	}
	loadedMapsMu.RUnlock()

	m, err := loadMapFromTMX(cleanName)
	if err != nil {
		return nil, err
	}

	loadedMapsMu.Lock()
	loadedMaps[cleanName] = m
	loadedMapsMu.Unlock()

	return m, nil
}

// loadMapFromTMX parses a TMX map and instantiates initial actors, triggers, and tiles.
func loadMapFromTMX(name string) (*Map, error) {
	if len(actorDefs) == 0 {
		_, _ = PreloadActorDefs()
	}
	if defaultSpriteDefs == nil {
		_, _ = PreloadSpriteDefs()
	}

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

	var mapProps MapProperties
	for _, p := range raw.Properties {
		val, _ := strconv.ParseBool(p.Value)
		switch p.Name {
		case "exit_to_world":
			mapProps.ExitToWorld = val
		default:
			return nil, fmt.Errorf("unknown map property %q for map %s", p.Name, name)
		}
	}

	m := &Map{
		Name:       name,
		Width:      raw.Width,
		Height:     raw.Height,
		TileWidth:  raw.TileWidth,
		TileHeight: raw.TileHeight,
		FirstGID:   firstGID,
		Tiles:      tiles,
		Properties: mapProps,
		Timers:     make([]*MapTimer, 0),
		Actors:     make([]*Actor, 0),
		Triggers:   make([]*Trigger, 0),
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

			objType := ""
			templateName := ""
			if strings.Contains(obj.Name, ":") {
				parts := strings.SplitN(obj.Name, ":", 2)
				objType = parts[0]
				templateName = parts[1]
			} else {
				objType = "actor"
				templateName = obj.Name
			}

			switch objType {
			case "actor":
				actorID := fmt.Sprintf("%s-%d", templateName, obj.ID)
				actor, err := NewActorFromDef(actorID, templateName, tileX, tileY)
				if err != nil {
					actor = NewActor(actorID, tileX, tileY, templateName)
				}
				m.AddActor(actor)
			case "trigger":
				var tileX, tileY, tileW, tileH int
				if obj.Width == 0 && obj.Height == 0 {
					// Point trigger object placed inside a tile cell
					tileX = int(math.Floor(obj.X / float64(raw.TileWidth)))
					tileY = int(math.Floor(obj.Y / float64(raw.TileHeight)))
					tileW = 1
					tileH = 1
				} else {
					tileX = int(math.Round(obj.X / float64(raw.TileWidth)))
					tileY = int(math.Round(obj.Y / float64(raw.TileHeight)))
					tileW = int(math.Round(obj.Width / float64(raw.TileWidth)))
					tileH = int(math.Round(obj.Height / float64(raw.TileHeight)))
					if tileW <= 0 {
						tileW = 1
					}
					if tileH <= 0 {
						tileH = 1
					}
				}

				scriptPath := templateName
				onStep := false
				onEnter := false

				for _, p := range obj.Properties {
					val, _ := strconv.ParseBool(p.Value)
					switch p.Name {
					case "on_step":
						onStep = val
					case "on_enter":
						onEnter = val
					case "script", "script_path":
						scriptPath = p.Value
					}
				}

				trig := &Trigger{
					ID:         obj.ID,
					Name:       obj.Name,
					Area:       image.Rect(tileX, tileY, tileX+tileW, tileY+tileH),
					ScriptPath: scriptPath,
					OnStep:     onStep,
					OnEnter:    onEnter,
				}
				m.AddTrigger(trig)
			}
		}
	}

	defaultMap = m
	cleanName := strings.TrimSuffix(name, ".tmx")
	cleanName = strings.TrimPrefix(cleanName, "data/")
	cleanName = strings.TrimPrefix(cleanName, "maps/")
	if cleanName == "world" {
		SetWorldMap(m)
	}
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

// GetActorByID searches the map's actors for an actor with the given ID.
// Returns nil if no matching actor is found.
func (m *Map) GetActorByID(id string) *Actor {
	if m == nil || len(m.Actors) == 0 || id == "" {
		return nil
	}
	for _, a := range m.Actors {
		if a != nil && a.ID == id {
			return a
		}
	}
	return nil
}

// RemoveActorByID removes an actor with the given ID from the map.
// Returns true if an actor was found and removed, false otherwise.
func (m *Map) RemoveActorByID(id string) bool {
	if m == nil || len(m.Actors) == 0 || id == "" {
		return false
	}
	idx := -1
	for i, actor := range m.Actors {
		if actor != nil && actor.ID == id {
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
		if m.Properties.ExitToWorld {
			worldMap := GetWorldMap()
			if worldMap != nil {
				SetMap(worldMap)
				p.X = p.WorldX
				p.Y = p.WorldY
				p.UpdateSpriteDef()
				worldMap.AdvanceTurn()
				return true
			}
		}
		return false
	}

	// Prevent party from moving onto a tile occupied by an Actor (unless in Spirit Mode)
	if !p.IsSpiritMode() && m.HasActorAt(targetX, targetY) {
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
	if m == defaultWorldMap || m.Name == "world" {
		p.WorldX = targetX
		p.WorldY = targetY
	}
	p.UpdateSpriteDef()

	// Check and activate on_step triggers for party
	m.ActivateTriggersOnStep(p.X, p.Y, "party")

	// Advance map turn after successful party movement
	m.AdvanceTurn()
	return true
}

// AddTrigger adds a trigger area to the map.
func (m *Map) AddTrigger(t *Trigger) {
	if m == nil || t == nil {
		return
	}
	m.Triggers = append(m.Triggers, t)
}

// ActivateTriggersOnStep checks for any triggers at (tileX, tileY) that have OnStep == true and executes them.
func (m *Map) ActivateTriggersOnStep(tileX, tileY int, actorID string) {
	if m == nil {
		return
	}
	pt := image.Pt(tileX, tileY)
	for _, trig := range m.Triggers {
		if trig != nil && trig.OnStep && pt.In(trig.Area) {
			_ = ExecuteTriggerScript(trig.ScriptPath, m.Name, tileX, tileY, actorID)
		}
	}
}

// ActivateTriggersOnEnter checks for any triggers at (tileX, tileY) that have OnEnter == true (and not OnStep) and executes them.
func (m *Map) ActivateTriggersOnEnter(tileX, tileY int, actorID string) {
	if m == nil {
		return
	}
	pt := image.Pt(tileX, tileY)
	for _, trig := range m.Triggers {
		if trig != nil && trig.OnEnter && !trig.OnStep && pt.In(trig.Area) {
			_ = ExecuteTriggerScript(trig.ScriptPath, m.Name, tileX, tileY, actorID)
		}
	}
}

// CalculateVisibility computes a 2-generation visibility bitmap for an area of the given radius centered on (centerX, centerY).
// Returns a BitSet from github.com/bits-and-blooms/bitset representing an area of size S x S where S = 2*radius + 1.
// Indexing into the bitset is given by row-major order: ly * S + lx, where (lx, ly) are local coordinates in [0, S-1].
func (m *Map) CalculateVisibility(centerX, centerY, radius int) *bitset.BitSet {
	if radius < 0 {
		return bitset.New(0)
	}

	side := 2*radius + 1
	totalBits := uint(side * side)

	vis1 := bitset.New(totalBits)

	type gridPoint struct {
		lx int
		ly int
	}

	// 1. Visit tile at center point (radius, radius) and all 4 cardinal adjacent locations
	cardinalDirs := [4][2]int{{0, -1}, {0, 1}, {-1, 0}, {1, 0}}

	vis1.Set(uint(radius*side + radius))
	queue := []gridPoint{{lx: radius, ly: radius}}

	for _, d := range cardinalDirs {
		adjLX := radius + d[0]
		adjLY := radius + d[1]
		if adjLX >= 0 && adjLX < side && adjLY >= 0 && adjLY < side {
			nidx := uint(adjLY*side + adjLX)
			if !vis1.Test(nidx) {
				vis1.Set(nidx)
				queue = append(queue, gridPoint{lx: adjLX, ly: adjLY})
			}
		}
	}

	// 2. Flood-fill along 4 cardinal directions, stopping propagation on tiles with blocks_vis == true
	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]

		mx := centerX + (curr.lx - radius)
		my := centerY + (curr.ly - radius)

		blocksVis := false
		if m == nil || mx < 0 || mx >= m.Width || my < 0 || my >= m.Height {
			blocksVis = true
		} else {
			tileIdx := m.GetTile(mx, my)
			props := GetTileProperties(tileIdx)
			blocksVis = props.BlocksVis
		}

		if blocksVis {
			continue
		}

		for _, d := range cardinalDirs {
			nlx := curr.lx + d[0]
			nly := curr.ly + d[1]
			if nlx >= 0 && nlx < side && nly >= 0 && nly < side {
				nidx := uint(nly*side + nlx)
				if !vis1.Test(nidx) {
					vis1.Set(nidx)
					queue = append(queue, gridPoint{lx: nlx, ly: nly})
				}
			}
		}
	}

	// 3. Create second generation visibility bitmap
	vis2 := bitset.New(totalBits)

	for ly := 0; ly < side; ly++ {
		for lx := 0; lx < side; lx++ {
			idx := uint(ly*side + lx)
			if vis1.Test(idx) {
				vis2.Set(idx)
				continue
			}

			mx := centerX + (lx - radius)
			my := centerY + (ly - radius)
			if m != nil && mx >= 0 && mx < m.Width && my >= 0 && my < m.Height {
				tileIdx := m.GetTile(mx, my)
				props := GetTileProperties(tileIdx)
				if props.BlocksVis {
					// Check if at least 2 of 4 cardinal adjacent locations are visible in vis1
					adjVisibleCount := 0
					for _, d := range cardinalDirs {
						adjX := lx + d[0]
						adjY := ly + d[1]
						if adjX >= 0 && adjX < side && adjY >= 0 && adjY < side {
							adjIdx := uint(adjY*side + adjX)
							if vis1.Test(adjIdx) {
								adjVisibleCount++
							}
						}
					}
					if adjVisibleCount >= 2 {
						vis2.Set(idx)
					}
				}
			}
		}
	}

	return vis2
}

// DrawCentered renders the map centered on the party's position into the map view area using assets at scale 1 or 2.
// Visibility is calculated with radius 11. Non-visible tiles are drawn as black void, actors in non-visible tiles are hidden,
// and the party sprite is always rendered at the center cell.
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

	// Always calculate visibility using 11 for radius (23x23 area)
	vis := m.CalculateVisibility(centerX, centerY, 11)

	// Helper to check visibility for a screen tile (stx, sty)
	isTileVisible := func(stx, sty int) bool {
		if vis == nil {
			return false
		}
		if scale == 1 {
			if stx >= 0 && stx < 23 && sty >= 0 && sty < 23 {
				return vis.Test(uint(sty*23 + stx))
			}
			return false
		}
		// Scale 2: inner 11x11 portion corresponding to offsets -5..+5 -> lx = 6+stx, ly = 6+sty
		if stx >= 0 && stx < 11 && sty >= 0 && sty < 11 {
			lx := 6 + stx
			ly := 6 + sty
			return vis.Test(uint(ly*23 + lx))
		}
		return false
	}

	// 1. Render map tiles
	for sty := 0; sty < rows; sty++ {
		for stx := 0; stx < cols; stx++ {
			if isTileVisible(stx, sty) {
				mx := centerX + (stx - centerStx)
				my := centerY + (sty - centerSty)

				if m != nil && mx >= 0 && mx < m.Width && my >= 0 && my < m.Height {
					tileIdx := m.GetTile(mx, my)
					assets.DrawMapTile(dst, tileIdx, stx, sty, scale)
				} else {
					assets.DrawBlackMapTile(dst, stx, sty, scale)
				}
			} else {
				assets.DrawBlackMapTile(dst, stx, sty, scale)
			}
		}
	}

	// 2. Render actors present on the map (only in visible locations)
	if m != nil && len(m.Actors) > 0 {
		for _, actor := range m.Actors {
			if actor == nil {
				continue
			}
			stx := centerStx + (actor.X - centerX)
			sty := centerSty + (actor.Y - centerY)

			if stx >= 0 && stx < cols && sty >= 0 && sty < rows && isTileVisible(stx, sty) {
				tileIdx := m.GetTile(actor.X, actor.Y)
				props := GetTileProperties(tileIdx)
				assets.DrawSpriteDefHalf(dst, actor.SpriteDef, stx, sty, scale, props.ActorHalfSprite)
			}
		}
	}

	// 3. Render party sprite at the center cell (always drawn regardless of visibility bitmap)
	if p != nil {
		half := false
		if m != nil {
			tileIdx := m.GetTile(p.X, p.Y)
			props := GetTileProperties(tileIdx)
			half = props.ActorHalfSprite
		}
		assets.DrawSpriteDefHalf(dst, p.GetSpriteDef(), centerStx, centerSty, scale, half)
	}

	// 4. In Wizard Mode, render 35% transparent blue rectangle over every covered tile in triggers
	if IsWizardMode() && m != nil && len(m.Triggers) > 0 {
		mapArea := dst.SubImage(image.Rect(0, 0, 352, 352)).(*ebiten.Image)
		blueOverlay := color.RGBA{R: 0, G: 0, B: 255, A: 89} // 35% transparent blue

		for _, trig := range m.Triggers {
			if trig == nil {
				continue
			}
			for ty := trig.Area.Min.Y; ty < trig.Area.Max.Y; ty++ {
				for tx := trig.Area.Min.X; tx < trig.Area.Max.X; tx++ {
					stx := centerStx + (tx - centerX)
					sty := centerSty + (ty - centerY)

					if stx >= 0 && stx < cols && sty >= 0 && sty < rows && isTileVisible(stx, sty) {
						var px, py, sz float32
						if scale == 2 {
							px = float32(stx * 32)
							py = float32(sty * 32)
							sz = 32
						} else {
							px = float32(-8 + stx*16)
							py = float32(-8 + sty*16)
							sz = 16
						}
						vector.FillRect(mapArea, px, py, sz, sz, blueOverlay, false)
					}
				}
			}
		}
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
