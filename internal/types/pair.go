package types

import (
	"io"
	"strings"

	"github.com/hymkor/jegan/internal/ansi"
)

type Pair struct {
	SpaceKey   []byte
	Key        string
	SpaceColon []byte
	Item
}

func (p *Pair) Clone() *Pair {
	clone := *p
	return &clone
}

func (p *Pair) LeadingSpace() []byte     { return p.SpaceKey }
func (p *Pair) SetLeadingSpace(v []byte) { p.SpaceValue = v }

func (pair *Pair) Display(w int) string {
	var b strings.Builder
	if pair.cursor {
		b.WriteString(ansi.UnderLine)
	}
	for i := 0; i < pair.nest; i++ {
		b.WriteString("  ")
	}
	highlightString(Marshal(pair.Key), ansi.Yellow, &b)
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
	if _, ok := p.Data().(SkipDump); ok {
		return
	}
	p.dumpKey(w)
	p.Item.DumpWithoutComma(w)
}

func (p *Pair) dumpKey(w io.Writer) {
	if _, ok := p.Item.data.(SkipDump); ok {
		return
	}
	w.Write(p.SpaceKey)
	w.Write(Marshal(p.Key))
	w.Write(p.SpaceColon)
	w.Write([]byte{':'})
}
