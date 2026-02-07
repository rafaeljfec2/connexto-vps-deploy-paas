package lock

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

type Locker struct {
	basePath string
	locks    map[string]*sync.Mutex
	mu       sync.RWMutex
}

func New(basePath string) *Locker {
	return &Locker{
		basePath: basePath,
		locks:    make(map[string]*sync.Mutex),
	}
}

func (l *Locker) Acquire(appID string) error {
	l.mu.Lock()
	lk, exists := l.locks[appID]
	if !exists {
		lk = &sync.Mutex{}
		l.locks[appID] = lk
	}
	l.mu.Unlock()

	lk.Lock()

	lockFile := l.lockFilePath(appID)
	if err := os.MkdirAll(filepath.Dir(lockFile), 0755); err != nil {
		lk.Unlock()
		return fmt.Errorf("failed to create lock directory: %w", err)
	}

	f, err := os.OpenFile(lockFile, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		if os.IsExist(err) {
			lk.Unlock()
			return fmt.Errorf("lock already held for app %s", appID)
		}
		lk.Unlock()
		return fmt.Errorf("failed to create lock file: %w", err)
	}
	f.Close()

	return nil
}

func (l *Locker) Release(appID string) error {
	l.mu.RLock()
	lk, exists := l.locks[appID]
	l.mu.RUnlock()

	if !exists {
		return fmt.Errorf("no lock exists for app %s", appID)
	}

	lockFile := l.lockFilePath(appID)
	if err := os.Remove(lockFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove lock file: %w", err)
	}

	lk.Unlock()
	return nil
}

func (l *Locker) IsLocked(appID string) bool {
	lockFile := l.lockFilePath(appID)
	_, err := os.Stat(lockFile)
	return err == nil
}

func (l *Locker) lockFilePath(appID string) string {
	return filepath.Join(l.basePath, ".locks", appID+".lock")
}

func (l *Locker) CleanupStale() error {
	lockDir := filepath.Join(l.basePath, ".locks")
	entries, err := os.ReadDir(lockDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		lockFile := filepath.Join(lockDir, entry.Name())
		os.Remove(lockFile)
	}

	return nil
}
