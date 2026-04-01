Jegan - A terminal JSON editor
==============================
( English / [Japanese](README_ja.md) )

Usage
-----

```

jegan some.json

```

or

```

jegan < some.json

```

Key bindings
------------

- `j`, `↓` : Move to the next item
- `k`, `↑` : Move to the previous item
- `<` : Move to the first item
- `>` : Move to the last item
- `o` : Insert a new item below the cursor.
  For object items, enter both key and value.
  For array items, enter only the value.
  The key is used as entered (no quotes required).
  The value is interpreted as follows:

  - `"..."` → string (escape sequences are interpreted)
  - Input that can be parsed as a number → number
  - `null` → null
  - `true` / `false` → boolean
  - `{}` → empty object
  - `[]` → empty array
  - Otherwise → string (used as-is)

  Note:

  - Empty input is treated as an empty string (`""`).
  - Duplicate keys in objects are not allowed.

- `r` : Modify the item at the cursor (same input method as `o`)
- `d` : Delete the item at the cursor  
  Non-empty objects and arrays cannot be deleted
- `w` : Save to file
- `q` : Quit
