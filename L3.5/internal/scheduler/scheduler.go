package scheduler

import (
	"sync"
	"time"
)

type Item struct {
	deathTime time.Time
}

type CornScheduler struct {
	tick  *time.Ticker
	store map[int64]*Item
	mu    sync.RWMutex
}

func New(d time.Duration) *CornScheduler {
	return &CornScheduler{
		store: make(map[int64]*Item),
		tick:  time.NewTicker(d),
	}
}

func (cs *CornScheduler) AddItem(id int64, duration time.Duration) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	cs.store[id] = &Item{
		deathTime: time.Now().Add(duration),
	}
}

func (cs *CornScheduler) SetInterval(d time.Duration) {
	cs.tick.Reset(d)
}

func (cs *CornScheduler) Do(task func(id int64)) {
	go func() {
		for range cs.tick.C {
			cs.checkStore(task)
		}
	}()
}

func (cs *CornScheduler) checkStore(task func(id int64)) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	now := time.Now()
	for id, item := range cs.store {
		if now.After(item.deathTime) {
			go task(id)
			delete(cs.store, id)
		}
	}
}
