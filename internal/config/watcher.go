package config

import (
	"log"
	"os"
	"sync"
	"time"
)

type Watcher struct {
	mu       sync.RWMutex
	path     string
	lastMod  time.Time
	onChange func(*Config)
	running  bool
	stopCh   chan struct{}
	interval time.Duration
}

func NewWatcher(path string, onChange func(*Config), intervalSec int) *Watcher {
	if intervalSec <= 0 {
		intervalSec = 5
	}
	return &Watcher{
		path:     path,
		onChange: onChange,
		stopCh:   make(chan struct{}),
		interval: time.Duration(intervalSec) * time.Second,
	}
}

func (w *Watcher) Start() error {
	info, err := os.Stat(w.path)
	if err != nil {
		return err
	}
	w.lastMod = info.ModTime()
	w.running = true

	go func() {
		ticker := time.NewTicker(w.interval)
		defer ticker.Stop()
		for {
			select {
			case <-w.stopCh:
				return
			case <-ticker.C:
				w.check()
			}
		}
	}()

	log.Printf("[config-watcher] watching %s (interval: %s)", w.path, w.interval)
	return nil
}

func (w *Watcher) check() {
	info, err := os.Stat(w.path)
	if err != nil {
		return
	}
	if info.ModTime().After(w.lastMod) {
		w.lastMod = info.ModTime()
		cfg, err := Load(w.path)
		if err != nil {
			log.Printf("[config-watcher] reload error: %v", err)
			return
		}
		log.Printf("[config-watcher] config changed, reloading")
		w.mu.Lock()
		fn := w.onChange
		w.mu.Unlock()
		if fn != nil {
			fn(cfg)
		}
	}
}

func (w *Watcher) Stop() {
	if w.running {
		close(w.stopCh)
		w.running = false
		log.Printf("[config-watcher] stopped")
	}
}

func (w *Watcher) SetOnChange(fn func(*Config)) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.onChange = fn
}
