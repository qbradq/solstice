package solstice

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var (
	saveDirMu       sync.RWMutex
	saveDirOverride string
)

// SetSaveDirOverride allows overriding the save directory for unit tests.
func SetSaveDirOverride(dir string) {
	saveDirMu.Lock()
	defer saveDirMu.Unlock()
	saveDirOverride = dir
}

// GetSaveDir returns the path to the solstice save directory in os.UserConfigDir().
func GetSaveDir() (string, error) {
	saveDirMu.RLock()
	override := saveDirOverride
	saveDirMu.RUnlock()

	if override != "" {
		if err := os.MkdirAll(override, 0o755); err != nil {
			return "", err
		}
		return override, nil
	}

	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user config directory: %w", err)
	}

	saveDir := filepath.Join(configDir, "solstice")
	if err := os.MkdirAll(saveDir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create save directory %s: %w", saveDir, err)
	}
	return saveDir, nil
}

// GetSlotFilePath returns the file path for the given save slot (1, 2, or 3).
func GetSlotFilePath(slot int) (string, error) {
	if slot < 1 || slot > 3 {
		return "", fmt.Errorf("invalid save slot %d (must be 1, 2, or 3)", slot)
	}
	saveDir, err := GetSaveDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(saveDir, fmt.Sprintf("slot_%d.json", slot)), nil
}

// SaveSlotInfo describes the metadata and availability of a save slot.
type SaveSlotInfo struct {
	Slot        int    `json:"slot"`
	Exists      bool   `json:"exists"`
	DisplayTime string `json:"display_time"`
	FilePath    string `json:"file_path"`
}

// SavedMapState serializes the in-memory state of a Map instance.
type SavedMapState struct {
	Name       string        `json:"name"`
	Width      int           `json:"width"`
	Height     int           `json:"height"`
	TileWidth  int           `json:"tile_width"`
	TileHeight int           `json:"tile_height"`
	FirstGID   int           `json:"first_gid"`
	Tiles      string        `json:"tiles"` // Base64-encoded zlib-compressed int32 slice
	Actors     []*Actor      `json:"actors"`
	Timers     []*MapTimer   `json:"timers"`
	Properties MapProperties `json:"properties"`
	Turn       int           `json:"turn"`
}

// SaveGameData is the root serialization object stored in slot JSON files.
type SaveGameData struct {
	Version        int                       `json:"version"`
	Slot           int                       `json:"slot"`
	Timestamp      string                    `json:"timestamp"`
	DisplayTime    string                    `json:"display_time"`
	Party          *Party                    `json:"party"`
	CurrentMapName string                    `json:"current_map_name"`
	Flags          map[string]bool           `json:"flags"`
	Maps           map[string]*SavedMapState `json:"maps"`
}

// EncodeTilesBase64 compresses and base64-encodes a slice of integer tile IDs.
func EncodeTilesBase64(tiles []int) (string, error) {
	buf := new(bytes.Buffer)
	for _, t := range tiles {
		if err := binary.Write(buf, binary.LittleEndian, int32(t)); err != nil {
			return "", err
		}
	}

	var compBuf bytes.Buffer
	zw := zlib.NewWriter(&compBuf)
	if _, err := zw.Write(buf.Bytes()); err != nil {
		return "", err
	}
	if err := zw.Close(); err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(compBuf.Bytes()), nil
}

// DecodeTilesBase64 base64-decodes and decompresses a tile string into a slice of integers.
func DecodeTilesBase64(encoded string, count int) ([]int, error) {
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64 tiles: %w", err)
	}

	zr, err := zlib.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to open zlib tile reader: %w", err)
	}
	decomp, err := io.ReadAll(zr)
	if err != nil {
		return nil, fmt.Errorf("failed to decompress tiles: %w", err)
	}
	_ = zr.Close()

	reader := bytes.NewReader(decomp)
	tiles := make([]int, count)
	for i := 0; i < count; i++ {
		var val int32
		if err := binary.Read(reader, binary.LittleEndian, &val); err != nil {
			return nil, fmt.Errorf("failed reading tile %d: %w", i, err)
		}
		tiles[i] = int(val)
	}

	return tiles, nil
}

// ToSavedState converts an in-memory Map to a SavedMapState.
func (m *Map) ToSavedState() (*SavedMapState, error) {
	if m == nil {
		return nil, nil
	}
	encodedTiles, err := EncodeTilesBase64(m.Tiles)
	if err != nil {
		return nil, fmt.Errorf("failed to encode tiles for map %s: %w", m.Name, err)
	}

	// Copy actors
	actorsCopy := make([]*Actor, len(m.Actors))
	for i, a := range m.Actors {
		if a != nil {
			ac := *a
			actorsCopy[i] = &ac
		}
	}

	// Copy timers
	timersCopy := make([]*MapTimer, len(m.Timers))
	for i, t := range m.Timers {
		if t != nil {
			tc := *t
			timersCopy[i] = &tc
		}
	}

	return &SavedMapState{
		Name:       m.Name,
		Width:      m.Width,
		Height:     m.Height,
		TileWidth:  m.TileWidth,
		TileHeight: m.TileHeight,
		FirstGID:   m.FirstGID,
		Tiles:      encodedTiles,
		Actors:     actorsCopy,
		Timers:     timersCopy,
		Properties: m.Properties,
		Turn:       m.Turn,
	}, nil
}

