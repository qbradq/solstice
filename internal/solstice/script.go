package solstice

import (
	"fmt"
	"io/fs"
	"math/rand"
	"path/filepath"
	"strings"

	"solstice/data"

	"github.com/d5/tengo/v2"
	"github.com/justinian/dice"
)

var (
	moduleMap        *tengo.ModuleMap
	compiledScripts  = make(map[string]*tengo.Compiled)
	rawScriptSources = make(map[string][]byte)
	gameState        = make(map[string]bool)
	gameStrings      = make(map[string]string)
	dialogEnded      bool
	currentAIActor   *Actor
)

// GetCurrentAIActor returns the actor whose AI script is currently executing.
func GetCurrentAIActor() *Actor {
	return currentAIActor
}

// SetCurrentAIActor sets the actor whose AI script is currently executing.
func SetCurrentAIActor(a *Actor) {
	currentAIActor = a
}

// RunActorAIScript executes an AI script for the specified actor with runtime globals actor_id, tile_x, and tile_y.
func RunActorAIScript(actor *Actor, scriptPath string) error {
	if actor == nil || scriptPath == "" {
		return nil
	}

	prevActor := currentAIActor
	currentAIActor = actor
	defer func() {
		currentAIActor = prevActor
	}()

	globals := map[string]interface{}{
		"actor_id": actor.ID,
		"tile_x":   actor.X,
		"tile_y":   actor.Y,
	}

	return ExecuteScriptWithGlobals(scriptPath, globals)
}

// EndDialog flags the active dialog to terminate.
func EndDialog() {
	dialogEnded = true
}

// IsDialogEnded returns true if end_dialog has been called.
func IsDialogEnded() bool {
	return dialogEnded
}

// SetFlag creates and sets the named flag to true.
func SetFlag(name string) {
	gameState[name] = true
}

// ClearFlag removes the named flag.
func ClearFlag(name string) {
	delete(gameState, name)
}

// ToggleFlag removes the named flag if it exists; otherwise creates it and sets it to true.
func ToggleFlag(name string) {
	if gameState[name] {
		delete(gameState, name)
	} else {
		gameState[name] = true
	}
}

// HasFlag returns true if the named flag exists and is true.
func HasFlag(name string) bool {
	return gameState[name]
}

// ClearAllFlags resets all flags.
func ClearAllFlags() {
	gameState = make(map[string]bool)
}

// GetAllFlags returns a copy of all current game flags.
func GetAllFlags() map[string]bool {
	res := make(map[string]bool, len(gameState))
	for k, v := range gameState {
		res[k] = v
	}
	return res
}

// RestoreFlags restores game flags from saved state.
func RestoreFlags(flags map[string]bool) {
	gameState = make(map[string]bool, len(flags))
	for k, v := range flags {
		gameState[k] = v
	}
}

// SetString creates or sets the persistent state string value named name to value.
func SetString(name, value string) {
	gameStrings[name] = value
}

// ClearString removes the persistent state string value named name.
func ClearString(name string) {
	delete(gameStrings, name)
}

// GetString returns the named persistent state string value or empty string if absent.
func GetString(name string) string {
	return gameStrings[name]
}

// ClearAllStrings resets all persistent string values.
func ClearAllStrings() {
	gameStrings = make(map[string]string)
}

// GetAllStrings returns a copy of all persistent string values.
func GetAllStrings() map[string]string {
	res := make(map[string]string, len(gameStrings))
	for k, v := range gameStrings {
		res[k] = v
	}
	return res
}

// RestoreStrings restores persistent string values from saved state.
func RestoreStrings(strings map[string]string) {
	gameStrings = make(map[string]string, len(strings))
	for k, v := range strings {
		gameStrings[k] = v
	}
}

// Backward-compatible aliases for state functions
func SetState(name string)      { SetFlag(name) }
func ClearState(name string)    { ClearFlag(name) }
func ToggleState(name string)   { ToggleFlag(name) }
func HasState(name string) bool { return HasFlag(name) }
func ClearAllState()            { ClearAllFlags() }

// GetScriptModuleMap returns the Tengo ModuleMap containing all registered modules.
func GetScriptModuleMap() *tengo.ModuleMap {
	return moduleMap
}

