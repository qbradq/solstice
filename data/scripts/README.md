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
* `game.set_string(name, value)`: Creates or sets a persistent state value named `name` with string `value`.
* `game.clear_string(name)`: Removes the persistent state value named `name`.
* `game.get_string(name)`: Returns the named persistent state string value or empty string `""` if not set.

#### Actors, Entities & Combat
* `game.add_to_party(actor_id)`: Adds the identified actor to the party as a new member, removing it from the current map and logging `"[actor_name] joins the party!"` (or `"Too many party members!"` if the party already contains 4 members). Alias: `game.add_party_member(actor_id)`.
* `game.get_party_members()`: Returns an array of map objects representing the current party members (`id`, `name`, `template`, `x`, `y`). Alias: `game.get_party()`.
* `game.teleport_actor(actor_id, x, y)`: Moves the actor with the given ID (and corresponding party member if applicable) to tile `(x, y)` on the current active map. Aliases: `game.set_actor_pos(actor_id, x, y)`, `game.move_actor_to(actor_id, x, y)`.
* `game.start_combat()`: Transitions from party mode to combat mode, setting each member's starting position to the party's location and selecting the first party member.
* `game.stop_combat()`: Transitions from combat mode back to party mode, setting the party's location to the first party member's position.
* `game.get_enemies_for_pack(name)`: Returns an array of actor template names rolled for the named pack from `data/json/packs.json`.
* `game.spawn_actor(template_id, [actor_id], x, y)`: Spawns an actor instance using the template definition from `data/json/actors.json` at tile `(x, y)` on the current active map. Automatically assigns a unique entity ID if the requested ID is already taken.
* `game.get_actor(actor_id)`: Returns a map of properties for the identified actor on the current map or in the party (`id`, `name`, `human`, `level`, `strength`, `dexterity`, `intelligence`, `max_hit_points`, `hit_points`, `max_magic_points`, `magic_points`, `range`, `damage`), or `undefined` if not found.
* `game.damage_actor(actor_id, amount)`: Deducts `amount` hit points from the identified actor. If the actor's hit points drop to 0 or lower, replaces the actor on the map with a `"human_corpse"` or `"animal_corpse"` item based on the actor's `human` property.
* `game.remove(entity_id)`: Removes the entity (actor or item) with the given ID from the current active map.
* `game.exec_map_script(script_path)`: Executes a map script located in `data/scripts/map/`.
* `game.effect_on_target(effect_script, target_id, source_id)`: Runs the given effect script located in `data/scripts/effects/` on the entity with `target_id` (if it exists on the map or in the party), injecting `target_id`, `target_x`, `target_y`, and `source_id` globals.
* `game.effect_at(effect_script, x, y, source_id)`: Runs the given effect script located in `data/scripts/effects/` at tile position `(x, y)` without a target entity (`target_id = ""`), injecting `target_x`, `target_y`, and `source_id` globals.

#### Dialog & Interaction
* `game.start_dialog([actor_id], dialog_script)`: Initiates a dialog interaction using the specified script (aliases: `enter_dialog`, `force_dialog`).
* `game.end_dialog()`: Terminates the active dialog session.

#### Logging, Timers & Utilities
* `game.log(format, [args...])`: Formats and appends a message to the in-game terminal log (matching the signature of `fmt.printf`).
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
| **Effect Scripts** | `data/scripts/effects/` | `target_x`, `target_y`, `target_id`, `source_id` | Executed on-demand by combat moves or script invocations |
| **Map Scripts** | `data/scripts/map/` | Context-dependent | Map setup and cutscene orchestration scripts |
| **REPL Autoexec** | `data/scripts/repl/autoexec.tengo` | None | Run on REPL initialization/reset, pre-loading modules |


