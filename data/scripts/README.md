# Script Environment

On game start, the script `main.tengo` is executed, which should setup
everything for the game loop.

## Available Modules

* `game` provides a limited interface to the global game state generally useful
  to all scripts.
  * `log(msg)` adds the string `msg` to the game terminal log.
  * `load_map(name)` loads the named map file. For instance, if name is "home",
    the file "data/maps/home.tmx" is loaded.
  * `teleport_party(x,y)` teleports the party to location x,y on the current
    map.
  * `set_map_tile(x,y,tile)` sets the tile index at position x,y in the current
    map to tile.
  * `set_flag(name)` creates and sets the named flag to true.
  * `clear_flag(name)` removes the named flag.
  * `toggle_flag(name)` if the named flag exists, remove it.
    Otherwise, create it and set it to true.
  * `has_flag(name)` returns true if the named flag exists and is true.
  * `end_dialog()` terminates the current dialog.
  * `random(args ...)` returns one of the arguments passed at random.
