package workspace

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

type lockInfo struct {
	PID     int       `json:"pid"`
	Host    string    `json:"host"`
	Started time.Time `json:"started"`
}

func (w *Workspace) acquireLock() error {
	if w.lockHeld {
		return nil
	}
	info := lockInfo{PID: os.Getpid(), Host: hostname(), Started: time.Now()}
	data, _ := json.Marshal(info)
	f, err := os.OpenFile(w.LockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		if !os.IsExist(err) {
			return err
		}
		existing, readErr := os.ReadFile(w.LockPath)
		var old lockInfo
		if readErr == nil && json.Unmarshal(existing, &old) == nil {
			if time.Since(old.Started) < 30*time.Minute {
				return fmt.Errorf("workspace: another build is running (pid %d, started %s)", old.PID, old.Started.Format(time.RFC3339))
			}
		}
		_ = os.Remove(w.LockPath)
		f, err = os.OpenFile(w.LockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
		if err != nil {
			return fmt.Errorf("workspace: acquire lock: %w", err)
		}
	}
	if _, err := f.Write(append(data, '\n')); err != nil {
		_ = f.Close()
		_ = os.Remove(w.LockPath)
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	w.lockHeld = true
	return nil
}

func (w *Workspace) releaseLock() {
	if !w.lockHeld {
		return
	}
	_ = os.Remove(w.LockPath)
	w.lockHeld = false
}

func hostname() string {
	h, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return h
}
