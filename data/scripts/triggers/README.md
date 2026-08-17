# Trigger Scripts

Scripts in the `data/scripts/triggers/` directory are trigger scripts. They are
executed when an actor or the party activates a trigger area.

# Calling Convention

Trigger scripts have the following globals defined at runtime.

* `map_name` is the name of the map.
* `tile_x` is the x coordinate of the tile the actor or party is standing on.
* `tile_y` is the y coordinate of the tile the actor or party is standing on.
* `actor_id` is the actor's id. This will be "party" if the party is triggering
  the trigger.

Before the script is run, all globals are updated.
