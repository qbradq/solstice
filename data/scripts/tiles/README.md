# Calling Conventions for Tile Scripts

Scripts in the `data/scripts/tiles` directory have the following global objects
defined.

* `tile` contains all of the information about the tile the script is being
  executed on.
  * `x` is the X position of the tile in the map.
  * `y` is the Y position of the tile in the map.
  * `tile` is the tile number from the tile set.

Before the script is called, the `tile` global is filled with the information
about the tile the script is running on.