// InitScriptSystem initializes the Tengo scripting system by recursively loading and pre-compiling
// all .tengo files in the data/scripts directory from data.FS.
func InitScriptSystem() error {

	moduleMap = tengo.NewModuleMap()

	// Register builtin "game" module
	gameModule := map[string]tengo.Object{
		"log": &tengo.UserFunction{
			Name: "log",
			Value: func(args ...tengo.Object) (tengo.Object, error) {
				if len(args) == 0 {
					return tengo.UndefinedValue, nil
				}
				format, ok := tengo.ToString(args[0])
				if !ok {
					format = args[0].String()
				}
				var msg string
				if len(args) == 1 {
					msg = format
				} else {
					formatted, err := tengo.Format(format, args[1:]...)
					if err != nil {
						return tengo.UndefinedValue, err
					}
					msg = formatted
				}
				if defaultTerminal != nil {
					defaultTerminal.AddMessage(msg)
				}
				return tengo.UndefinedValue, nil
			},
		},
		"set_map_tile": &tengo.UserFunction{
			Name: "set_map_tile",
			Value: func(args ...tengo.Object) (tengo.Object, error) {
				if len(args) < 3 {
					return tengo.UndefinedValue, fmt.Errorf("set_map_tile requires 3 arguments: x, y, tileIdx")
				}
				x, ok1 := tengo.ToInt(args[0])
				y, ok2 := tengo.ToInt(args[1])
				tileIdx, ok3 := tengo.ToInt(args[2])
				if !ok1 || !ok2 || !ok3 {
					return tengo.UndefinedValue, fmt.Errorf("set_map_tile arguments must be integers")
				}
				if defaultMap != nil {
					defaultMap.SetTile(x, y, tileIdx)
				}
				return tengo.UndefinedValue, nil
			},
		},
		"add_timer": &tengo.UserFunction{
			Name: "add_timer",
			Value: func(args ...tengo.Object) (tengo.Object, error) {
				if len(args) < 2 {
					return tengo.UndefinedValue, fmt.Errorf("add_timer requires at least 2 arguments: delayTurns, scriptName, [globals]")
				}
				delayTurns, ok1 := tengo.ToInt(args[0])
				scriptName, ok2 := tengo.ToString(args[1])
				if !ok1 || !ok2 {
					return tengo.UndefinedValue, fmt.Errorf("add_timer arguments: delayTurns (int), scriptName (string)")
				}

				globalsMap := make(map[string]interface{})
				if len(args) >= 3 && args[2] != tengo.UndefinedValue {
					if objMap, ok := args[2].(*tengo.Map); ok {
						for k, v := range objMap.Value {
							globalsMap[k] = tengo.ToInterface(v)
						}
					}
				}

				if defaultMap != nil {
					defaultMap.AddTimer(delayTurns, scriptName, globalsMap)
				}

				return tengo.UndefinedValue, nil
			},
		},
		"set_flag": &tengo.UserFunction{
			Name: "set_flag",
			Value: func(args ...tengo.Object) (tengo.Object, error) {
				if len(args) < 1 {
					return tengo.UndefinedValue, fmt.Errorf("set_flag requires 1 argument: name")
				}
				name, ok := tengo.ToString(args[0])
				if !ok {
					return tengo.UndefinedValue, fmt.Errorf("set_flag argument must be a string")
				}
				SetState(name)
				return tengo.UndefinedValue, nil
			},
		},
		"clear_flag": &tengo.UserFunction{
			Name: "clear_flag",
			Value: func(args ...tengo.Object) (tengo.Object, error) {
				if len(args) < 1 {
					return tengo.UndefinedValue, fmt.Errorf("clear_flag requires 1 argument: name")
				}
				name, ok := tengo.ToString(args[0])
				if !ok {
					return tengo.UndefinedValue, fmt.Errorf("clear_flag argument must be a string")
				}
				ClearState(name)
				return tengo.UndefinedValue, nil
			},
		},
		"toggle_flag": &tengo.UserFunction{
			Name: "toggle_flag",
			Value: func(args ...tengo.Object) (tengo.Object, error) {
				if len(args) < 1 {
					return tengo.UndefinedValue, fmt.Errorf("toggle_flag requires 1 argument: name")
				}
				name, ok := tengo.ToString(args[0])
				if !ok {
					return tengo.UndefinedValue, fmt.Errorf("toggle_flag argument must be a string")
				}
				ToggleState(name)
				return tengo.UndefinedValue, nil
			},
		},
		"has_flag": &tengo.UserFunction{
			Name: "has_flag",
			Value: func(args ...tengo.Object) (tengo.Object, error) {
				if len(args) < 1 {
					return tengo.FalseValue, nil
				}
				name, ok := tengo.ToString(args[0])
				if !ok {
					return tengo.FalseValue, nil
				}
				if HasState(name) {
					return tengo.TrueValue, nil
				}
				return tengo.FalseValue, nil
			},
		},
		"set_string": &tengo.UserFunction{
			Name: "set_string",
			Value: func(args ...tengo.Object) (tengo.Object, error) {
				if len(args) < 2 {
					return tengo.UndefinedValue, fmt.Errorf("set_string requires 2 arguments: name, value")
				}
				name, ok1 := tengo.ToString(args[0])
				val, ok2 := tengo.ToString(args[1])
				if !ok1 || !ok2 {
					return tengo.UndefinedValue, fmt.Errorf("set_string arguments must be strings")
				}
				SetString(name, val)
				return tengo.UndefinedValue, nil
			},
		},
		"clear_string": &tengo.UserFunction{
			Name: "clear_string",
			Value: func(args ...tengo.Object) (tengo.Object, error) {
				if len(args) < 1 {
					return tengo.UndefinedValue, fmt.Errorf("clear_string requires 1 argument: name")
				}
				name, ok := tengo.ToString(args[0])
				if !ok {
					return tengo.UndefinedValue, fmt.Errorf("clear_string argument must be a string")
				}
				ClearString(name)
				return tengo.UndefinedValue, nil
			},
		},
		"get_string": &tengo.UserFunction{
			Name: "get_string",
			Value: func(args ...tengo.Object) (tengo.Object, error) {
				if len(args) < 1 {
					return tengo.UndefinedValue, fmt.Errorf("get_string requires 1 argument: name")
				}
				name, ok := tengo.ToString(args[0])
				if !ok {
					return tengo.UndefinedValue, fmt.Errorf("get_string argument must be a string")
				}
				val := GetString(name)
				return &tengo.String{Value: val}, nil
			},
		},
		"end_dialog": &tengo.UserFunction{
			Name: "end_dialog",
			Value: func(args ...tengo.Object) (tengo.Object, error) {
				EndDialog()
				return tengo.UndefinedValue, nil
			},
		},
		"random": &tengo.UserFunction{
			Name: "random",
			Value: func(args ...tengo.Object) (tengo.Object, error) {
				if len(args) == 0 {
					return tengo.UndefinedValue, nil
				}
				if len(args) == 1 {
					if arr, ok := args[0].(*tengo.Array); ok {
						if len(arr.Value) == 0 {
							return tengo.UndefinedValue, nil
						}
						idx := rand.Intn(len(arr.Value))
						return arr.Value[idx], nil
					}
					return args[0], nil
				}
				if len(args) == 2 {
					minInt, ok1 := tengo.ToInt(args[0])
					maxInt, ok2 := tengo.ToInt(args[1])
					if ok1 && ok2 {
						if minInt > maxInt {
							minInt, maxInt = maxInt, minInt
						}
						n := rand.Intn(maxInt-minInt+1) + minInt
						return &tengo.Int{Value: int64(n)}, nil
					}
				}
				idx := rand.Intn(len(args))
				return args[idx], nil
			},
		},
		"roll": &tengo.UserFunction{
			Name: "roll",
			Value: func(args ...tengo.Object) (tengo.Object, error) {
				if len(args) < 1 {
					return tengo.UndefinedValue, fmt.Errorf("roll requires 1 argument: expression")
				}
				expr, ok := tengo.ToString(args[0])
				if !ok {
					return tengo.UndefinedValue, fmt.Errorf("roll argument must be a string")
				}
				res, _, err := dice.Roll(expr)
				if err != nil {
					return tengo.UndefinedValue, fmt.Errorf("roll error for %q: %w", expr, err)
				}
				return &tengo.Int{Value: int64(res.Int())}, nil
			},
		},
		"get_enemies_for_pack": &tengo.UserFunction{
			Name: "get_enemies_for_pack",
			Value: func(args ...tengo.Object) (tengo.Object, error) {
				if len(args) < 1 {
					return tengo.UndefinedValue, fmt.Errorf("get_enemies_for_pack requires 1 argument: name")
				}
				packName, ok := tengo.ToString(args[0])
				if !ok {
					return tengo.UndefinedValue, fmt.Errorf("get_enemies_for_pack argument must be a string")
				}

				pack, exists := GetEnemyPack(packName)
				if !exists {
					return &tengo.Array{Value: []tengo.Object{}}, nil
				}

				numEnemies, err := pack.RollNumEnemies()
				if err != nil {
					return tengo.UndefinedValue, fmt.Errorf("failed to roll num enemies for pack %s: %w", packName, err)
				}

				arr := make([]tengo.Object, numEnemies)
				for i := 0; i < numEnemies; i++ {
					tmpl := pack.ChooseEnemy()
					arr[i] = &tengo.String{Value: tmpl}
				}

				return &tengo.Array{Value: arr}, nil
			},
		},
		"load_map": &tengo.UserFunction{
			Name: "load_map",
			Value: func(args ...tengo.Object) (tengo.Object, error) {
				if len(args) < 1 {
					return tengo.UndefinedValue, fmt.Errorf("load_map requires 1 argument: name")
				}
				mapName, ok := tengo.ToString(args[0])
				if !ok {
					return tengo.UndefinedValue, fmt.Errorf("load_map argument must be a string")
				}
				m, err := LoadMap(mapName)
				if err != nil {
					return tengo.UndefinedValue, fmt.Errorf("failed to load map %s: %w", mapName, err)
				}
				SetMap(m)
				return tengo.UndefinedValue, nil
			},
		},
		"reload_map": &tengo.UserFunction{
			Name: "reload_map",
			Value: func(args ...tengo.Object) (tengo.Object, error) {
				if len(args) < 1 {
					return tengo.UndefinedValue, fmt.Errorf("reload_map requires 1 argument: name")
				}
				mapName, ok := tengo.ToString(args[0])
				if !ok {
					return tengo.UndefinedValue, fmt.Errorf("reload_map argument must be a string")
				}
				_, err := ReloadMap(mapName)
				if err != nil {
					return tengo.UndefinedValue, fmt.Errorf("failed to reload map %s: %w", mapName, err)
				}
				return tengo.UndefinedValue, nil
			},
		},
		"teleport_party": &tengo.UserFunction{
			Name: "teleport_party",
			Value: func(args ...tengo.Object) (tengo.Object, error) {
				if len(args) < 2 {
					return tengo.UndefinedValue, fmt.Errorf("teleport_party requires 2 arguments: x, y")
				}
				x, ok1 := tengo.ToInt(args[0])
				y, ok2 := tengo.ToInt(args[1])
				if !ok1 || !ok2 {
					return tengo.UndefinedValue, fmt.Errorf("teleport_party arguments must be integers")
				}
				party := GetParty()
				if party != nil {
					party.X = x
					party.Y = y
				}
				return tengo.UndefinedValue, nil
			},
		},
		"teleport_party_on_world_map": &tengo.UserFunction{
			Name: "teleport_party_on_world_map",
			Value: func(args ...tengo.Object) (tengo.Object, error) {
				if len(args) < 2 {
					return tengo.UndefinedValue, fmt.Errorf("teleport_party_on_world_map requires 2 arguments: x, y")
				}
				x, ok1 := tengo.ToInt(args[0])
				y, ok2 := tengo.ToInt(args[1])
				if !ok1 || !ok2 {
					return tengo.UndefinedValue, fmt.Errorf("teleport_party_on_world_map arguments must be integers")
				}
				party := GetParty()
				if party != nil {
					party.WorldX = x
					party.WorldY = y
				}
				return tengo.UndefinedValue, nil
			},
		},
		"add_to_party": &tengo.UserFunction{
			Name: "add_to_party",
			Value: func(args ...tengo.Object) (tengo.Object, error) {
				if len(args) < 1 {
					return tengo.UndefinedValue, fmt.Errorf("add_to_party requires 1 argument: actor_id")
				}
				actorID, ok := tengo.ToString(args[0])
				if !ok {
					return tengo.UndefinedValue, fmt.Errorf("add_to_party argument must be a string")
				}

				party := GetParty()
				if party == nil {
					return tengo.UndefinedValue, fmt.Errorf("party is not initialized")
				}

				if len(party.Members) >= MaxPartyMembers {
					if defaultTerminal != nil {
						defaultTerminal.AddMessage("Too many party members!")
					}
					return tengo.UndefinedValue, nil
				}

				m := GetMap()
				var actor *Actor
				if m != nil {
					actor = m.GetActorByID(actorID)
				}
				if actor == nil {
					if _, ok := GetActorDef(actorID); ok {
						actor, _ = NewActorFromDef(actorID, actorID, 0, 0)
					}
				}
				if actor == nil {
					return tengo.UndefinedValue, fmt.Errorf("actor %q not found", actorID)
				}

				name := actor.Name
				if name == "" {
					name = actor.ID
				}

				if err := party.AddMember(*actor); err != nil {
					if defaultTerminal != nil {
						defaultTerminal.AddMessage("Too many party members!")
					}
					return tengo.UndefinedValue, nil
				}

				if m != nil {
					m.RemoveActorByID(actor.ID)
				}

				if defaultTerminal != nil {
					defaultTerminal.AddMessage(fmt.Sprintf("%s joins the party!", name))
				}

				return tengo.UndefinedValue, nil
			},
		},
		"start_combat": &tengo.UserFunction{
			Name: "start_combat",
			Value: func(args ...tengo.Object) (tengo.Object, error) {
				StartCombat()
				return tengo.UndefinedValue, nil
			},
		},
		"stop_combat": &tengo.UserFunction{
			Name: "stop_combat",
			Value: func(args ...tengo.Object) (tengo.Object, error) {
				StopCombat()
				return tengo.UndefinedValue, nil
			},
		},
		"get_party_members": &tengo.UserFunction{
			Name: "get_party_members",
			Value: func(args ...tengo.Object) (tengo.Object, error) {
				party := GetParty()
				if party == nil {
					return &tengo.Array{Value: []tengo.Object{}}, nil
				}
				arr := make([]tengo.Object, len(party.Members))
				for i, m := range party.Members {
					arr[i] = &tengo.Map{
						Value: map[string]tengo.Object{
							"id":   &tengo.String{Value: m.ID},
							"name": &tengo.String{Value: m.Name},
							"x":    &tengo.Int{Value: int64(m.X)},
							"y":    &tengo.Int{Value: int64(m.Y)},
						},
					}
				}
				return &tengo.Array{Value: arr}, nil
			},
		},
		"teleport_actor": &tengo.UserFunction{
			Name: "teleport_actor",
			Value: func(args ...tengo.Object) (tengo.Object, error) {
				if len(args) < 3 {
					return tengo.UndefinedValue, fmt.Errorf("teleport_actor requires 3 arguments: actor_id, x, y")
				}
				actorID, ok1 := tengo.ToString(args[0])
				x, ok2 := tengo.ToInt(args[1])
				y, ok3 := tengo.ToInt(args[2])
				if !ok1 || !ok2 || !ok3 {
					return tengo.UndefinedValue, fmt.Errorf("teleport_actor arguments must be (string, int, int)")
				}
				if m := GetMap(); m != nil {
					if actor := m.GetActorByID(actorID); actor != nil {
						actor.X = x
						actor.Y = y
					}
				}
				if party := GetParty(); party != nil {
					for i := range party.Members {
						if party.Members[i].ID == actorID {
							party.Members[i].X = x
							party.Members[i].Y = y
						}
					}
				}
				return tengo.UndefinedValue, nil
			},
		},
		"get_party": &tengo.UserFunction{
			Name: "get_party",
			Value: func(args ...tengo.Object) (tengo.Object, error) {
				party := GetParty()
				if party == nil {
					return &tengo.Array{Value: []tengo.Object{}}, nil
				}
				arr := make([]tengo.Object, len(party.Members))
				for i, m := range party.Members {
					arr[i] = &tengo.Map{
						Value: map[string]tengo.Object{
							"id":   &tengo.String{Value: m.ID},
							"name": &tengo.String{Value: m.Name},
							"x":    &tengo.Int{Value: int64(m.X)},
							"y":    &tengo.Int{Value: int64(m.Y)},
						},
					}
				}
				return &tengo.Array{Value: arr}, nil
			},
		},
		"get_actor": &tengo.UserFunction{
			Name: "get_actor",
			Value: func(args ...tengo.Object) (tengo.Object, error) {
				if len(args) < 1 {
					return tengo.UndefinedValue, fmt.Errorf("get_actor requires 1 argument: actor_id")
				}
				actorID, ok := tengo.ToString(args[0])
				if !ok {
					return tengo.UndefinedValue, fmt.Errorf("get_actor argument must be a string")
				}

				var actor *Actor
				m := GetMap()
				if m != nil {
					actor = m.GetActorByID(actorID)
				}
				if actor == nil {
					p := GetParty()
					if p != nil {
						for i := range p.Members {
							if p.Members[i].ID == actorID {
								actor = &p.Members[i]
								break
							}
						}
					}
				}

				if actor == nil {
					return tengo.UndefinedValue, nil
				}

				humanVal := tengo.FalseValue
				if actor.Human {
					humanVal = tengo.TrueValue
				}

				return &tengo.Map{
					Value: map[string]tengo.Object{
						"id":               &tengo.String{Value: actor.ID},
						"name":             &tengo.String{Value: actor.Name},
						"human":            humanVal,
						"level":            &tengo.Int{Value: int64(actor.Level)},
						"strength":         &tengo.Int{Value: int64(actor.Strength)},
						"dexterity":        &tengo.Int{Value: int64(actor.Dexterity)},
						"intelligence":     &tengo.Int{Value: int64(actor.Intelligence)},
						"max_hit_points":   &tengo.Int{Value: int64(actor.MaxHitPoints)},
						"hit_points":       &tengo.Int{Value: int64(actor.HitPoints)},
						"max_magic_points": &tengo.Int{Value: int64(actor.MaxMagicPoints)},
						"magic_points":     &tengo.Int{Value: int64(actor.MagicPoints)},
						"range":            &tengo.Int{Value: int64(actor.GetWeaponRange())},
						"damage":           &tengo.String{Value: actor.GetWeaponDamage()},
					},
				}, nil
			},
		},
		"damage_actor": &tengo.UserFunction{
			Name: "damage_actor",
			Value: func(args ...tengo.Object) (tengo.Object, error) {
				if len(args) < 2 {
					return tengo.UndefinedValue, fmt.Errorf("damage_actor requires 2 arguments: actor_id, amount")
				}
				actorID, ok1 := tengo.ToString(args[0])
				amount, ok2 := tengo.ToInt(args[1])
				if !ok1 || !ok2 {
					return tengo.UndefinedValue, fmt.Errorf("damage_actor arguments must be (string, int)")
				}

				var actor *Actor
				m := GetMap()
				if m != nil {
					actor = m.GetActorByID(actorID)
				}
				var partyMember *Actor
				p := GetParty()
				if p != nil {
					for i := range p.Members {
						if p.Members[i].ID == actorID {
							partyMember = &p.Members[i]
							if actor == nil {
								actor = partyMember
							}
							break
						}
					}
				}

				if actor == nil {
					return tengo.UndefinedValue, nil
				}

				actor.HitPoints -= amount
				if partyMember != nil {
					partyMember.HitPoints = actor.HitPoints
				}

				if actor.HitPoints <= 0 {
					if m != nil {
						corpseTmpl := "animal_corpse"
						if actor.Human {
							corpseTmpl = "human_corpse"
						}
						corpseID := m.GenerateUniqueItemID(fmt.Sprintf("%s_corpse", actor.ID))
						item := NewItem(corpseID, corpseTmpl, actor.X, actor.Y)
						m.RemoveActor(actor)
						m.AddItem(item)
					}
				}

				return tengo.UndefinedValue, nil
			},
		},
		"set_actor_pos": &tengo.UserFunction{
			Name: "set_actor_pos",
			Value: func(args ...tengo.Object) (tengo.Object, error) {
				if len(args) < 3 {
					return tengo.UndefinedValue, fmt.Errorf("set_actor_pos requires 3 arguments: actor_id, x, y")
				}
				actorID, ok1 := tengo.ToString(args[0])
				x, ok2 := tengo.ToInt(args[1])
				y, ok3 := tengo.ToInt(args[2])
				if !ok1 || !ok2 || !ok3 {
					return tengo.UndefinedValue, fmt.Errorf("set_actor_pos arguments must be (string, int, int)")
				}
				if m := GetMap(); m != nil {
					if actor := m.GetActorByID(actorID); actor != nil {
						actor.X = x
						actor.Y = y
					}
				}
				if party := GetParty(); party != nil {
					for i := range party.Members {
						if party.Members[i].ID == actorID {
							party.Members[i].X = x
							party.Members[i].Y = y
						}
					}
				}
				return tengo.UndefinedValue, nil
			},
		},
		"start_dialog": &tengo.UserFunction{
			Name: "start_dialog",
			Value: func(args ...tengo.Object) (tengo.Object, error) {
				if len(args) < 1 {
					return tengo.UndefinedValue, fmt.Errorf("start_dialog requires at least 1 argument: [actor_id], dialog_script")
				}
				str1, ok1 := tengo.ToString(args[0])
				if !ok1 {
					return tengo.UndefinedValue, fmt.Errorf("start_dialog arguments must be strings")
				}

				scriptPath := str1
				actorID := ""
				if len(args) >= 2 {
					str2, ok2 := tengo.ToString(args[1])
					if ok2 {
						if strings.HasSuffix(str2, ".tengo") || strings.Contains(str2, "/") {
							scriptPath = str2
							actorID = str1
						} else {
							scriptPath = str1
							actorID = str2
						}
					}
				}

				m := GetMap()
				var actor *Actor
				if m != nil && actorID != "" {
					actor = m.GetActorByID(actorID)
				}
				if actor == nil && actorID != "" {
					if _, ok := GetActorDef(actorID); ok {
						actor, _ = NewActorFromDef(actorID, actorID, 0, 0)
					}
				}
				if actor == nil {
					actor = &Actor{
						Entity:       Entity{ID: actorID},
						DialogScript: scriptPath,
					}
				}
				if scriptPath == "" && actor.DialogScript != "" {
					scriptPath = actor.DialogScript
				}

				dialogMode := NewDialogMode(actor, scriptPath)
				if g := GetGame(); g != nil {
					g.PushMode(dialogMode)
				}
				return tengo.UndefinedValue, nil
			},
		},
		"force_dialog": &tengo.UserFunction{
			Name: "force_dialog",
			Value: func(args ...tengo.Object) (tengo.Object, error) {
				if len(args) < 1 {
					return tengo.UndefinedValue, fmt.Errorf("force_dialog requires at least 1 argument: [actor_id], dialog_script")
				}
				str1, ok1 := tengo.ToString(args[0])
				if !ok1 {
					return tengo.UndefinedValue, fmt.Errorf("force_dialog arguments must be strings")
				}

				scriptPath := str1
				actorID := ""
				if len(args) >= 2 {
					str2, ok2 := tengo.ToString(args[1])
					if ok2 {
						if strings.HasSuffix(str2, ".tengo") || strings.Contains(str2, "/") {
							scriptPath = str2
							actorID = str1
						} else {
							scriptPath = str1
							actorID = str2
						}
					}
				}

				m := GetMap()
				var actor *Actor
				if m != nil && actorID != "" {
					actor = m.GetActorByID(actorID)
				}
				if actor == nil && actorID != "" {
					if _, ok := GetActorDef(actorID); ok {
						actor, _ = NewActorFromDef(actorID, actorID, 0, 0)
					}
				}
				if actor == nil {
					actor = &Actor{
						Entity:       Entity{ID: actorID},
						DialogScript: scriptPath,
					}
				}
				if scriptPath == "" && actor.DialogScript != "" {
					scriptPath = actor.DialogScript
				}

				dialogMode := NewDialogMode(actor, scriptPath)
				if g := GetGame(); g != nil {
					g.PushMode(dialogMode)
				}
				return tengo.UndefinedValue, nil
			},
		},
		"enter_dialog": &tengo.UserFunction{
			Name: "enter_dialog",
			Value: func(args ...tengo.Object) (tengo.Object, error) {
				if len(args) < 1 {
					return tengo.UndefinedValue, fmt.Errorf("enter_dialog requires at least 1 argument: [actor_id], dialog_script")
				}
				str1, ok1 := tengo.ToString(args[0])
				if !ok1 {
					return tengo.UndefinedValue, fmt.Errorf("enter_dialog arguments must be strings")
				}

				scriptPath := str1
				actorID := ""
				if len(args) >= 2 {
					str2, ok2 := tengo.ToString(args[1])
					if ok2 {
						if strings.HasSuffix(str2, ".tengo") || strings.Contains(str2, "/") {
							scriptPath = str2
							actorID = str1
						} else {
							scriptPath = str1
							actorID = str2
						}
					}
				}

				m := GetMap()
				var actor *Actor
				if m != nil && actorID != "" {
					actor = m.GetActorByID(actorID)
				}
				if actor == nil && actorID != "" {
					if _, ok := GetActorDef(actorID); ok {
						actor, _ = NewActorFromDef(actorID, actorID, 0, 0)
					}
				}
				if actor == nil {
					actor = &Actor{
						Entity:       Entity{ID: actorID},
						DialogScript: scriptPath,
					}
				}
				if scriptPath == "" && actor.DialogScript != "" {
					scriptPath = actor.DialogScript
				}

				dialogMode := NewDialogMode(actor, scriptPath)
				if g := GetGame(); g != nil {
					g.PushMode(dialogMode)
				}
				return tengo.UndefinedValue, nil
			},
		},
		"spawn_actor": &tengo.UserFunction{
			Name: "spawn_actor",
			Value: func(args ...tengo.Object) (tengo.Object, error) {
				if len(args) < 3 {
					return tengo.UndefinedValue, fmt.Errorf("spawn_actor requires at least 3 arguments: template_id, x, y (or template_id, actor_id, x, y)")
				}

				var templateID, actorID string
				var x, y int

				if len(args) == 3 {
					tID, ok1 := tengo.ToString(args[0])
					xi, ok2 := tengo.ToInt(args[1])
					yi, ok3 := tengo.ToInt(args[2])
					if !ok1 || !ok2 || !ok3 {
						return tengo.UndefinedValue, fmt.Errorf("spawn_actor arguments must be (template string, x int, y int)")
					}
					templateID = tID
					actorID = tID
					x = xi
					y = yi
				} else {
					// 4 arguments:
					// Try Format A: (template_id string, x int, y int, actor_id string)
					if str1, ok1 := tengo.ToString(args[0]); ok1 {
						if xi, ok2 := tengo.ToInt(args[1]); ok2 {
							if yi, ok3 := tengo.ToInt(args[2]); ok3 {
								if str2, ok4 := tengo.ToString(args[3]); ok4 {
									templateID = str1
									actorID = str2
									x = xi
									y = yi
								}
							}
						}
					}

					// Try Format B: (template_id string, actor_id string, x int, y int)
					if templateID == "" {
						str1, ok1 := tengo.ToString(args[0])
						str2, ok2 := tengo.ToString(args[1])
						xi, ok3 := tengo.ToInt(args[2])
						yi, ok4 := tengo.ToInt(args[3])
						if !ok1 || !ok2 || !ok3 || !ok4 {
							return tengo.UndefinedValue, fmt.Errorf("spawn_actor arguments must be (string, string, int, int) or (string, int, int, string)")
						}

						templateID = str1
						actorID = str2
						x = xi
						y = yi

						// Allow flexible ordering if template exists under the other parameter name
						if _, ok := GetActorDef(str2); ok {
							if _, okOld := GetActorDef(str1); !okOld {
								templateID = str2
								actorID = str1
							}
						}
					}
				}

				m := GetMap()
				if m != nil {
					actorID = m.GenerateUniqueActorID(actorID)
				}

				actor, err := NewActorFromDef(actorID, templateID, x, y)
				if err != nil {
					return tengo.UndefinedValue, fmt.Errorf("failed to spawn actor %s from template %s: %w", actorID, templateID, err)
				}

				if m != nil {
					m.AddActor(actor)
				}
				return &tengo.String{Value: actorID}, nil
			},
		},
		"spawn_item": &tengo.UserFunction{
			Name: "spawn_item",
			Value: func(args ...tengo.Object) (tengo.Object, error) {
				if len(args) < 4 {
					return tengo.UndefinedValue, fmt.Errorf("spawn_item requires 4 arguments: template, x, y, entity_id")
				}
				tmpl, ok1 := tengo.ToString(args[0])
				x, ok2 := tengo.ToInt(args[1])
				y, ok3 := tengo.ToInt(args[2])
				entityID, ok4 := tengo.ToString(args[3])
				if !ok1 || !ok2 || !ok3 || !ok4 {
					return tengo.UndefinedValue, fmt.Errorf("spawn_item arguments must be (template string, x int, y int, entity_id string)")
				}

				m := GetMap()
				if m == nil {
					return tengo.UndefinedValue, fmt.Errorf("no active map")
				}

				item := NewItem(entityID, tmpl, x, y)
				m.AddItem(item)
				return tengo.UndefinedValue, nil
			},
		},
		"find_items": &tengo.UserFunction{
			Name: "find_items",
			Value: func(args ...tengo.Object) (tengo.Object, error) {
				if len(args) < 1 {
					return tengo.UndefinedValue, fmt.Errorf("find_items requires 1 argument: template")
				}
				tmpl, ok := tengo.ToString(args[0])
				if !ok {
					return tengo.UndefinedValue, fmt.Errorf("find_items argument must be a string")
				}

				m := GetMap()
				if m == nil {
					return &tengo.Array{Value: []tengo.Object{}}, nil
				}

				items := m.FindItemsByTemplate(tmpl)
				arr := make([]tengo.Object, 0, len(items))
				for _, item := range items {
					if item == nil {
						continue
					}
					itemMap := map[string]tengo.Object{
						"id":       &tengo.String{Value: item.ID},
						"name":     &tengo.String{Value: item.Name},
						"template": &tengo.String{Value: item.Template},
						"type":     &tengo.String{Value: item.Type},
						"x":        &tengo.Int{Value: int64(item.X)},
						"y":        &tengo.Int{Value: int64(item.Y)},
					}
					if item.Meta != nil {
						metaMap := make(map[string]tengo.Object, len(item.Meta))
						for k, v := range item.Meta {
							if obj, err := tengo.FromInterface(v); err == nil {
								metaMap[k] = obj
							}
						}
						itemMap["meta"] = &tengo.Map{Value: metaMap}
					}
					arr = append(arr, &tengo.Map{Value: itemMap})
				}

				return &tengo.Array{Value: arr}, nil
			},
		},
		"remove": &tengo.UserFunction{
			Name: "remove",
			Value: func(args ...tengo.Object) (tengo.Object, error) {
				if len(args) < 1 {
					return tengo.UndefinedValue, fmt.Errorf("remove requires 1 argument: entity_id")
				}
				entityID, ok := tengo.ToString(args[0])
				if !ok {
					return tengo.UndefinedValue, fmt.Errorf("remove argument must be a string")
				}

				if m := GetMap(); m != nil {
					m.RemoveEntityByID(entityID)
				}
				return tengo.UndefinedValue, nil
			},
		},
		"exec_map_script": &tengo.UserFunction{
			Name: "exec_map_script",
			Value: func(args ...tengo.Object) (tengo.Object, error) {
				if len(args) < 1 {
					return tengo.UndefinedValue, fmt.Errorf("exec_map_script requires 1 argument: script_path")
				}
				scriptName, ok := tengo.ToString(args[0])
				if !ok {
					return tengo.UndefinedValue, fmt.Errorf("exec_map_script argument must be a string")
				}
				if err := ExecuteMapScript(scriptName); err != nil {
					return tengo.UndefinedValue, err
				}
				return tengo.UndefinedValue, nil
			},
		},
		"effect_on_target": &tengo.UserFunction{
			Name: "effect_on_target",
			Value: func(args ...tengo.Object) (tengo.Object, error) {
				if len(args) < 3 {
					return tengo.UndefinedValue, fmt.Errorf("effect_on_target requires 3 arguments: effect_script, target_id, source_id")
				}
				effectScript, ok := tengo.ToString(args[0])
				if !ok {
					return tengo.UndefinedValue, fmt.Errorf("effect_on_target first argument must be a string")
				}
				targetID, ok := tengo.ToString(args[1])
				if !ok {
					return tengo.UndefinedValue, fmt.Errorf("effect_on_target second argument must be a string")
				}
				sourceID, ok := tengo.ToString(args[2])
				if !ok {
					return tengo.UndefinedValue, fmt.Errorf("effect_on_target third argument must be a string")
				}

				targetX, targetY := 0, 0
				found := false
				m := GetMap()
				if m != nil {
					if act := m.GetActorByID(targetID); act != nil {
						targetX, targetY = act.X, act.Y
						found = true
					} else if item := m.GetItemByID(targetID); item != nil {
						targetX, targetY = item.X, item.Y
						found = true
					}
				}
				if !found {
					p := GetParty()
					if p != nil {
						for i := range p.Members {
							if p.Members[i].ID == targetID {
								targetX, targetY = p.Members[i].X, p.Members[i].Y
								found = true
								break
							}
						}
					}
				}

				if !found {
					return tengo.UndefinedValue, nil
				}

				if err := ExecuteEffectScript(effectScript, targetX, targetY, targetID, sourceID); err != nil {
					return tengo.UndefinedValue, err
				}
				return tengo.UndefinedValue, nil
			},
		},
		"effect_at": &tengo.UserFunction{
			Name: "effect_at",
			Value: func(args ...tengo.Object) (tengo.Object, error) {
				if len(args) < 4 {
					return tengo.UndefinedValue, fmt.Errorf("effect_at requires 4 arguments: effect_script, x, y, source_id")
				}
				effectScript, ok := tengo.ToString(args[0])
				if !ok {
					return tengo.UndefinedValue, fmt.Errorf("effect_at first argument must be a string")
				}
				x, ok := tengo.ToInt(args[1])
				if !ok {
					return tengo.UndefinedValue, fmt.Errorf("effect_at second argument must be an integer")
				}
				y, ok := tengo.ToInt(args[2])
				if !ok {
					return tengo.UndefinedValue, fmt.Errorf("effect_at third argument must be an integer")
				}
				sourceID, ok := tengo.ToString(args[3])
				if !ok {
					return tengo.UndefinedValue, fmt.Errorf("effect_at fourth argument must be a string")
				}

				if err := ExecuteEffectScript(effectScript, x, y, "", sourceID); err != nil {
					return tengo.UndefinedValue, err
				}
				return tengo.UndefinedValue, nil
			},
		},
	}
	moduleMap.AddBuiltinModule("game", gameModule)

	// Register builtin "cut_scene" module
	cutSceneModule := map[string]tengo.Object{
		"delay": &tengo.UserFunction{
			Name: "delay",
			Value: func(args ...tengo.Object) (tengo.Object, error) {
				if len(args) < 1 {
					return tengo.UndefinedValue, fmt.Errorf("delay requires 1 argument: frames")
				}
				frames, ok := tengo.ToInt(args[0])
				if !ok {
					return tengo.UndefinedValue, fmt.Errorf("delay argument must be an integer")
				}
				EnqueueCutSceneCommand(CutSceneCommand{
					Type:   CmdDelay,
					Frames: frames,
				})
				return tengo.UndefinedValue, nil
			},
		},
		"next": &tengo.UserFunction{
			Name: "next",
			Value: func(args ...tengo.Object) (tengo.Object, error) {
				EnqueueCutSceneCommand(CutSceneCommand{
					Type:   CmdDelay,
					Frames: 1,
				})
				return tengo.UndefinedValue, nil
			},
		},
		"move": &tengo.UserFunction{
			Name: "move",
			Value: func(args ...tengo.Object) (tengo.Object, error) {
				if len(args) < 2 {
					return tengo.UndefinedValue, fmt.Errorf("move requires 2 arguments: actor_id, direction")
				}
				actorID, ok1 := tengo.ToString(args[0])
				dir, ok2 := tengo.ToString(args[1])
				if !ok1 || !ok2 {
					return tengo.UndefinedValue, fmt.Errorf("move arguments must be (string, string)")
				}
				EnqueueCutSceneCommand(CutSceneCommand{
					Type:    CmdMove,
					ActorID: actorID,
					Dir:     dir,
				})
				return tengo.UndefinedValue, nil
			},
		},
		"set_tile": &tengo.UserFunction{
			Name: "set_tile",
			Value: func(args ...tengo.Object) (tengo.Object, error) {
				if len(args) < 3 {
					return tengo.UndefinedValue, fmt.Errorf("set_tile requires 3 arguments: x, y, tile_id")
				}
				x, ok1 := tengo.ToInt(args[0])
				y, ok2 := tengo.ToInt(args[1])
				tileID, ok3 := tengo.ToInt(args[2])
				if !ok1 || !ok2 || !ok3 {
					return tengo.UndefinedValue, fmt.Errorf("set_tile arguments must be integers (x, y, tile_id)")
				}
				EnqueueCutSceneCommand(CutSceneCommand{
					Type:   CmdSetTile,
					X:      x,
					Y:      y,
					TileID: tileID,
				})
				return tengo.UndefinedValue, nil
			},
		},
		"remove_actor": &tengo.UserFunction{
			Name: "remove_actor",
			Value: func(args ...tengo.Object) (tengo.Object, error) {
				if len(args) < 1 {
					return tengo.UndefinedValue, fmt.Errorf("remove_actor requires 1 argument: actor_id")
				}
				actorID, ok := tengo.ToString(args[0])
				if !ok {
					return tengo.UndefinedValue, fmt.Errorf("remove_actor argument must be a string")
				}
				EnqueueCutSceneCommand(CutSceneCommand{
					Type:    CmdRemoveActor,
					ActorID: actorID,
				})
				return tengo.UndefinedValue, nil
			},
		},
	}
	moduleMap.AddBuiltinModule("cut-scene", cutSceneModule)

	// Register builtin "fmt" module
	fmtModule := map[string]tengo.Object{
		"print": &tengo.UserFunction{
			Name: "print",
			Value: func(args ...tengo.Object) (tengo.Object, error) {
				var sb strings.Builder
				for _, arg := range args {
					s, ok := tengo.ToString(arg)
					if !ok {
						s = arg.String()
					}
					sb.WriteString(s)
				}
				if r := GetTengoREPL(); r != nil {
					r.AddOutput(sb.String())
				}
				return tengo.UndefinedValue, nil
			},
		},
		"printf": &tengo.UserFunction{
			Name: "printf",
			Value: func(args ...tengo.Object) (tengo.Object, error) {
				if len(args) == 0 {
					return tengo.UndefinedValue, tengo.ErrWrongNumArguments
				}
				format, ok := tengo.ToString(args[0])
				if !ok {
					return tengo.UndefinedValue, tengo.ErrInvalidArgumentType{
						Name:     "format",
						Expected: "string",
						Found:    args[0].TypeName(),
					}
				}
				s, err := tengo.Format(format, args[1:]...)
				if err != nil {
					return tengo.UndefinedValue, err
				}
				if r := GetTengoREPL(); r != nil {
					r.AddOutput(s)
				}
				return tengo.UndefinedValue, nil
			},
		},
		"println": &tengo.UserFunction{
			Name: "println",
			Value: func(args ...tengo.Object) (tengo.Object, error) {
				var sb strings.Builder
				for i, arg := range args {
					if i > 0 {
						sb.WriteString(" ")
					}
					s, ok := tengo.ToString(arg)
					if !ok {
						s = arg.String()
					}
					sb.WriteString(s)
				}
				if r := GetTengoREPL(); r != nil {
					r.AddOutput(sb.String())
				}
				return tengo.UndefinedValue, nil
			},
		},
		"sprintf": &tengo.UserFunction{
			Name: "sprintf",
			Value: func(args ...tengo.Object) (tengo.Object, error) {
				if len(args) == 0 {
					return tengo.UndefinedValue, tengo.ErrWrongNumArguments
				}
				format, ok := tengo.ToString(args[0])
				if !ok {
					return tengo.UndefinedValue, tengo.ErrInvalidArgumentType{
						Name:     "format",
						Expected: "string",
						Found:    args[0].TypeName(),
					}
				}
				s, err := tengo.Format(format, args[1:]...)
				if err != nil {
					return tengo.UndefinedValue, err
				}
				return &tengo.String{Value: s}, nil
			},
		},
	}
	moduleMap.AddBuiltinModule("fmt", fmtModule)

	// Register builtin "ai" module
	aiModule := map[string]tengo.Object{
		"get_nearest_party_member": &tengo.UserFunction{
			Name: "get_nearest_party_member",
			Value: func(args ...tengo.Object) (tengo.Object, error) {
				if len(args) < 2 {
					return tengo.UndefinedValue, fmt.Errorf("get_nearest_party_member requires 2 arguments: x, y")
				}
				x, ok1 := tengo.ToInt(args[0])
				y, ok2 := tengo.ToInt(args[1])
				if !ok1 || !ok2 {
					return tengo.UndefinedValue, fmt.Errorf("get_nearest_party_member arguments must be integers")
				}

				party := GetParty()
				if party == nil || len(party.Members) == 0 {
					return &tengo.String{Value: ""}, nil
				}

				nearestID := ""
				minDistSq := -1
				for i := range party.Members {
					m := &party.Members[i]
					dx := m.X - x
					dy := m.Y - y
					distSq := dx*dx + dy*dy
					if minDistSq == -1 || distSq < minDistSq {
						minDistSq = distSq
						nearestID = m.ID
					}
				}

				return &tengo.String{Value: nearestID}, nil
			},
		},
		"step": &tengo.UserFunction{
			Name: "step",
			Value: func(args ...tengo.Object) (tengo.Object, error) {
				if len(args) < 1 {
					return tengo.UndefinedValue, fmt.Errorf("step requires 1 argument: direction")
				}
				dir, ok := tengo.ToString(args[0])
				if !ok {
					return tengo.UndefinedValue, fmt.Errorf("step argument must be a string")
				}

				actor := GetCurrentAIActor()
				if actor == nil {
					return tengo.UndefinedValue, fmt.Errorf("no active AI actor context for step")
				}

				m := GetMap()
				if m == nil {
					return tengo.UndefinedValue, fmt.Errorf("no active map")
				}

				dx, dy := 0, 0
				switch strings.ToLower(dir) {
				case "n", "north", "up":
					dy = -1
				case "s", "south", "down":
					dy = 1
				case "e", "east", "right":
					dx = 1
				case "w", "west", "left":
					dx = -1
				default:
					return tengo.UndefinedValue, fmt.Errorf("invalid direction %q (expected n, e, s, w)", dir)
				}

				moved := m.MoveActor(actor, dx, dy)
				if moved {
					return tengo.TrueValue, nil
				}
				return tengo.FalseValue, nil
			},
		},
	}
	moduleMap.AddBuiltinModule("ai", aiModule)

	// Walk data/scripts directory in embedded data.FS
	root := "scripts"
	entries, err := fs.ReadDir(data.FS, root)
	if err != nil || len(entries) == 0 {
		root = "data/scripts"
	}

	err = fs.WalkDir(data.FS, root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, ".tengo") {
			src, err := data.FS.ReadFile(path)
			if err != nil {
				return fmt.Errorf("failed to read script file %s: %w", path, err)
			}

			normPath := filepath.ToSlash(path)
			rawScriptSources[normPath] = src

			// Pre-compile script
			script := tengo.NewScript(src)
			script.SetImports(moduleMap)
			_ = script.Add("tile_x", nil)
			_ = script.Add("tile_y", nil)
			_ = script.Add("tile_idx", nil)
			_ = script.Add("keyword", "")
			_ = script.Add("reply", "")
			_ = script.Add("map_name", "")
			_ = script.Add("actor_id", "")
			_ = script.Add("target_x", nil)
			_ = script.Add("target_y", nil)
			_ = script.Add("target_id", "")
			_ = script.Add("source_id", "")
			compiled, err := script.Compile()
			if err != nil {
				return fmt.Errorf("failed to pre-compile script %s: %w", path, err)
			}

			// Store under multiple keys for flexible lookup (e.g. "data/scripts/main.tengo", "scripts/main.tengo", "main.tengo")
			compiledScripts[normPath] = compiled

			relPath, _ := filepath.Rel(root, path)
			if relPath != "" {
				compiledScripts[filepath.ToSlash(relPath)] = compiled
			}
			if !strings.HasPrefix(normPath, "data/") {
				compiledScripts["data/"+normPath] = compiled
			}

			// Register as a source module for script imports if applicable
			modName := strings.TrimSuffix(filepath.ToSlash(relPath), ".tengo")
			if modName != "" && modName != "main" {
				moduleMap.AddSourceModule(modName, src)
			}
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to load and pre-compile scripts from %s: %w", root, err)
	}

	return nil
}

