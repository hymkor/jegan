Changelog
=========
( English / [Japanese](CHANGELOG_ja.md) )

- Preserve original JSON formatting as much as possible when saving (#11, #12):
  - whitespace around tokens
  - indentation and line endings
  - object key order
  - literal representations (e.g. escape sequences)
- Highlight modified values (#14)
- Support JSONL with round-trip preservation (#16)
- Treat trailing non-JSON data as a virtual element (RawBytes). Preserve and allow editing of such data (#19)
- Support multiple input files and wildcards, and allow explicit stdin input using `-` (#18)
- Add horizontal scrolling for long lines instead of truncating them (#22)
  - `l`, Right Arrow, Ctrl-F: Scroll view to the right
  - `h`, Left Arrow, Ctrl-B: Scroll view to the left
- Add SKK-based Japanese input support using nyagosorg/go-readline-ny (#23)
- Improve JSON key input UX (#29)  
  Quotes are now rendered as virtual characters when editing object keys.
  This makes it clear that users do not need to type them explicitly,
  preventing redundant input such as "\"foo\"".

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
