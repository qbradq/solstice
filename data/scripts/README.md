# Script Environment

On game start, the script `main.tengo` is executed, which should setup
everything for the game loop.

## Available Modules

* `game` provides a limited interface to the global game state generally useful
  to all scripts.
  * `log(msg)` adds the string `msg` to the game terminal log.
  * `set_map_tile(x,y,tile)` sets the tile index at position x,y in the current
    map to tile.
