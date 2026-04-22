package ttywrap

import (
	"github.com/nyaosorg/go-ttyadapter"
)

type ttyWrap struct {
	ttyadapter.Tty
	wrapper func() (string, error)
}

func (t *ttyWrap) GetKey() (string, error) {
	return t.wrapper()
}

func New(tty ttyadapter.Tty, f func() (string, error)) *ttyWrap {
	return &ttyWrap{
		Tty:     tty,
		wrapper: f,
	}
}
