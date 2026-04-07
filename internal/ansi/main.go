package ansi

const (
	CursorOn  = "\x1B[?25h"
	CursorOff = "\x1B[?25l"

	EraseLine = "\x1B[0K"

	Bold        = "\x1B[1m"
	Thin        = "\x1B[22m"
	UnderLine   = "\x1B[4m"
	NoUnderLine = "\x1B[24m"
	Reverse     = "\x1B[7m"
	Inverse     = "\x1B[27m"

	Red     = "\x1B[31m"
	Yellow  = "\x1B[33m"
	Magenta = "\x1B[35m"
	Cyan    = "\x1B[36m"
	Default = "\x1B[39m"
)