// RunMainScript executes the script data/scripts/main.tengo.
func RunMainScript() error {
	return ExecuteScript("data/scripts/main.tengo")
}

// RunNewGameScript executes the script data/scripts/new_game.tengo.
func RunNewGameScript() error {
	return ExecuteScript("data/scripts/new_game.tengo")
}

// ExecuteScript executes a script by path or name (e.g. "data/scripts/main.tengo").
func ExecuteScript(scriptPath string) error {
	return ExecuteScriptWithGlobals(scriptPath, nil)
}

// ExecuteScriptWithGlobals executes a script with arbitrary global variables injected into its VM context.
func ExecuteScriptWithGlobals(scriptPath string, globals map[string]interface{}) error {
	cleanKey := filepath.ToSlash(scriptPath)
	compiled, ok := compiledScripts[cleanKey]
	if !ok {
		cleanKey = strings.TrimPrefix(cleanKey, "data/")
		compiled, ok = compiledScripts[cleanKey]
		if !ok {
			cleanKey = strings.TrimPrefix(cleanKey, "scripts/")
			compiled, ok = compiledScripts[cleanKey]
		}
	}

	if !ok || compiled == nil {
		err := fmt.Errorf("script %s not found or not pre-compiled", scriptPath)
		if defaultTerminal != nil {
			defaultTerminal.AddMessageColored(err.Error(), VGAPalette16[12])
		}
		return err
	}

	vm := compiled.Clone()
	for k, v := range globals {
		_ = vm.Set(k, v)
	}

	if err := vm.Run(); err != nil {
		execErr := fmt.Errorf("failed to execute script %s: %w", scriptPath, err)
		if defaultTerminal != nil {
			defaultTerminal.AddMessageColored(execErr.Error(), VGAPalette16[12])
		}
		return execErr
	}

	return nil
}

