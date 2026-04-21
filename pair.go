package jegan

import (
	"io"
	"strings"

	"github.com/hymkor/jegan/internal/ansi"
)

type Pair struct {
	spaceKey   []byte
	key        string
	spaceColon []byte
	Item
}

func (p *Pair) LeadingSpace() []byte     { return p.spaceKey }
func (p *Pair) SetLeadingSpace(v []byte) { p.spaceValue = v }

func (pair *Pair) Display(w int) string {
	var b strings.Builder
	if pair.cursor {
		b.WriteString(ansi.UnderLine)
	}
	for i := 0; i < pair.nest; i++ {
		b.WriteString("  ")
	}
	highlightString(marshal(pair.key), ansi.Yellow, &b)
	b.WriteString(": ")
	pair.Item.highlight(&b)
	if pair.cursor {
		b.WriteString(strings.Repeat(" ", w))
		b.WriteString(ansi.NoUnderLine)
	}
	return b.String()
}

func (p *Pair) Dump(w io.Writer) {
	p.dumpKey(w)
	p.Item.Dump(w)
}

func (p *Pair) DumpWithoutComma(w io.Writer) {
	if _, ok := p.Data().(*tombstone); ok {
		return
	}
	p.dumpKey(w)
	p.Item.DumpWithoutComma(w)
}

func (p *Pair) dumpKey(w io.Writer) {
	if _, ok := p.Item.data.(*tombstone); ok {
		return
	}
	w.Write(p.spaceKey)
	w.Write(marshal(p.key))
	w.Write(p.spaceColon)
	w.Write([]byte{':'})
}
