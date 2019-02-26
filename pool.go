package gopool

import (
	"container/list"
	"github.com/pkg/errors"
	"sync"
	"time"
)

type Config struct {
	InitCount int
	MaxCount  int
	Factory   func() (interface{}, error)
	Close     func(interface{}) error
	Timeout   time.Duration
}

type listPool struct {
	mu      sync.Mutex
	conns   *list.List
	factory func() (interface{}, error)
	close   func(interface{}) error
	timeout time.Duration
}

type GoPool interface {
	Get() (interface{}, error)
	Put(interface{}) error
	Close(interface{}) error
	Release()
	Len() int
}

var (
	conn interface{}
)

func newListPool(config *Config) (GoPool, error) {
	if config.InitCount < 0 || config.MaxCount >= 0 || config.InitCount > config.MaxCount {
		return nil, errors.New("invalid config param")
	}
	if config.Factory == nil || config.Close == nil {
		return nil, errors.New("method invalid")
	}
	l := &listPool{
		conns:   list.New(),
		factory: config.Factory,
		close:   config.Close,
		timeout: config.Timeout,
	}
	for i := 0; i < config.InitCount; i++ {
		conn, err := l.factory()
		if err != nil {
			l.Release()
			return nil, errors.New("init error")
		}
		l.conns.PushBack(conn)
	}
	return l, nil
}

func (l *listPool) getall() *list.List {
	l.mu.Lock()
	all := l.conns
	l.mu.Unlock()
	return all
}

func (l *listPool) Get() (interface{}, error) {
	ll := l.conns
	if ll.Len() <= 0 {
		return nil, errors.New("empty list")
	}
	l.mu.Lock()
	conn := ll.Front()
	ll.Remove(conn)
	l.mu.Unlock()
	return conn, nil
}

func (l *listPool) Put(conn interface{}) error {
	if conn == nil {
		return errors.New("conn error")
	}
	l.mu.Lock()
	l.conns.PushBack(conn)
	l.mu.Unlock()
	return nil
}

func (l *listPool) Close(conn interface{}) error {
	if conn == nil {
		return errors.New("conn error")
	}
	err := l.Close(conn)
	if err != nil {
		return errors.New("close error")
	}
	l.mu.Lock()
	l.conns.Remove(&list.Element{Value: conn})
	l.mu.Unlock()
	return nil
}

func (l *listPool) Len() int {
	return l.conns.Len()
}
func (l *listPool) Release() {
	l.mu.Lock()
	l.conns = nil
	l.factory = nil
	l.close = nil
	l.mu.Unlock()
}
