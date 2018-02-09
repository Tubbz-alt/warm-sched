package main

import (
	"../core"
	"../events"
	"fmt"
	"path"
	"sync"
)

type SnapStatus struct {
	Id            string
	Description   string
	Captured      bool
	Summary       string
	LoadCount     int //实际被加载次数
	OccurCount    int //满足被加载条件次数
	ConfigurePath string
}

func (h *History) SnapStatus() []SnapStatus {
	var ret []SnapStatus
	for _, v := range h.status {
		ret = append(ret, v)
	}
	return ret
}

type History struct {
	status   map[string]SnapStatus
	cacheDir string
	ss       *snapshotSource
}

func NewHistory(cache string) *History {
	ss := &snapshotSource{
		loaded: make(map[string]bool),
	}
	events.Register(ss)
	return &History{
		cacheDir: cache,
		status:   make(map[string]SnapStatus),
		ss:       ss,
	}
}

// implement snapshot apply events
type snapshotSource struct {
	lock   sync.Mutex
	loaded map[string]bool
}

func (s *snapshotSource) markLoaded(id string) {
	s.lock.Lock()
	s.loaded[id] = true
	s.lock.Unlock()
}

func (*snapshotSource) Scope() string { return "snapshot" }
func (s *snapshotSource) Check(ids []string) []string {
	var ret []string
	s.lock.Lock()
	for _, id := range ids {
		if s.loaded[id] {
			ret = append(ret, id)
		}
	}
	s.lock.Unlock()
	return ret
}

func (h *History) Has(id string) bool    { return FileExist(h.path(id)) }
func (h *History) path(id string) string { return path.Join(h.cacheDir, id) }

func (h *History) DoCapture(id string, methods []*core.CaptureMethod) error {
	snap, err := core.CaptureSnapshot(methods...)
	if err != nil {
		return fmt.Errorf("DoCapture %q failed: %v", id, err)
	}
	return StoreTo(h.path(id), snap)
}

func (h *History) DoApply(id string) error {
	var snap core.Snapshot
	err := LoadFrom(h.path(id), &snap)
	if err != nil {
		return err
	}
	err = core.ApplySnapshot(&snap, true)
	if err != nil {
		return err
	}
	h.ss.markLoaded(id)
	return nil
}
