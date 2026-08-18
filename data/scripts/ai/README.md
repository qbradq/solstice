# AI Scripts

Scripts in the directory `data/ai` are AI scripts. They are executed every turn
for each actor in the game. The AI scripts are separated into different
directories based on the type of AI behavior they implement.

* `idle` scripts run when an actor is otherwise unbothered.
* `combat` scripts run when an actor is in combat.

## Calling Convention

AI scripts have the following global objects defined at runtime.

* `actor_id` is the ID of the actor whose script is being executed.
* `tile_x` is the tile x coordinate of the actor.
* `tile_y` is the tile y coordinate of the actor.

