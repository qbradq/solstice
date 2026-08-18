package solstice

import (
	"encoding/json"
	"fmt"
	"strings"

	"solstice/data"
)

// ItemDef represents an item template definition loaded from JSON.
type ItemDef struct {
	Name   string `json:"name,omitempty"`
	Sprite string `json:"sprite,omitempty"` // Key into sprites.json
}

// Item represents a map entity item.
type Item struct {
	Entity
	Template string `json:"template"`
}

var itemDefs = make(map[string]ItemDef)

// PreloadItemDefs pre-loads item definitions from data/json/items.json at program start.
func PreloadItemDefs() (map[string]ItemDef, error) {
	return LoadItemDefs("json/items.json")
}

// LoadItemDefs loads item definitions from a JSON file in data.FS.
func LoadItemDefs(path string) (map[string]ItemDef, error) {
	cleanPath := strings.TrimPrefix(path, "data/")
	if !strings.HasPrefix(cleanPath, "json/") {
		cleanPath = "json/" + cleanPath
	}

	b, err := data.FS.ReadFile(cleanPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read item defs file %s: %w", path, err)
	}

	var defs map[string]ItemDef
	if err := json.Unmarshal(b, &defs); err != nil {
		return nil, fmt.Errorf("failed to parse item defs JSON %s: %w", path, err)
	}

	for k, v := range defs {
		itemDefs[k] = v
	}

	return itemDefs, nil
}

// GetItemDef returns the ItemDef for a given key from loaded definitions.
func GetItemDef(key string) (ItemDef, bool) {
	def, ok := itemDefs[key]
	return def, ok
}

// NewItem creates a new Item instance for the given ID, template, and coordinates.
func NewItem(id string, template string, x, y int) *Item {
	name := id
	var sd SpriteDef
	if def, ok := GetItemDef(template); ok {
		if def.Name != "" {
			name = def.Name
		}
		if def.Sprite != "" {
			if s, found := GetSpriteDef(def.Sprite); found {
				sd = s
			}
		}
	} else if s, found := GetSpriteDef(template); found {
		sd = s
	}

	return &Item{
		Entity: Entity{
			ID:        id,
			Name:      name,
			SpriteDef: sd,
			X:         x,
			Y:         y,
		},
		Template: template,
	}
}