// RestoreMapFromSavedState restores a Map instance from a SavedMapState, merging with baseline TMX triggers.
func RestoreMapFromSavedState(saved *SavedMapState) (*Map, error) {
	if saved == nil {
		return nil, fmt.Errorf("nil saved map state")
	}

	// Load baseline map from TMX to obtain triggers and structural geometry
	baseline, err := loadMapFromTMX(saved.Name)
	if err != nil {
		// If TMX loading fails, instantiate standalone map
		baseline = &Map{
			Name:       saved.Name,
			Width:      saved.Width,
			Height:     saved.Height,
			TileWidth:  saved.TileWidth,
			TileHeight: saved.TileHeight,
			FirstGID:   saved.FirstGID,
			Triggers:   make([]*Trigger, 0),
		}
	}

	// Decompress and restore tile matrix
	tiles, err := DecodeTilesBase64(saved.Tiles, saved.Width*saved.Height)
	if err != nil {
		return nil, fmt.Errorf("failed to decode saved tiles for map %s: %w", saved.Name, err)
	}

	baseline.Tiles = tiles
	baseline.Properties = saved.Properties
	baseline.Turn = saved.Turn
	baseline.Actors = saved.Actors
	baseline.Timers = saved.Timers

	return baseline, nil
}

// GetAllSaveSlotInfos inspects all 3 slots and returns their info.
func GetAllSaveSlotInfos() ([]SaveSlotInfo, error) {
	infos := make([]SaveSlotInfo, 3)
	for i := 1; i <= 3; i++ {
		path, err := GetSlotFilePath(i)
		if err != nil {
			return nil, err
		}

		info := SaveSlotInfo{
			Slot:        i,
			FilePath:    path,
			DisplayTime: "[Empty]",
			Exists:      false,
		}

		if data, err := os.ReadFile(path); err == nil {
			var save SaveGameData
			if err := json.Unmarshal(data, &save); err == nil && save.DisplayTime != "" {
				info.Exists = true
				info.DisplayTime = save.DisplayTime
			} else if stat, err := os.Stat(path); err == nil {
				info.Exists = true
				info.DisplayTime = stat.ModTime().Format("2006/01/02 3:04 PM")
			}
		}

		infos[i-1] = info
	}
	return infos, nil
}

// HasAnySaveFiles returns true if at least one save slot contains a valid save file.
func HasAnySaveFiles() bool {
	infos, err := GetAllSaveSlotInfos()
	if err != nil {
		return false
	}
	for _, info := range infos {
		if info.Exists {
			return true
		}
	}
	return false
}

// SaveGame serializes the current game state to the specified slot.
// If pretty is true, output JSON is indented.
func SaveGame(slot int, pretty bool) error {
	path, err := GetSlotFilePath(slot)
	if err != nil {
		return err
	}

	party := GetParty()
	if party == nil {
		return fmt.Errorf("cannot save game: party is nil")
	}
	currentMap := GetMap()
	if currentMap == nil {
		return fmt.Errorf("cannot save game: currentMap is nil")
	}

	now := time.Now()
	saveData := SaveGameData{
		Version:        1,
		Slot:           slot,
		Timestamp:      now.Format(time.RFC3339),
		DisplayTime:    now.Format("2006/01/02 3:04 PM"),
		Party:          party,
		CurrentMapName: currentMap.Name,
		Flags:          GetAllFlags(),
		Maps:           make(map[string]*SavedMapState),
	}

	// Save all loaded maps in memory
	allMaps := GetAllLoadedMaps()
	for name, m := range allMaps {
		savedMap, err := m.ToSavedState()
		if err != nil {
			return fmt.Errorf("failed to serialize map %s: %w", name, err)
		}
		saveData.Maps[name] = savedMap
	}

	var data []byte
	if pretty {
		data, err = json.MarshalIndent(saveData, "", "  ")
	} else {
		data, err = json.Marshal(saveData)
	}
	if err != nil {
		return fmt.Errorf("failed to marshal save data: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("failed to write save file %s: %w", path, err)
	}

	return nil
}

// LoadGame deserializes and restores the game state from the specified slot.
func LoadGame(slot int) error {
	path, err := GetSlotFilePath(slot)
	if err != nil {
		return err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read save file %s: %w", path, err)
	}

	var saveData SaveGameData
	if err := json.Unmarshal(data, &saveData); err != nil {
		return fmt.Errorf("failed to unmarshal save file %s: %w", path, err)
	}

	// 1. Reset in-memory map cache
	ClearLoadedMaps()

	// 2. Restore all saved maps
	for name, savedMap := range saveData.Maps {
		m, err := RestoreMapFromSavedState(savedMap)
		if err != nil {
			return fmt.Errorf("failed to restore map %s: %w", name, err)
		}
		SetLoadedMap(name, m)
		if name == "world" || NormalizeMapName(name) == "world" {
			defaultWorldMap = m
		}
	}

	// If world map was not loaded yet, preload baseline
	if defaultWorldMap == nil {
		if _, err := PreloadWorldMap(); err != nil {
			return fmt.Errorf("failed to preload world map: %w", err)
		}
	}

	// 3. Restore party
	SetParty(saveData.Party)
	if saveData.Party != nil {
		saveData.Party.UpdateSpriteDef()
	}

	// 4. Restore flags
	RestoreFlags(saveData.Flags)

	// 5. Restore current active map
	currentMap, err := LoadMap(saveData.CurrentMapName)
	if err != nil {
		return fmt.Errorf("failed to load current map %s: %w", saveData.CurrentMapName, err)
	}
	SetMap(currentMap)

	// 6. Reset cutscene state
	ClearCutScene()

	return nil
}
