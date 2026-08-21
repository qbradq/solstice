package solstice

import (
	"encoding/xml"
	"fmt"
	"image"
	"image/color"
	"math"
	"strconv"
	"strings"

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
	Animated        bool   `json:"animated"`
	Frames          int    `json:"frames"`
	AWTBasic        bool   `json:"awt_basic"`
	AWTMaskTL       bool   `json:"awt_mask_tl"`
	AWTMaskTR       bool   `json:"awt_mask_tr"`
	AWTMaskBL       bool   `json:"awt_mask_bl"`
	AWTMaskBR       bool   `json:"awt_mask_br"`
	AWTMaskRiver    bool   `json:"awt_mask_river"`
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
	RemainingTurns int                    `json:"remaining_turns"`
	ScriptPath     string                 `json:"script_path"`
	Globals        map[string]interface{} `json:"globals"`
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
	ExitToWorld bool   `json:"exit_to_world"`
	LoadScript  string `json:"load_script,omitempty"`
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
	Items      []*Item       // Active items on this map
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
	XMLName    xml.Name  `xml:"tileset"`
	Name       string    `xml:"name,attr"`
	TileWidth  int       `xml:"tilewidth,attr"`
	TileHeight int       `xml:"tileheight,attr"`
	TileCount  int       `xml:"tilecount,attr"`
	Columns    int       `xml:"columns,attr"`
	Tiles      []tsxTile `xml:"tile"`
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
			case "animated":
				tp.Animated = val
			case "frames":
				f, _ := strconv.Atoi(p.Value)
				tp.Frames = f
			case "awt_basic":
				tp.AWTBasic = val
			case "awt_mask_tl":
				tp.AWTMaskTL = val
			case "awt_mask_tr":
				tp.AWTMaskTR = val
			case "awt_mask_bl":
				tp.AWTMaskBL = val
			case "awt_mask_br":
				tp.AWTMaskBR = val
			case "awt_mask_river":
				tp.AWTMaskRiver = val
			default:
				return nil, fmt.Errorf("unknown tile property %q on tile ID %d", p.Name, t.ID)
			}
		}
		ts.Properties[t.ID] = tp
	}

	defaultTileSet = ts
	if defaultAssets != nil {
		UpdateAnimatedWaterTiles(defaultAssets, defaultTileSet, globalAnimFrame)
	}
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

// ResolveAnimatedTile returns the animated frame tile index if tileID has Animated=true and Frames > 1.
func ResolveAnimatedTile(tileID int) int {
	props := GetTileProperties(tileID)
	if props.Animated && props.Frames > 1 {
		return tileID + (GetAnimFrame() % props.Frames)
	}
	return tileID
}

var (
	loadedMaps = make(map[string]*Map)
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
	loadedMaps = make(map[string]*Map)
}

// GetAllLoadedMaps returns a copy of all loaded map instances in memory.
func GetAllLoadedMaps() map[string]*Map {
	res := make(map[string]*Map, len(loadedMaps))
	for k, v := range loadedMaps {
		res[k] = v
	}
	return res
}

// SetLoadedMap caches or replaces a map instance in memory.
func SetLoadedMap(name string, m *Map) {
	cleanName := NormalizeMapName(name)
	loadedMaps[cleanName] = m
}

// LoadMap loads a TMX map by name from data.FS (e.g. "home" loads "data/maps/home.tmx"),
// or returns the in-memory cached instance if already loaded.
func LoadMap(name string) (*Map, error) {
	cleanName := NormalizeMapName(name)

	if cached, ok := loadedMaps[cleanName]; ok && cached != nil {
		return cached, nil
	}

	m, err := loadMapFromTMX(cleanName)
	if err != nil {
		return nil, err
	}

	loadedMaps[cleanName] = m

	return m, nil
}

// ReloadMap reloads a map from embedded data.FS if it is currently present in loadedMaps,
// wiping out the existing map in loadedMaps.
// If the named map is not in loadedMaps, it does nothing and returns (nil, nil).
// If the current map was reloaded, it updates the current map pointer.
// If the world map was reloaded, it updates the world map pointer.
func ReloadMap(name string) (*Map, error) {
	cleanName := NormalizeMapName(name)

	existing, ok := loadedMaps[cleanName]
	if !ok || existing == nil {
		return nil, nil
	}

	newMap, err := loadMapFromTMX(cleanName)
	if err != nil {
		return nil, err
	}

	loadedMaps[cleanName] = newMap

	if defaultMap != nil && (defaultMap == existing || NormalizeMapName(defaultMap.Name) == cleanName) {
		SetMap(newMap)
	}

	if defaultWorldMap != nil && (defaultWorldMap == existing || NormalizeMapName(defaultWorldMap.Name) == cleanName || cleanName == "world") {
		SetWorldMap(newMap)
	}

	return newMap, nil
}

