# Script Environment

On game start, the script `main.tengo` is executed, which should setup
everything for the game loop.

## Available Modules

* `game` provides a limited interface to the global game state generally useful
  to all scripts.
  * `log(msg)` adds the string `msg` to the game terminal log.
  * `set_map_tile(x,y,tile)` sets the tile index at position x,y in the current
    map to tile.
  * `set_state(name)` creates and sets the named state variable to true.
  * `clear_state(name)` removes the named state variable.
  * `toggle_state(name)` if the named state variable exists, remove it.
    Otherwise, create it and set it to true.
  * `end_dialog()` terminates the current dialog.
  * `random(args ...)` returns one of the arguments passed at random.
