package asyncpager

import (
	"container/list"
	"io"
	"time"

	"github.com/nyaosorg/go-ttyadapter"

	"github.com/hymkor/jegan/internal/nonblock"
	"github.com/hymkor/jegan/internal/pager"
)

type Displayer = pager.Displayer

type Session = pager.Session

type ttyX struct {
	ttyadapter.Tty
	nonBlock *nonblock.NonBlock[Displayer]
	work     func(Displayer, error) bool
}

func newTtyX(
	tty ttyadapter.Tty,
	dataGetter func() (Displayer, error),
	work func(Displayer, error) bool) *ttyX {

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

type Pager pager.Pager

func (pg *Pager) EventLoop(
	tty ttyadapter.Tty,
	getter func() (Displayer, error),
	store func(Displayer, error) bool,
	L *list.List,
	ttyout io.Writer) error {

	session := &Session{
		List:   L,
		TtyOut: ttyout,
		Pager:  (*pager.Pager)(pg),
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
	newStore := func(obj Displayer, err error) (cont bool) {
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
