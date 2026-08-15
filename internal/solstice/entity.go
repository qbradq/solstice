package solstice

import (
	"encoding/json"
	"fmt"
	"strings"

	"solstice/data"
)

// SpriteDef holds the information for the graphical representation of an
// entity.
type SpriteDef struct {
	Tile     int  `json:"tile"`     // Starting tile of the animation
	Animated bool `json:"animated"` // If true, cycle through frames from Tile to Tile+Frames-1
	Frames   int  `json:"frames"`   // Number of frames if Animated is true
}

// Entity represents the common data and base functionality for objects and
// actors within the world.
type Entity struct {
	SpriteDef `json:"sprite"` // Sprite definition
	Name      string          `json:"name"` // Descriptive name
	X         int             `json:"x"`    // X position
	Y         int             `json:"y"`    // Y position
}

var defaultSpriteDefs map[string]SpriteDef

// PreloadSpriteDefs pre-loads data/json/spritedefs.json from data.FS at program start.
func PreloadSpriteDefs() (map[string]SpriteDef, error) {
	return LoadSpriteDefs("json/spritedefs.json")
}

// LoadSpriteDefs loads a sprite definitions JSON file from data.FS into a map of string -> SpriteDef.
func LoadSpriteDefs(path string) (map[string]SpriteDef, error) {
	cleanPath := strings.TrimPrefix(path, "data/")
	if !strings.HasPrefix(cleanPath, "json/") {
		cleanPath = "json/" + cleanPath
	}

	b, err := data.FS.ReadFile(cleanPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read sprite defs file %s: %w", path, err)
	}

	var defs map[string]SpriteDef
	if err := json.Unmarshal(b, &defs); err != nil {
		return nil, fmt.Errorf("failed to parse sprite defs JSON %s: %w", path, err)
	}

	defaultSpriteDefs = defs
	return defs, nil
}

// GetSpriteDef returns a copy of the SpriteDef for the given name from pre-loaded sprite definitions.
func GetSpriteDef(name string) (SpriteDef, bool) {
	if defaultSpriteDefs == nil {
		return SpriteDef{}, false
	}
	sd, ok := defaultSpriteDefs[name]
	return sd, ok
}
