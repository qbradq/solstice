# Effect Scripts

Scripts in the `data/scripts/effects` directory are effect scripts. They are
executed on-demand by the engine, for example by combat moves. They can also be
invoked by other scripts.

## Calling Convention

Effect scripts have the following global objects defined at runtime.

* `target_x` is the X position of the effect target.
* `target_y` is the Y position of the effect target.
* `target_id` is the entity ID of the effect target, if any.
* `source_id` is the entity ID of the source of the effect. Always populated.

Before the script is called, the globals are filled with the information about
the effect target.
