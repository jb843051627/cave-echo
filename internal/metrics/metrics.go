package metrics

import "sync"

type Registry struct {
	mu     sync.RWMutex
	values map[string]int64
}

func New() *Registry                            { return &Registry{values: map[string]int64{}} }
func (r *Registry) Add(key string, delta int64) { r.values[key] += delta }
func (r *Registry) Get(key string) int64        { r.mu.RLock(); defer r.mu.RUnlock(); return r.values[key] }
func (r *Registry) Snapshot() map[string]int64 { return r.values }
