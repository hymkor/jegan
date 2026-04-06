Changelog
=========
( English / [Japanese](CHANGELOG_ja.md) )

- Preserve original JSON formatting as much as possible when saving (#11, #12):
  - whitespace around tokens
  - indentation and line endings
  - object key order
  - literal representations (e.g. escape sequences)
- Highlight modified values (#14)

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