// ExecuteMapScript executes a map script located in data/scripts/map.
// No special globals exist for map scripts.
func ExecuteMapScript(scriptPath string) error {
	cleanPath := scriptPath
	if !strings.HasSuffix(cleanPath, ".tengo") {
		cleanPath += ".tengo"
	}
	if !strings.HasPrefix(cleanPath, "map/") && !strings.HasPrefix(cleanPath, "scripts/map/") && !strings.HasPrefix(cleanPath, "data/scripts/map/") {
		cleanPath = "map/" + cleanPath
	}
	err := ExecuteScript(cleanPath)
	if err != nil && cleanPath != scriptPath {
		if err2 := ExecuteScript(scriptPath); err2 == nil {
			return nil
		}
	}
	return err
}

// ExecuteTileScript executes a tile script with tile_x, tile_y, and tile_idx globals.
func ExecuteTileScript(scriptPath string, tileX, tileY, tileIdx int) error {
	return ExecuteScriptWithGlobals(scriptPath, map[string]interface{}{
		"tile_x":   tileX,
		"tile_y":   tileY,
		"tile_idx": tileIdx,
	})
}

// ExecuteTriggerScript executes a trigger script with map_name, tile_x, tile_y, and actor_id globals.
func ExecuteTriggerScript(scriptPath string, mapName string, tileX, tileY int, actorID string) error {
	return ExecuteScriptWithGlobals(scriptPath, map[string]interface{}{
		"map_name": mapName,
		"tile_x":   tileX,
		"tile_y":   tileY,
		"actor_id": actorID,
	})
}

