package metrics

import "sync"

type Registry struct {
	mu     sync.RWMutex
	values map[string]int64
}

func New() *Registry { return &Registry{values: map[string]int64{}} }

func (r *Registry) Add(key string, delta int64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.values[key] += delta
}

func (r *Registry) Get(key string) int64 {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.values[key]
}

// Snapshot 返回当前计数的独立副本，调用方对其的任何修改都不会影响注册表内部存储。
func (r *Registry) Snapshot() map[string]int64 {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make(map[string]int64, len(r.values))
	for k, v := range r.values {
		out[k] = v
	}
	return out
}
