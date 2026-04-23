package nonblockpush

import (
	"context"
	"errors"
	"io"
	"os"
	"sync"
	"time"
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

type dataResponse[T any] struct {
	val T
	err error
}

type NonBlockPush[T any] struct {
	chKeyReq  chan struct{}
	chKeyRes  chan keyResponse
	chDataRes chan dataResponse[T]
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	GetOr     func(work func(val T, err error) bool) (string, error)
}

func New[T any](keyGetter func() (string, error)) *NonBlockPush[T] {
	ctx, cancel := context.WithCancel(context.Background())

	w := &NonBlockPush[T]{
		chKeyReq:  make(chan struct{}),
		chKeyRes:  make(chan keyResponse),
		chDataRes: make(chan dataResponse[T]),
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

func (w *NonBlockPush[T]) PushData(ctx context.Context, data T, err error) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case w.chDataRes <- dataResponse[T]{val: data, err: err}:
		return nil
	}
}

func (w *NonBlockPush[T]) CloseData() {
	close(w.chDataRes)
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

func (w *NonBlockPush[T]) Fetch() (T, error) {
	res, ok := <-w.chDataRes
	if !ok {
		w.GetOr = w.getOr1
		var zero T
		return zero, ErrNoDataResponse
	}
	if errors.Is(res.err, io.EOF) {
		w.GetOr = w.getOr1
	}
	return res.val, res.err
}

// TryFetch reads a single data item with a timeout.
// This method is intended for use cases where only data retrieval is needed
// and no key input is involved.
// If the timeout expires, it returns os.ErrDeadlineExceeded.
// If the data input channel is closed, it returns io.EOF.
func (w *NonBlockPush[T]) TryFetch(timeout time.Duration) (T, error) {
	var zero T
	select {
	case res, ok := <-w.chDataRes:
		if !ok {
			w.GetOr = w.getOr1
			return zero, ErrNoDataResponse
		}
		if errors.Is(res.err, io.EOF) {
			w.GetOr = w.getOr1
		}
		return res.val, res.err
	case <-time.After(timeout):
		return zero, os.ErrDeadlineExceeded
	}
}

func (w *NonBlockPush[T]) Close() {
	w.cancel()
	w.wg.Wait()
}
