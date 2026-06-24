# item_icons

This directory stores item icon resources referenced by `item_definition.icon`.

Recommended usage:

- For a standalone icon image, set `item_definition.icon` directly to a `res://...png` path.
- For an icon inside a sprite sheet, create an `AtlasTexture` `.tres` resource here and set `item_definition.icon` to that `res://...tres` path.

Example:

- `res://resources/item_icons/red_potion_icon.tres`

The client currently loads `item_definition.icon` directly as a `Texture2D` resource.
