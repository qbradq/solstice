# Script Environment

Solstice uses the [Tengo](https://github.com/d5/tengo) scripting language for game logic, dialogs, map triggers, cutscenes, timers, and interactive debugging in the pull-down console REPL.

When a new game begins, the script `new_game.tengo` is executed to initialize the party, starting location, and map.

---

## Available Modules

### `game` Module
The `game` module provides access to the game world, party, flags, maps, dialogs, timers, and logging.

```golang
game := import("game")
```

#### Map & World Navigation
* `game.load_map(name)`: Loads the named map file (e.g. `"home"` loads `data/maps/home.tmx`), caching it in memory and setting it as the current active map.
* `game.reload_map(name)`: Reloads the named map from the embedded filesystem if present in the loaded maps cache, updating current map and world map pointers if applicable.
* `game.teleport_party(x, y)`: Teleports the party to tile coordinates `(x, y)` on the current active map.
* `game.teleport_party_on_world_map(x, y)`: Sets the party's location on the world map to tile coordinates `(x, y)`.
* `game.set_map_tile(x, y, tile_idx)`: Sets the tile index at position `(x, y)` on the current active map.

#### Game Flags & State
* `game.set_flag(name)`: Creates and sets the named flag to `true`.
* `game.clear_flag(name)`: Removes the named flag.
* `game.toggle_flag(name)`: Toggles the named flag (removes if `true`, sets `true` if absent/`false`).
* `game.has_flag(name)`: Returns `true` if the named flag exists and is `true`, `false` otherwise.

#### Actors, Entities & Combat
* `game.add_to_party(actor_id)`: Adds the identified actor to the party as a new member, removing it from the current map and logging `"[actor_name] joins the party!"` (or `"Too many party members!"` if the party already contains 4 members).
* `game.start_combat()`: Transitions from party mode to combat mode, setting each member's starting position to the party's location and selecting the first party member.
* `game.stop_combat()`: Transitions from combat mode back to party mode, setting the party's location to the first party member's position.
* `game.spawn_actor(template_id, actor_id, x, y)`: Spawns an actor instance using the template definition from `data/json/actors.json` at tile `(x, y)` on the current active map.
* `game.spawn_item(template, x, y, entity_id)`: Spawns an item instance using the template definition from `data/json/items.json` at tile `(x, y)` on the current active map.
* `game.find_items(template)`: Returns an array of item map objects matching the named template on the current active map.
* `game.remove(entity_id)`: Removes the entity (actor or item) with the given ID from the current active map.
* `game.exec_map_script(script_path)`: Executes a map script located in `data/scripts/map/`.

#### Dialog & Interaction
* `game.start_dialog([actor_id], dialog_script)`: Initiates a dialog interaction using the specified script (aliases: `enter_dialog`, `force_dialog`).
* `game.end_dialog()`: Terminates the active dialog session.

#### Logging, Timers & Utilities
* `game.log(msg)`: Appends the string `msg` to the in-game terminal log.
* `game.roll(expression)`: Evaluates a dice roll expression string (e.g. `"1d4"`, `"3d6+2"`) and returns the integer result.
* `game.add_timer(delay_turns, script_name, [globals])`: Schedules a map timer that executes `script_name` after `delay_turns` turns on the current map with an optional map of injected global variables.
* `game.random(args...)`: Returns one of the provided arguments at random.

---

### `ai` Module
The `ai` module provides movement and targeting utilities for actor AI scripts.

```golang
ai := import("ai")
```

* `ai.get_nearest_party_member(x, y)`: Returns the ID string of the party member actor closest to coordinates `(x, y)` on the current map.
* `ai.step(direction)`: Moves the executing AI actor by 1 tile in the specified direction (`"n"`, `"e"`, `"s"`, or `"w"`), respecting map boundaries, tile walkability, and actor collisions. Returns `true` if moved, `false` if blocked.

---

### `cut-scene` Module
The `cut-scene` module queues sequential, frame-timed cutscene actions that advance in lock-step with game animation frames.

```golang
cs := import("cut-scene")
```

* `cs.delay(frames)`: Queues a pause of the specified number of animation frames.
* `cs.next()`: Queues a 1-frame pause (shorthand for `cs.delay(1)`).
* `cs.move(actor_id, direction)`: Queues a 1-tile movement step for `actor_id` in direction `"n"`, `"s"`, `"w"`, or `"e"`.
* `cs.set_tile(x, y, tile_id)`: Queues setting tile `(x, y)` on the current map to `tile_id`.
* `cs.remove_actor(actor_id)`: Queues removing `actor_id` from the current map.

---

### `fmt` Module
The `fmt` module provides formatted text output mirroring Tengo's standard library `fmt` module, routing printed output into the pull-down Tengo REPL terminal in bright white.

```golang
fmt := import("fmt")
```

* `fmt.print(args...)`: Formats arguments and appends word-wrapped output to the Tengo terminal without spaces between operands.
* `fmt.printf(format, args...)`: Formats according to format specifiers (e.g. `%s`, `%d`, `%v`) and appends word-wrapped output to the Tengo terminal in bright white.
* `fmt.println(args...)`: Formats arguments separated by spaces and appends word-wrapped output to the Tengo terminal in bright white.
* `fmt.sprintf(format, args...)`: Formats according to format specifiers and returns the resulting string without printing to the terminal.

---

## Script Contexts & Injected Globals

| Context | Location | Injected Globals | Description |
| :--- | :--- | :--- | :--- |
| **New Game** | `data/scripts/new_game.tengo` | None | Run on game start / new game initialization |
| **Tile Scripts** | `data/scripts/tile/` | `tile_x`, `tile_y`, `tile_idx` | Triggered when interacting with a tile having `use_script` |
| **Trigger Scripts** | `data/scripts/triggers/` | `map_name`, `tile_x`, `tile_y`, `actor_id` | Triggered on step or enter into a map trigger zone |
| **Dialog Scripts** | `data/scripts/dialog/` | `keyword` (input query), `reply` (output text) | Invoked during NPC conversation keyword responses |
| **AI Scripts** | `data/scripts/ai/` | `actor_id`, `tile_x`, `tile_y` | Executed each turn per actor (idle scripts during normal play, combat scripts in combat) |
| **Map Scripts** | `data/scripts/map/` | Context-dependent | Map setup and cutscene orchestration scripts |
| **REPL Autoexec** | `data/scripts/repl/autoexec.tengo` | None | Run on REPL initialization/reset, pre-loading modules |

