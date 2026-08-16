package solstice

import (
	"encoding/json"
	"fmt"
	"strings"

	"solstice/data"
)

// ActorDef represents an actor definition loaded from JSON.
type ActorDef struct {
	Name         string `json:"name"`
	Sprite       string `json:"sprite"`        // Key into spritedefs.json
	DialogScript string `json:"dialog_script"` // Path to dialog script
}

// Actor represents an agent or character in the game world (NPC, monster, party member).
// It embeds Entity for position and graphical representation.
type Actor struct {
	Entity
	ID           string `json:"id"`
	DialogScript string `json:"dialog_script"`
}

var actorDefs = make(map[string]ActorDef)

// PreloadActorDefs pre-loads actor definitions from data/json/actors.json at program start.
func PreloadActorDefs() (map[string]ActorDef, error) {
	return LoadActorDefs("json/actors.json")
}

// LoadActorDefs loads actor definitions from a JSON file in data.FS.
func LoadActorDefs(path string) (map[string]ActorDef, error) {
	cleanPath := strings.TrimPrefix(path, "data/")
	if !strings.HasPrefix(cleanPath, "json/") {
		cleanPath = "json/" + cleanPath
	}

	b, err := data.FS.ReadFile(cleanPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read actor defs file %s: %w", path, err)
	}

	var defs map[string]ActorDef
	if err := json.Unmarshal(b, &defs); err != nil {
		return nil, fmt.Errorf("failed to parse actor defs JSON %s: %w", path, err)
	}

	for k, v := range defs {
		actorDefs[k] = v
	}

	return actorDefs, nil
}

// GetActorDef returns the ActorDef for a given key from the loaded definitions.
func GetActorDef(key string) (ActorDef, bool) {
	def, ok := actorDefs[key]
	return def, ok
}

// NewActor creates a new Actor with specified ID, position, and sprite definition name.
func NewActor(id string, x, y int, spriteName string) *Actor {
	sd, ok := GetSpriteDef(spriteName)
	if !ok {
		// Fallback sprite definition if not found
		sd = SpriteDef{Tile: 0, Animated: false, Frames: 1}
	}

	return &Actor{
		Entity: Entity{
			Name:      id,
			SpriteDef: sd,
			X:         x,
			Y:         y,
		},
		ID: id,
	}
}

// NewActorFromDef creates a new Actor using a preloaded ActorDef key.
func NewActorFromDef(id string, defKey string, x, y int) (*Actor, error) {
	def, ok := actorDefs[defKey]
	if !ok {
		return nil, fmt.Errorf("actor definition %q not found", defKey)
	}

	actor := NewActor(id, x, y, def.Sprite)
	if def.Name != "" {
		actor.Name = def.Name
	}
	actor.DialogScript = def.DialogScript
	return actor, nil
}
