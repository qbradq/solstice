package solstice

import (
	"encoding/json"
	"fmt"
	"strings"

	"solstice/data"
)

// FlexibleBool unmarshals both boolean and string representations of booleans (e.g. true, "true").
type FlexibleBool bool

func (b *FlexibleBool) UnmarshalJSON(data []byte) error {
	s := strings.ToLower(strings.Trim(string(data), `"`))
	if s == "true" || s == "1" {
		*b = true
		return nil
	}
	*b = false
	return nil
}

// ActorDef represents an actor definition loaded from JSON.
type ActorDef struct {
	Name           string       `json:"name"`
	Sprite         string       `json:"sprite"` // Key into sprites.json
	Human          FlexibleBool `json:"human,omitempty"`
	DialogScript   string       `json:"dialog_script"` // Path to dialog script
	IdleScript     string       `json:"idle_script,omitempty"`
	CombatScript   string       `json:"combat_script,omitempty"`
	Experience     int          `json:"experience"`
	Level          int          `json:"level"`
	Strength       int          `json:"strength"`
	Dexterity      int          `json:"dexterity"`
	Intelligence   int          `json:"intelligence"`
	MaxHitPoints   int          `json:"max_hit_points"`
	HitPoints      int          `json:"hit_points"`
	MaxMagicPoints int          `json:"max_magic_points"`
	MagicPoints    int          `json:"magic_points"`
	Move           int          `json:"move"`
	Equipment      []string     `json:"equipment,omitempty"`
}

// Actor represents an agent or character in the game world (NPC, monster, party member).
// It embeds Entity for position and graphical representation.
type Actor struct {
	Entity
	Human          bool     `json:"human,omitempty"`
	DialogScript   string   `json:"dialog_script,omitempty"`
	IdleScript     string   `json:"idle_script,omitempty"`
	CombatScript   string   `json:"combat_script,omitempty"`
	Experience     int      `json:"experience"`
	Level          int      `json:"level"`
	Strength       int      `json:"strength"`
	Dexterity      int      `json:"dexterity"`
	Intelligence   int      `json:"intelligence"`
	MaxHitPoints   int      `json:"max_hit_points"`
	HitPoints      int      `json:"hit_points"`
	MaxMagicPoints int      `json:"max_magic_points"`
	MagicPoints    int      `json:"magic_points"`
	Move           int      `json:"move"`
	Weapon         *Item    `json:"weapon,omitempty"`
	Equipment      []string `json:"equipment,omitempty"`
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
			ID:        id,
			Name:      id,
			SpriteDef: sd,
			X:         x,
			Y:         y,
		},
		Level: 1,
		Move:  3,
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
	actor.IdleScript = def.IdleScript
	actor.CombatScript = def.CombatScript
	actor.Human = bool(def.Human)
	actor.Experience = def.Experience
	actor.Level = def.Level
	if actor.Level == 0 {
		actor.Level = 1
	}
	actor.Strength = def.Strength
	actor.Dexterity = def.Dexterity
	actor.Intelligence = def.Intelligence
	actor.MaxHitPoints = def.MaxHitPoints
	actor.HitPoints = def.HitPoints
	if actor.HitPoints == 0 && actor.MaxHitPoints > 0 {
		actor.HitPoints = actor.MaxHitPoints
	}
	actor.MaxMagicPoints = def.MaxMagicPoints
	actor.MagicPoints = def.MagicPoints
	if actor.MagicPoints == 0 && actor.MaxMagicPoints > 0 {
		actor.MagicPoints = actor.MaxMagicPoints
	}
	actor.Move = def.Move
	if actor.Move == 0 {
		actor.Move = 3
	}

	if len(def.Equipment) > 0 {
		actor.Equipment = make([]string, len(def.Equipment))
		copy(actor.Equipment, def.Equipment)
		for _, eqTmpl := range def.Equipment {
			if itemDef, ok := GetItemDef(eqTmpl); ok && itemDef.Type == "weapon" {
				actor.Weapon = NewItem(fmt.Sprintf("%s_weapon", id), eqTmpl, 0, 0)
				break
			}
		}
	}

	return actor, nil
}

// EquipWeapon equips a weapon item to the actor if its item definition has type "weapon".
func (a *Actor) EquipWeapon(item *Item) error {
	if item == nil {
		a.Weapon = nil
		return nil
	}
	if item.Type != "weapon" {
		if def, ok := GetItemDef(item.Template); ok && def.Type == "weapon" {
			item.Type = "weapon"
		} else {
			return fmt.Errorf("item %q is not a weapon (type %q)", item.Template, item.Type)
		}
	}
	a.Weapon = item
	return nil
}

// GetWeaponRange returns the range of the equipped weapon, or 1 if unequipped / undefined.
func (a *Actor) GetWeaponRange() int {
	if a != nil && a.Weapon != nil && a.Weapon.Meta != nil {
		if rVal, ok := a.Weapon.Meta["range"]; ok {
			switch v := rVal.(type) {
			case int:
				if v > 0 {
					return v
				}
			case int64:
				if v > 0 {
					return int(v)
				}
			case float64:
				if v > 0 {
					return int(v)
				}
			}
		}
	}
	return 1
}

// GetWeaponDamage returns the damage formula of the equipped weapon, or "1d4" if unequipped / undefined.
func (a *Actor) GetWeaponDamage() string {
	if a != nil && a.Weapon != nil && a.Weapon.Meta != nil {
		if dVal, ok := a.Weapon.Meta["damage"]; ok {
			if str, ok := dVal.(string); ok && str != "" {
				return str
			}
		}
	}
	return "1d4"
}
