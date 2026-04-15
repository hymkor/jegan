//go:build debug

package jegan

import (
	"github.com/nyaosorg/go-windows-dbg"
)

func debug(v ...any) {
	dbg.Println(v...)
}
