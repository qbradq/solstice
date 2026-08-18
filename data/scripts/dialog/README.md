# Dialog Scripts

Scripts in the directory `data/scripts/dialog` are dialog scripts. They are
executed when the player talks to an actor.

## Calling Convention

Dialog scripts have the following global objects defined at runtime.

* `keyword` the lower-case, normalized keyword. This is only the first four
  letters of the word. For instance, if the player says, "pendant", the keyword
  variable will contain the string "pend".
* `reply` is empty on entry and its contents are logged to the game's terminal
  to facilitate a two-way dialog.
