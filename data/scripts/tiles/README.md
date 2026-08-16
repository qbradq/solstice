# Tile Scripts

Scripts in the `data/scripts/tiles` directory are tile scripts. They are
executed when the player uses a tile, either by explicitly using it or bumping
into it.

## Calling Convention

Tile scripts have the following global objects defined at runtime.

* `tile_x` is the X position of the tile in the map.
* `tile_y` is the Y position of the tile in the map.
* `tile_idx` is the tile's index number from the tile set.

Before the script is called, the globals are filled with the information about
the tile the script is running on.
