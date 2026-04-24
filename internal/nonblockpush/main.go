package nonblockpush

import (
	"context"
	"errors"
	"io"
	"sync"
)

var (
	ErrNoKeyResponse = errors.New("no key response")

	// ErrNoDataResponse indicates that no more data will be produced.
	// Currently it is aliased to io.EOF for backward compatibility.
	ErrNoDataResponse = io.EOF // errors.New("no data response")
)

type keyResponse struct {
	key string
	err error
}

type DataStream[T any] struct {
	val T
	err error
}

func (d DataStream[T]) Val() T     { return d.val }
func (d DataStream[T]) Err() error { return d.err }
func NewDataStream[T any](v T, e error) DataStream[T] {
	return DataStream[T]{val: v, err: e}
}

type NonBlockPush[T any] struct {
	chKeyReq  chan struct{}
	chKeyRes  chan keyResponse
	chDataRes chan DataStream[T]
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	GetOr     func(work func(val T, err error) bool) (string, error)
}

func New[T any](keyGetter func() (string, error)) *NonBlockPush[T] {
	ctx, cancel := context.WithCancel(context.Background())

	w := &NonBlockPush[T]{
		chKeyReq:  make(chan struct{}),
		chKeyRes:  make(chan keyResponse),
		chDataRes: make(chan DataStream[T]),
		cancel:    cancel,
	}
	w.wg.Add(1)

	go func() {
		defer w.wg.Done()
		defer close(w.chKeyRes)
		for {
			select {
			case <-ctx.Done():
				return
			case <-w.chKeyReq:
				key, err := keyGetter()
				select {
				case w.chKeyRes <- keyResponse{key: key, err: err}:
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	w.GetOr = w.getOr2
	return w
}

func (w *NonBlockPush[T]) getOr1(work func(val T, err error) bool) (string, error) {
	w.chKeyReq <- struct{}{}
	res, ok := <-w.chKeyRes
	if !ok {
		return "", ErrNoKeyResponse
	}
	return res.key, res.err
}

func (w *NonBlockPush[T]) getOr2(work func(val T, err error) bool) (string, error) {
	w.chKeyReq <- struct{}{}
	for {
		select {
		case res, ok := <-w.chKeyRes:
			if !ok {
				return "", ErrNoKeyResponse
			}
			return res.key, res.err
		case res, ok := <-w.chDataRes:
			if !ok || work == nil || !work(res.val, res.err) {
				res, ok := <-w.chKeyRes
				w.GetOr = w.getOr1
				if !ok {
					return "", ErrNoKeyResponse
				}
				return res.key, res.err
			}
		}
	}
}

func (w *NonBlockPush[T]) DataStream() chan DataStream[T] {
	return w.chDataRes
}

func (w *NonBlockPush[T]) Close() {
	w.cancel()
	w.wg.Wait()
}