// loadMapFromTMX parses a TMX map and instantiates initial actors, items, triggers, and tiles.
func loadMapFromTMX(name string) (*Map, error) {
	if len(actorDefs) == 0 {
		_, _ = PreloadActorDefs()
	}
	if len(itemDefs) == 0 {
		_, _ = PreloadItemDefs()
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
		switch p.Name {
		case "exit_to_world":
			val, _ := strconv.ParseBool(p.Value)
			mapProps.ExitToWorld = val
		case "load_script":
			mapProps.LoadScript = p.Value
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
		Items:      make([]*Item, 0),
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

			// Check object properties for map-level properties (e.g. exit_to_world, load_script)
			for _, p := range obj.Properties {
				switch p.Name {
				case "exit_to_world":
					val, _ := strconv.ParseBool(p.Value)
					mapProps.ExitToWorld = val
				case "load_script":
					mapProps.LoadScript = p.Value
				}
			}

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
				actorID := templateName
				if m.GetActorByID(actorID) != nil {
					idx := 1
					for {
						candidate := fmt.Sprintf("%s-%d", templateName, idx)
						if m.GetActorByID(candidate) == nil {
							actorID = candidate
							break
						}
						idx++
					}
				}
				actor, err := NewActorFromDef(actorID, templateName, tileX, tileY)
				if err != nil {
					actor = NewActor(actorID, tileX, tileY, templateName)
				}
				m.AddActor(actor)
			case "item":
				itemID := templateName
				if m.GetItemByID(itemID) != nil {
					idx := 1
					for {
						candidate := fmt.Sprintf("%s-%d", templateName, idx)
						if m.GetItemByID(candidate) == nil {
							itemID = candidate
							break
						}
						idx++
					}
				}
				item := NewItem(itemID, templateName, tileX, tileY)
				m.AddItem(item)
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

	m.Properties = mapProps

	// Execute load_script if defined on map load
	if m.Properties.LoadScript != "" {
		SetMap(m)
		_ = ExecuteMapScript(m.Properties.LoadScript)
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

// GenerateUniqueActorID returns an unused actor ID on the map based on desiredID.
func (m *Map) GenerateUniqueActorID(desiredID string) string {
	if m == nil || m.GetActorByID(desiredID) == nil {
		return desiredID
	}
	base := desiredID
	idx := 1
	for {
		candidate := fmt.Sprintf("%s-%d", base, idx)
		if m.GetActorByID(candidate) == nil {
			return candidate
		}
		idx++
	}
}

// GenerateUniqueItemID returns an unused item ID on the map based on desiredID.
func (m *Map) GenerateUniqueItemID(desiredID string) string {
	if m == nil || m.GetItemByID(desiredID) == nil {
		return desiredID
	}
	base := desiredID
	idx := 1
	for {
		candidate := fmt.Sprintf("%s-%d", base, idx)
		if m.GetItemByID(candidate) == nil {
			return candidate
		}
		idx++
	}
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

// AddItem adds an item to the map.
func (m *Map) AddItem(item *Item) {
	if m == nil || item == nil {
		return
	}
	m.Items = append(m.Items, item)
}

// GetItemByID searches the map's items for an item with the given ID.
// Returns nil if no matching item is found.
func (m *Map) GetItemByID(id string) *Item {
	if m == nil || len(m.Items) == 0 || id == "" {
		return nil
	}
	for _, it := range m.Items {
		if it != nil && it.ID == id {
			return it
		}
	}
	return nil
}

// RemoveItemByID removes an item with the given ID from the map.
// Returns true if an item was found and removed, false otherwise.
func (m *Map) RemoveItemByID(id string) bool {
	if m == nil || len(m.Items) == 0 || id == "" {
		return false
	}
	idx := -1
	for i, it := range m.Items {
		if it != nil && it.ID == id {
			idx = i
			break
		}
	}
	if idx >= 0 {
		m.Items = append(m.Items[:idx], m.Items[idx+1:]...)
		return true
	}
	return false
}

// FindItemsByTemplate returns all items on the map created from the given template.
func (m *Map) FindItemsByTemplate(template string) []*Item {
	if m == nil || len(m.Items) == 0 {
		return nil
	}
	var res []*Item
	for _, it := range m.Items {
		if it != nil && it.Template == template {
			res = append(res, it)
		}
	}
	return res
}

// RemoveEntityByID removes any actor or item matching the given ID from the map.
// Returns true if an entity was found and removed, false otherwise.
func (m *Map) RemoveEntityByID(id string) bool {
	if m == nil || id == "" {
		return false
	}
	removedActor := m.RemoveActorByID(id)
	removedItem := m.RemoveItemByID(id)
	return removedActor || removedItem
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
// Increments turn counter, runs idle AI scripts for actors when not in combat,
// decrements all active map timers, and executes any timers that expire.
func (m *Map) AdvanceTurn() {
	if m == nil {
		return
	}
	m.Turn++

	// Run idle scripts on actors if not in combat
	if !IsInCombat() && len(m.Actors) > 0 {
		actors := make([]*Actor, len(m.Actors))
		copy(actors, m.Actors)
		for _, actor := range actors {
			if actor != nil && actor.IdleScript != "" {
				_ = RunActorAIScript(actor, actor.IdleScript)
			}
		}
	}

	if len(m.Timers) == 0 {
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

// IsWalkable returns true if the tile at (x, y) is within map bounds and has the Walkable tile property.
func (m *Map) IsWalkable(x, y int) bool {
	if m == nil || x < 0 || x >= m.Width || y < 0 || y >= m.Height {
		return false
	}
	tileIdx := m.GetTile(x, y)
	return GetTileProperties(tileIdx).Walkable
}

// CanActorMoveTo returns true if an actor can step into tile (x, y).
// It verifies the tile is within bounds, is walkable, is not occupied by another actor,
// and is not occupied by the party (when not in combat mode).
func (m *Map) CanActorMoveTo(x, y int) bool {
	if m == nil || x < 0 || x >= m.Width || y < 0 || y >= m.Height {
		return false
	}
	if !m.IsWalkable(x, y) {
		return false
	}
	if m.HasActorAt(x, y) {
		return false
	}
	if !IsInCombat() {
		p := GetParty()
		if p != nil && p.X == x && p.Y == y {
			return false
		}
	}
	return true
}

// MoveActor moves an actor by (dx, dy) if the target tile is walkable and unoccupied.
// Returns true if movement succeeded, or false if blocked.
func (m *Map) MoveActor(actor *Actor, dx, dy int) bool {
	if m == nil || actor == nil || (dx == 0 && dy == 0) {
		return false
	}
	targetX := actor.X + dx
	targetY := actor.Y + dy
	if !m.CanActorMoveTo(targetX, targetY) {
		return false
	}
	actor.X = targetX
	actor.Y = targetY
	return true
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

// DrawCentered renders the map centered on the party's position (or the active party member in combat mode)
// into the map view area using assets at scale 1 or 2.
func (m *Map) DrawCentered(dst *ebiten.Image, assets *Assets, p *Party, scale int) {
	if p == nil {
		p = GetParty()
	}

	centerX := 16
	centerY := 16
	if p != nil {
		if IsInCombat() {
			if focus := GetCombatFocusActor(); focus != nil {
				centerX = focus.X
				centerY = focus.Y
			} else if len(p.Members) > 0 {
				curIdx := GetCombatMemberIndex()
				if curIdx >= len(p.Members) {
					curIdx = 0
				}
				centerX = p.Members[curIdx].X
				centerY = p.Members[curIdx].Y
			} else {
				centerX = p.X
				centerY = p.Y
			}
		} else {
			centerX = p.X
			centerY = p.Y
		}
	}

	m.DrawCenteredAt(dst, assets, p, scale, centerX, centerY, centerX, centerY, nil)
}

// DrawCenteredAt renders the map centered on specific tile coordinates (viewCenterX, viewCenterY) into the map view area,
// with visibility field calculations centered on (visCenterX, visCenterY).
// For areas outside the visibility field, they are treated as non-visible.
func (m *Map) DrawCenteredAt(dst *ebiten.Image, assets *Assets, p *Party, scale int, viewCenterX, viewCenterY int, visCenterX, visCenterY int, highlightTiles map[image.Point]bool, highlightColor ...color.Color) {
	if assets == nil {
		return
	}

	if p == nil {
		p = GetParty()
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

	// Always calculate visibility using 11 for radius (23x23 area) centered on (visCenterX, visCenterY)
	vis := m.CalculateVisibility(visCenterX, visCenterY, 11)

	// Helper to check visibility for map coordinates (mx, my)
	isTileVisible := func(mx, my int) bool {
		if vis == nil {
			return false
		}
		lx := 11 + (mx - visCenterX)
		ly := 11 + (my - visCenterY)
		if lx >= 0 && lx < 23 && ly >= 0 && ly < 23 {
			return vis.Test(uint(ly*23 + lx))
		}
		return false
	}

	// 1. Render map tiles
	for sty := 0; sty < rows; sty++ {
		for stx := 0; stx < cols; stx++ {
			mx := viewCenterX + (stx - centerStx)
			my := viewCenterY + (sty - centerSty)

			if isTileVisible(mx, my) {
				if m != nil && mx >= 0 && mx < m.Width && my >= 0 && my < m.Height {
					tileIdx := m.GetTile(mx, my)
					if tileIdx == 157 && p != nil && p.X == mx && p.Y == my+1 {
						tileIdx = 158
					}
					assets.DrawMapTile(dst, tileIdx, stx, sty, scale)
				} else {
					assets.DrawBlackMapTile(dst, stx, sty, scale)
				}
			} else {
				assets.DrawBlackMapTile(dst, stx, sty, scale)
			}
		}
	}

	// 2. Render items and actors present on the map (only in visible locations)
	if m != nil && len(m.Items) > 0 {
		for _, item := range m.Items {
			if item == nil || item.SpriteDef.Tile <= 0 {
				continue
			}
			stx := centerStx + (item.X - viewCenterX)
			sty := centerSty + (item.Y - viewCenterY)

			if stx >= 0 && stx < cols && sty >= 0 && sty < rows && isTileVisible(item.X, item.Y) {
				tileIdx := m.GetTile(item.X, item.Y)
				props := GetTileProperties(tileIdx)
				assets.DrawSpriteDefHalf(dst, item.SpriteDef, stx, sty, scale, props.ActorHalfSprite)
			}
		}
	}

	if m != nil && len(m.Actors) > 0 {
		for _, actor := range m.Actors {
			if actor == nil {
				continue
			}
			stx := centerStx + (actor.X - viewCenterX)
			sty := centerSty + (actor.Y - viewCenterY)

			if stx >= 0 && stx < cols && sty >= 0 && sty < rows && isTileVisible(actor.X, actor.Y) {
				tileIdx := m.GetTile(actor.X, actor.Y)
				props := GetTileProperties(tileIdx)
				assets.DrawSpriteDefHalf(dst, actor.SpriteDef, stx, sty, scale, props.ActorHalfSprite)
			}
		}
	}

	// 3. Render party (in combat mode: re-render the current party member on top; in party mode: render aggregate party sprite)
	if p != nil {
		if IsInCombat() && len(p.Members) > 0 {
			curIdx := GetCombatMemberIndex()
			if curIdx >= len(p.Members) {
				curIdx = 0
			}

			// Re-render current party member on top
			curMember := &p.Members[curIdx]
			stx := centerStx + (curMember.X - viewCenterX)
			sty := centerSty + (curMember.Y - viewCenterY)
			if stx >= 0 && stx < cols && sty >= 0 && sty < rows && isTileVisible(curMember.X, curMember.Y) {
				half := false
				if m != nil {
					tileIdx := m.GetTile(curMember.X, curMember.Y)
					half = GetTileProperties(tileIdx).ActorHalfSprite
				}
				assets.DrawSpriteDefHalf(dst, curMember.SpriteDef, stx, sty, scale, half)
			}
		} else {
			// Party mode: render aggregate party sprite at its relative position
			stx := centerStx + (p.X - viewCenterX)
			sty := centerSty + (p.Y - viewCenterY)
			if stx >= 0 && stx < cols && sty >= 0 && sty < rows && isTileVisible(p.X, p.Y) {
				half := false
				if m != nil {
					tileIdx := m.GetTile(p.X, p.Y)
					props := GetTileProperties(tileIdx)
					half = props.ActorHalfSprite
				}
				assets.DrawSpriteDefHalf(dst, p.GetSpriteDef(), stx, sty, scale, half)
			}
		}
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
					stx := centerStx + (tx - viewCenterX)
					sty := centerSty + (ty - viewCenterY)

					if stx >= 0 && stx < cols && sty >= 0 && sty < rows && isTileVisible(tx, ty) {
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

	// 5. Render highlighted tiles (e.g. reachable tiles in targeting mode)
	if len(highlightTiles) > 0 {
		mapArea := dst.SubImage(image.Rect(0, 0, 352, 352)).(*ebiten.Image)
		overlay := color.Color(color.RGBA{R: 0, G: 127, B: 0, A: 15}) // Default half-intensity, low alpha green
		if len(highlightColor) > 0 && highlightColor[0] != nil {
			overlay = highlightColor[0]
		}

		for pt := range highlightTiles {
			stx := centerStx + (pt.X - viewCenterX)
			sty := centerSty + (pt.Y - viewCenterY)

			if stx >= 0 && stx < cols && sty >= 0 && sty < rows && isTileVisible(pt.X, pt.Y) {
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
				vector.FillRect(mapArea, px, py, sz, sz, overlay, false)
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
