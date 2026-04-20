package asyncpager

import (
	"io"
	"time"

	"github.com/nyaosorg/go-ttyadapter"

	"github.com/hymkor/go-generics-list"

	"github.com/hymkor/jegan/internal/nonblock"
	"github.com/hymkor/jegan/internal/pager"
)

type Displayer = pager.Displayer

type ttyX[T Displayer] struct {
	ttyadapter.Tty
	nonBlock *nonblock.NonBlock[T]
	work     func(T, error) bool
}

func newTtyX[T Displayer](
	tty ttyadapter.Tty,
	dataGetter func() (T, error),
	work func(T, error) bool) *ttyX[T] {

	return &ttyX[T]{
		Tty:      tty,
		nonBlock: nonblock.New[T](tty.GetKey, dataGetter),
		work:     work,
	}
}

func (t *ttyX[T]) GetKey() (string, error) {
	return t.nonBlock.GetOr(t.work)
}

func (t *ttyX[T]) Close() error {
	t.nonBlock.Close()
	return nil
}

type Pager[T pager.Displayer] pager.Pager[T]

func (pg *Pager[T]) EventLoop(
	tty ttyadapter.Tty,
	getter func() (T, error),
	store func(T, error) bool,
	L *list.List[T],
	ttyout io.Writer) error {

	session := &pager.Session[T]{
		List:   L,
		TtyOut: ttyout,
		Pager:  (*pager.Pager[T])(pg),
	}

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

	const interval = 4
	displayUpdateTime := time.Now().Add(time.Second / interval)
	newStore := func(obj T, err error) (cont bool) {
		cont = store(obj, err)
		if !cont || time.Now().After(displayUpdateTime) {
			session.UpdateStatus()
			displayUpdateTime = time.Now().Add(time.Second / interval)
		}
		return
	}

	i := 0
	for {
		data, err := getter()
		if !store(data, err) {
			session.GetKey = tty.GetKey
			break
		}
		i++
		if i >= session.Pager.Height {
			newTtyX1 := newTtyX(tty, getter, newStore)
			session.GetKey = newTtyX1.GetKey
			defer newTtyX1.Close()
			break
		}
	}
	return session.EventLoop()
}
