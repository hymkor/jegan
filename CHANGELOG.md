Changelog
=========
( English / [Japanese](CHANGELOG_ja.md) )

- Improve search: now matches both keys and values, case-insensitive (#68)
- Undo support for replace operations (`r`, `R`) (#69)
- Undo support for deletions (#69)
  - Deleted elements are now marked as `<DEL>` instead of being removed immediately
  - Press `u` to restore deleted elements
  - `<DEL>` entries are omitted when saving, so deletions are finalized on disk

v0.4.0
------
Apr 17, 2026

- Display current JSON path and value in status line (#62, #63)
- Add shortcut (Ctrl-C) to copy current path and value (#64)
- Value search feature (#65)
  - `/` : search forward
  - `?` : search backward
  - `n` : repeat search in the same direction
  - `N` : repeat search in the opposite direction
  - Search applies to values only (keys are ignored)
  - Search input follows the same rules as the `o` command:
    - `"text"` → string
    - `123`, `1.23` → number
    - `true`, `false`, `nil` → literals
    - others → treated as a string
- Add Key bindings to reset horizontal scroll and return to column 0: `0`, `^` (#66)

v0.3.1
------
Apr 14, 2026

- Preserve colon spacing when adding object pairs with "o" (#57)
- Improve rendering of non-JSON text by normalizing whitespace and preserving visible characters instead of escaped string representation (Thanks to @rinodrops, #51, #58)

v0.3.0
------
Apr 14, 2026

- Support JSON wrapped in JavaScript assignments
  (as used in X/Twitter archives) (#44)
- Show a simple progress animation during save (#45)
- Add `-auto` option to simulate key and readline inputs for testing (#46)
- Add page up/down navigation (Space / b, PageUp / PageDown) (#49)
- Fix string rendering to preserve original representation (avoid HTML escaping) (#50)
- Fix issue where newline before closing brace and comma placement were incorrect in saved output when adding a pair with "o" (Thanks to @rinodrops, #51, #54)

v0.2.1
------
Apr 12, 2026

### Bug Fixes

- Fix parsing issue where empty arrays or objects containing whitespace caused incorrect nesting (#40)
- Fix error propagation so messages are not lost (#41)
- Fix bug where `\"` was incorrectly recognized as an escaped quote when preceded by an even number of backslashes (#42)

v0.2.0
------
Apr 12, 2026

### Data Integrity

- Preserve original JSON formatting as much as possible when saving (#11, #12):
  - whitespace around tokens
  - indentation and line endings
  - object key order
  - literal representations (e.g. escape sequences)
  - trailing non-JSON data is preserved as a virtual element (RawBytes) (#19)
  - separators between top-level elements in JSONL (#16)

- Highlight modified values (#14)

### Usability Improvements

- Add horizontal scrolling for long lines (#22):
  - `l`, Right Arrow, Ctrl-F: scroll right
  - `h`, Left Arrow, Ctrl-B: scroll left

- Normalize save/quit prompt behavior:
  - Treat non-Yes/No input as cancel (#32)

### Readline Improvements

- Add SKK-based Japanese input support using nyagosorg/go-readline-ny (#23)

- Improve JSON key input UX (#29):
  Quotes are now rendered automatically when editing object keys,
  so users do not need to type them explicitly.

### CLI and I/O Improvements

- Support multiple input files and wildcards, and allow explicit stdin input using `-` (#18)

- Support I/O redirection (#31):
  - Render UI to stderr when stdin is redirected
  - Output to stdout when `-` or an empty filename is specified

### Bug Fixes

- Fix modified mark (`*`) being appended to unrelated messages (#32)
- Change Ctrl-L behavior during input (e.g. cell or filename editing) (#37):
  - Redraw the current input instead of clearing the entire screen.
  - Update go-readline-ny to v1.14.3 supporting redrawing the current input.

v0.1.0
------
Apr 5, 2026

- Preserve object key order when loading and saving JSON (#2)
- Preserve indentation width and line endings when saving JSON (best-effort) (#3)
- Improve JSON parser error messages with line and column information (#4)
- Use alphabet keys instead of numeric keys for type selection in Shift+R (#5)
- Place the cursor before the file extension when editing the save filename (#6)
- Preserve trailing newline at EOF when saving JSON (#7)
- Output empty objects and arrays as `{}` and `[]` without extra newlines (#8)
- Avoid marking the document as modified when a value is updated without actual changes (#9)

v0.0.1
------
Apr 4, 2026

- Initial version
