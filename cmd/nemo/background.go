package main

import (
	"github.com/nyaosorg/go-ttyadapter"

	"github.com/hymkor/jegan/internal/nonblock"
	"github.com/hymkor/jegan/internal/pager"
)

type ttyX struct {
	ttyadapter.Tty
	nonBlock *nonblock.NonBlock[any]
	work     func(any, error) bool
}

func newTtyX(
	tty ttyadapter.Tty,
	dataGetter func() (any, error),
	work func(any, error) bool) *ttyX {

	return &ttyX{
		Tty:      tty,
		nonBlock: nonblock.New(tty.GetKey, dataGetter),
		work:     work,
	}
}

func (t *ttyX) GetKey() (string, error) {
	return t.nonBlock.GetOr(t.work)
}

func (t *ttyX) Close() error {
	t.nonBlock.Close()
	return nil
}

func eventLoop(session *pager.Session,
	tty ttyadapter.Tty,
	getter func() (any, error),
	store func(any, error) bool) error {

	if err := tty.Open(nil); err != nil {
		return err
	}
	defer tty.Close()

	width, height, err := tty.Size()
	if err != nil {
		return err
	}
	session.Pager.Width = width
	session.Pager.Height = height - 1

	i := 0
	for {
		data, err := getter()
		if !store(data, err) {
			session.GetKey = tty.GetKey
			break
		}
		i++
		if i >= session.Pager.Height {
			newTtyX1 := newTtyX(tty, getter, store)
			session.GetKey = newTtyX1.GetKey
			defer newTtyX1.Close()
			break
		}
	}
	return session.EventLoop()
}
