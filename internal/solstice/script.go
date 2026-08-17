package solstice

import (
	"fmt"
	"io/fs"
	"math/rand"
	"path/filepath"
	"strings"

	"solstice/data"

	"github.com/d5/tengo/v2"
)

var (
	moduleMap        *tengo.ModuleMap
	compiledScripts  = make(map[string]*tengo.Compiled)
	rawScriptSources = make(map[string][]byte)
	gameState        = make(map[string]bool)
	dialogEnded      bool
)

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

// Backward-compatible aliases for state functions
func SetState(name string)       { SetFlag(name) }
func ClearState(name string)     { ClearFlag(name) }
func ToggleState(name string)    { ToggleFlag(name) }
func HasState(name string) bool  { return HasFlag(name) }
func ClearAllState()             { ClearAllFlags() }

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
				msg, ok := tengo.ToString(args[0])
				if !ok {
					msg = args[0].String()
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
					return args[0], nil
				}
				idx := rand.Intn(len(args))
				return args[idx], nil
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
						ID:           actorID,
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
						ID:           actorID,
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
						ID:           actorID,
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
				if len(args) < 4 {
					return tengo.UndefinedValue, fmt.Errorf("spawn_actor requires 4 arguments: template_id, actor_id, x, y")
				}
				str1, ok1 := tengo.ToString(args[0])
				str2, ok2 := tengo.ToString(args[1])
				x, ok3 := tengo.ToInt(args[2])
				y, ok4 := tengo.ToInt(args[3])
				if !ok1 || !ok2 || !ok3 || !ok4 {
					return tengo.UndefinedValue, fmt.Errorf("spawn_actor arguments must be (string, string, int, int)")
				}

				templateID := str1
				actorID := str2

				// Allow flexible ordering if template exists under the other parameter name
				if _, ok := GetActorDef(str2); ok {
					if _, okOld := GetActorDef(str1); !okOld {
						templateID = str2
						actorID = str1
					}
				}

				actor, err := NewActorFromDef(actorID, templateID, x, y)
				if err != nil {
					return tengo.UndefinedValue, fmt.Errorf("failed to spawn actor %s from template %s: %w", actorID, templateID, err)
				}

				if m := GetMap(); m != nil {
					m.AddActor(actor)
				}
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

				if m := GetMap(); m != nil {
					m.RemoveActorByID(actorID)
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
		"run_map_script": &tengo.UserFunction{
			Name: "run_map_script",
			Value: func(args ...tengo.Object) (tengo.Object, error) {
				if len(args) < 1 {
					return tengo.UndefinedValue, fmt.Errorf("run_map_script requires 1 argument: script_path")
				}
				scriptName, ok := tengo.ToString(args[0])
				if !ok {
					return tengo.UndefinedValue, fmt.Errorf("run_map_script argument must be a string")
				}
				if err := ExecuteMapScript(scriptName); err != nil {
					return tengo.UndefinedValue, err
				}
				return tengo.UndefinedValue, nil
			},
		},
		"execute_map_script": &tengo.UserFunction{
			Name: "execute_map_script",
			Value: func(args ...tengo.Object) (tengo.Object, error) {
				if len(args) < 1 {
					return tengo.UndefinedValue, fmt.Errorf("execute_map_script requires 1 argument: script_path")
				}
				scriptName, ok := tengo.ToString(args[0])
				if !ok {
					return tengo.UndefinedValue, fmt.Errorf("execute_map_script argument must be a string")
				}
				if err := ExecuteMapScript(scriptName); err != nil {
					return tengo.UndefinedValue, err
				}
				return tengo.UndefinedValue, nil
			},
		},
		"map_script": &tengo.UserFunction{
			Name: "map_script",
			Value: func(args ...tengo.Object) (tengo.Object, error) {
				if len(args) < 1 {
					return tengo.UndefinedValue, fmt.Errorf("map_script requires 1 argument: script_path")
				}
				scriptName, ok := tengo.ToString(args[0])
				if !ok {
					return tengo.UndefinedValue, fmt.Errorf("map_script argument must be a string")
				}
				if err := ExecuteMapScript(scriptName); err != nil {
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
		return fmt.Errorf("script %s not found or not pre-compiled", scriptPath)
	}

	vm := compiled.Clone()
	for k, v := range globals {
		_ = vm.Set(k, v)
	}

	if err := vm.Run(); err != nil {
		return fmt.Errorf("failed to execute script %s: %w", scriptPath, err)
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
		return false, fmt.Errorf("dialog script %s not found or not pre-compiled", scriptPath)
	}

	vm := compiled.Clone()
	_ = vm.Set("keyword", normKeyword)
	_ = vm.Set("reply", "")

	if err := vm.Run(); err != nil {
		return dialogEnded, fmt.Errorf("failed to execute dialog script %s: %w", scriptPath, err)
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