// ExecuteEffectScript executes an effect script located in data/scripts/effects with target_x, target_y, target_id, and source_id globals.
func ExecuteEffectScript(scriptPath string, targetX, targetY int, targetID, sourceID string) error {
	cleanPath := scriptPath
	if !strings.HasSuffix(cleanPath, ".tengo") {
		cleanPath += ".tengo"
	}
	if !strings.HasPrefix(cleanPath, "effects/") && !strings.HasPrefix(cleanPath, "scripts/effects/") && !strings.HasPrefix(cleanPath, "data/scripts/effects/") {
		cleanPath = "effects/" + cleanPath
	}
	globals := map[string]interface{}{
		"target_x":  targetX,
		"target_y":  targetY,
		"target_id": targetID,
		"source_id": sourceID,
	}
	err := ExecuteScriptWithGlobals(cleanPath, globals)
	if err != nil && cleanPath != scriptPath {
		if err2 := ExecuteScriptWithGlobals(scriptPath, globals); err2 == nil {
			return nil
		}
	}
	return err
}

// ExecuteDialogScript executes a dialog script with the specified keyword ("look", "name", "job", "bye", etc.).
// Pre-populates 'keyword' and 'reply' globals, logs any reply string returned by the script to the terminal,
// and returns true if game.end_dialog() was called during execution.
func ExecuteDialogScript(scriptPath string, keyword string) (bool, error) {
	dialogEnded = false

	normKeyword := strings.ToLower(strings.TrimSpace(keyword))
	if len(normKeyword) > 4 {
		normKeyword = normKeyword[:4]
	}

	cleanKey := filepath.ToSlash(scriptPath)
	compiled, ok := compiledScripts[cleanKey]
	if !ok {
		cleanKey = strings.TrimPrefix(cleanKey, "data/")
		compiled, ok = compiledScripts[cleanKey]
		if !ok {
			cleanKey = strings.TrimPrefix(cleanKey, "scripts/")
			compiled, ok = compiledScripts[cleanKey]
		}
	}

	if !ok || compiled == nil {
		err := fmt.Errorf("dialog script %s not found or not pre-compiled", scriptPath)
		if defaultTerminal != nil {
			defaultTerminal.AddMessageColored(err.Error(), VGAPalette16[12])
		}
		return false, err
	}

	vm := compiled.Clone()
	_ = vm.Set("keyword", normKeyword)
	_ = vm.Set("reply", "")

	if err := vm.Run(); err != nil {
		execErr := fmt.Errorf("failed to execute dialog script %s: %w", scriptPath, err)
		if defaultTerminal != nil {
			defaultTerminal.AddMessageColored(execErr.Error(), VGAPalette16[12])
		}
		return dialogEnded, execErr
	}

	if replyObj := vm.Get("reply"); replyObj != nil {
		replyVal := replyObj.Value()
		if replyStr, ok := replyVal.(string); ok && replyStr != "" {
			if defaultTerminal != nil {
				defaultTerminal.AddMessage(replyStr)
			}
		}
	}

	return dialogEnded, nil
}
