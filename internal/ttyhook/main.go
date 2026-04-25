package ttyhook

import (
	"github.com/nyaosorg/go-ttyadapter"
)

type TtyHook struct {
	ttyadapter.Tty
	getKey func(func() (string, error)) (string, error)
}

func (t *TtyHook) GetKey() (string, error) {
	return t.getKey(t.Tty.GetKey)
}

func New(tty ttyadapter.Tty, f func(func() (string, error)) (string, error)) *TtyHook {
	return &TtyHook{
		Tty:    tty,
		getKey: f,
	}
}
