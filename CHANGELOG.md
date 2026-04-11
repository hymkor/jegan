Changelog
=========
( English / [Japanese](CHANGELOG_ja.md) )

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
