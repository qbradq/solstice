package solstice

import (
	"fmt"
	"io/fs"
	"math/rand"
	"path/filepath"
	"strings"
	"sync"

	"solstice/data"

	"github.com/d5/tengo/v2"
)

var (
	scriptMu         sync.RWMutex
	stateMu          sync.RWMutex
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

// SetState creates and sets the named state variable to true.
func SetState(name string) {
	stateMu.Lock()
	defer stateMu.Unlock()
	gameState[name] = true
}

// ClearState removes the named state variable.
func ClearState(name string) {
	stateMu.Lock()
	defer stateMu.Unlock()
	delete(gameState, name)
}

// ToggleState removes the named state variable if it exists; otherwise creates it and sets it to true.
func ToggleState(name string) {
	stateMu.Lock()
	defer stateMu.Unlock()
	if gameState[name] {
		delete(gameState, name)
	} else {
		gameState[name] = true
	}
}

// HasState returns true if the named state variable exists and is true.
func HasState(name string) bool {
	stateMu.RLock()
	defer stateMu.RUnlock()
	return gameState[name]
}

// ClearAllState resets all state variables.
func ClearAllState() {
	stateMu.Lock()
	defer stateMu.Unlock()
	gameState = make(map[string]bool)
}

// InitScriptSystem initializes the Tengo scripting system by recursively loading and pre-compiling
// all .tengo files in the data/scripts directory from data.FS.
func InitScriptSystem() error {
	scriptMu.Lock()
	defer scriptMu.Unlock()

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
		"set_state": &tengo.UserFunction{
			Name: "set_state",
			Value: func(args ...tengo.Object) (tengo.Object, error) {
				if len(args) < 1 {
					return tengo.UndefinedValue, fmt.Errorf("set_state requires 1 argument: name")
				}
				name, ok := tengo.ToString(args[0])
				if !ok {
					return tengo.UndefinedValue, fmt.Errorf("set_state argument must be a string")
				}
				SetState(name)
				return tengo.UndefinedValue, nil
			},
		},
		"clear_state": &tengo.UserFunction{
			Name: "clear_state",
			Value: func(args ...tengo.Object) (tengo.Object, error) {
				if len(args) < 1 {
					return tengo.UndefinedValue, fmt.Errorf("clear_state requires 1 argument: name")
				}
				name, ok := tengo.ToString(args[0])
				if !ok {
					return tengo.UndefinedValue, fmt.Errorf("clear_state argument must be a string")
				}
				ClearState(name)
				return tengo.UndefinedValue, nil
			},
		},
		"toggle_state": &tengo.UserFunction{
			Name: "toggle_state",
			Value: func(args ...tengo.Object) (tengo.Object, error) {
				if len(args) < 1 {
					return tengo.UndefinedValue, fmt.Errorf("toggle_state requires 1 argument: name")
				}
				name, ok := tengo.ToString(args[0])
				if !ok {
					return tengo.UndefinedValue, fmt.Errorf("toggle_state argument must be a string")
				}
				ToggleState(name)
				return tengo.UndefinedValue, nil
			},
		},
		"has_state": &tengo.UserFunction{
			Name: "has_state",
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
	}
	moduleMap.AddBuiltinModule("game", gameModule)

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

// ExecuteScript executes a script by path or name (e.g. "data/scripts/main.tengo").
func ExecuteScript(scriptPath string) error {
	return ExecuteScriptWithGlobals(scriptPath, nil)
}

// ExecuteScriptWithGlobals executes a script with arbitrary global variables injected into its VM context.
func ExecuteScriptWithGlobals(scriptPath string, globals map[string]interface{}) error {
	scriptMu.RLock()
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
	scriptMu.RUnlock()

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

// ExecuteTileScript executes a tile script with tile_x, tile_y, and tile_idx globals.
func ExecuteTileScript(scriptPath string, tileX, tileY, tileIdx int) error {
	return ExecuteScriptWithGlobals(scriptPath, map[string]interface{}{
		"tile_x":   tileX,
		"tile_y":   tileY,
		"tile_idx": tileIdx,
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

	scriptMu.RLock()
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
	scriptMu.RUnlock()

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
