Jegan - A terminal JSON editor
==============================
<!-- go run github.com/hymkor/example-into-readme/cmd/badges@latest | -->
[![Go Test](https://github.com/hymkor/jegan/actions/workflows/go.yml/badge.svg)](https://github.com/hymkor/jegan/actions/workflows/go.yml)
[![License](https://img.shields.io/badge/License-MIT-red)](https://github.com/hymkor/jegan/blob/master/LICENSE)
[![Go Reference](https://pkg.go.dev/badge/github.com/hymkor/jegan.svg)](https://pkg.go.dev/github.com/hymkor/jegan)
[![GitHub](https://img.shields.io/badge/github-repo-blue?logo=github)](https://github.com/hymkor/jegan)
<!-- -->
( English / [Japanese](README_ja.md) )

![](./demo.gif)

Features
--------

### &#x1F6E1;&#xFE0F; Preserve original JSON with minimal changes

Jegan keeps your JSON as close to the original as possible.
Only the parts you edit are modified.

* Key order is preserved
* Whitespace, indentation, and line endings are preserved
* String representations (e.g. `\uXXXX` vs raw UTF-8) are preserved
* Even non-JSON parts (e.g. JavaScript wrappers) are kept intact  
&nbsp;
* Only modified fields are updated
* Changes are highlighted in bold
* Original style is reused when inserting new values
* Backup file is created on save

### &#x26A1; Instant startup even with large JSON files

Jegan starts immediately, even when loading large JSON files.

Instead of blocking on file loading, data is processed in the background and becomes available progressively.

* The UI appears instantly
* You can start navigating before the entire file is loaded
* No more waiting for the full file to load before interacting

### &#x1F504; Undo support for safe editing

* Press `u` to undo recent changes
* Replace operations (`r`, `R`, etc.) can be undone
* Deletions can also be undone

Deleted entries are not removed immediately but marked as `<DEL>`:

* `<DEL>` entries can be restored with `u`
* `<DEL>` entries are omitted when saving, so deletions are finalized on disk

### &#x1F9E9; Collapsible view for containers

* Press `z` to toggle collapse/expand for objects and arrays
* Collapsed elements are displayed as a single line

### &#x1F4E6; Supports real-world JSON formats

* JSON
* JSON Lines (JSONL)
* JavaScript-style assignments (e.g. X/Twitter archives)

### &#x1F9ED; Structured navigation in terminal

* Navigate items with `j` / `k`
* Horizontal scrolling for long lines
* JSON path and current value shown in status line
* Search with `/`, `?`, `n`, `N`

### &#x1F50C; CLI-friendly

* Read from file or stdin
* Write to file or stdout
* Works as a filter:

```
jegan < input.json > output.json
```

### &#x2328;&#xFE0F; Efficient editing

* vi-like navigation
* Emacs-style input for editing values

Install
-------

### Manual Installation

Download the binary package from [Releases](https://github.com/hymkor/jegan/releases) and extract the executable.

> &#9888;&#65039; Note: The macOS build is experimental and not yet tested.
> Please let us know if you encounter any issues!

<!-- go run github.com/hymkor/example-into-readme/cmd/how2install@latest | -->

### Use [eget] installer (cross-platform)

```sh
brew install eget        # Unix-like systems
# or
scoop install eget       # Windows

cd (YOUR-BIN-DIRECTORY)
eget hymkor/jegan
```

[eget]: https://github.com/zyedidia/eget

### Use [scoop]-installer (Windows only)

```
scoop install https://raw.githubusercontent.com/hymkor/jegan/master/jegan.json
```

or

```
scoop bucket add hymkor https://github.com/hymkor/scoop-bucket
scoop install jegan
```

[scoop]: https://scoop.sh/

### Use "go install" (requires Go toolchain)

```
go install github.com/hymkor/jegan/cmd/jegan@latest
```

Note: `go install` places the executable in `$HOME/go/bin` or `$GOPATH/bin`, so you need to add this directory to your `$PATH` to run `jegan`.
<!-- -->

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

- `F1` : Show help screen (press `q` to close)
- `j`, `↓`, `Ctrl-N` : Move to the next item
- `k`, `↑`, `Ctrl-P` : Move to the previous item
- `l`, `→`, `Ctrl-F` : Scroll the view to the right
- `h`, `←`, `Ctrl-B` : Scroll the view to the left
- `0`, `^` : Reset horizontal scroll (jump to column 0)
- `Space`, `PageDown` : Move to the next page of items
- `b`, `PageUp`       : Move to the previous page of items
- `<` : Move to the first item
- `>` : Move to the last item
- `/` : Search forward
- `?` : Search backward
- `n` : Repeat search in the same direction
- `N` : Repeat search in the opposite direction
- '@' : Jump to the item specified by a JSON path
- `z` : Toggle collapse/expand
- `o` : Insert a new item below the cursor.
  - For object items, enter both key and value.
  - For array items, enter only the value.
  - The key is used as entered (no quotes required).
  - The value is interpreted as follows:
    - `"..."` → string (escape sequences are interpreted)
    - *Input that can be parsed as a number* → number
    - `null` → null
    - `true` / `false` → boolean
    - `{}` → empty object
    - `[]` → empty array
    - *Otherwise* → string (used as-is)
  - `Ctrl+G` cancels the current input
  - Empty input is treated as an empty string (`""`).
  - Duplicate keys in objects are not allowed.
- `r` : Modify the item at the cursor (same input method as `o`)
- `R` : Modify the item at the cursor (explicitly specify the value type)
- `d` : Delete the item at the cursor
- `u` : UNDO
- `Ctrl+C` : Copy the current path and value to the clipboard
- `w` : Save to file
- `q` : Quit

Changelog
---------

- [English](CHANGELOG.md)
- [Japanese](CHANGELOG_ja.md)

Acknowledgements
----------------

- [rinodrops (Rino)](https://github.com/rinodrops)

Author
------

- [HAYAMA Kaoru](https://github.com/hymkor)
