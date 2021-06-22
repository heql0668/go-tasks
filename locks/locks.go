package locks

import "sync"

type LocalLock struct {
    mu sync.Mutex
}

func (l *LocalLock) Lock(resource string) bool {
    l.mu = sync.Mutex{}
    l.mu.Lock()
    return true
}

func (l *LocalLock) Unlock() bool {
    l.mu.Unlock()
    return true
}
