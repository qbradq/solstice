# Solstice

Solstice is an Ebitengine-based application written in Go.

## Building and Running

### Run directly
```bash
go run ./cmd/solstice
```

### Build binary
```bash
go build ./cmd/solstice
./solstice
```

## License

This project is licensed under the MIT License. See [LICENSE](LICENSE) for the
full text of the license.

## Game Design Details

### Actor Properties

* Experience counts the total experience gained by the character. It ranges from
  0 to 65000.
* Level records the current character level.
  * When a character reaches certain breakpoints of experience their level
    changes. The breakpoints are below.
    * Level 1 - 0
    * Level 2 - 2000
    * Level 3 - 5000
    * Level 4 - 1000
    * Level 5 - 16000
    * Level 6 - 27000
    * Level 7 - 42000
    * Level 8 - 65000
  * When the party kills a character, each member of the party is awarded 100
    experience points per level of the character killed.
* Strength describes the physical strength of the actor. Melee weapons and armor
  have strength requirements to equip. High strength will increase damage done
  by melee weapons. Low strength will diminish it.
* Dexterity describes the agility and deftness of movement of the actor. Ranged
  weapons like bows and boomerangs have dexterity requirements to equip. High
  dexterity will reduce the chances of being hit by melee and ranged attacks.
  Low dexterity will increase the chances of being hit by melee and ranged
  attacks.
* Intelligence describes the cunning and mental fortitude of the player. Each
  spell circle has an intelligence requirement to cast. High intelligence
  increases the effectiveness of all spells and magical items. Low intelligence
  decreases their effectiveness.
* Hit Points track how much damage a character can withstand before becoming
  incapacitated. Characters that remain at or below 0 hit points for too long
  will die, but may be revived by magical means. Hit points can be restored by a
  number of magical and mundane means and also regenerates slowly as turns pass.
  * Characters are given between 15 and 30 max hit points per level creating a
    total range of 15 to 240.
  * How many max hit points are given per level depend on the character's
    strength property, with 3 and below giving 15 and 18 and above giving 30.
* Magic Points track how much magical energy a character has. All spells cost
  magic points to cast. Magic points may not go below 0. Magic points regenerate
  faster than hit points.
  * Not all characters will gain max magic points on level up.
  * When a character gains max magic points on level up, they gain between 5 and
    15 points per level creating a total range of 0 to 120.
  * How many max magic points are given per level depend on the character's
    intelligence property.
    * Nothing is given below 8.
    * For values 8 through 18, 5 to 15 max magic points are given.
    * For values above 18, 15 are given.
* Strength, Dexterity, and Intelligence all have typical ranges between 6 and 18
  but may be as low as 3 and as high as 24 in extreme cases.
