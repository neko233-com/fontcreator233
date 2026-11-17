package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// CleanOptions selects managed files to remove.
type CleanOptions struct {
	DryRun    bool
	All       bool
	Force     bool
	Targets   []string
	OlderThan time.Duration
}

// CleanResult lists what would be or was removed.
type CleanResult struct {
	Deleted  []string `json:"deleted"`
	Skipped  []string `json:"skipped"`
	Modified []string `json:"modified"`
	Bytes    int64    `json:"bytes"`
	DryRun   bool     `json:"dryRun"`
}

// Clean removes state-tracked output files, never untracked user files.
func (w *Workspace) Clean(o CleanOptions) (CleanResult, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if err := w.acquireLock(); err != nil {
		return CleanResult{}, err
	}
	defer w.releaseLock()
	if !o.All && len(o.Targets) == 0 && o.OlderThan == 0 && !o.DryRun {
		return CleanResult{}, fmt.Errorf("clean: no selection; use --all, --target, or --older-than")
	}
	now := time.Now()
	targets := map[string]bool{}
	for _, t := range o.Targets {
		targets[t] = true
	}
	var rels []string
	for rel := range w.State.Files {
		rels = append(rels, rel)
	}
	sort.Strings(rels)
	res := CleanResult{DryRun: o.DryRun}
	var parents []string
	for _, rel := range rels {
		rec := w.State.Files[rel]
		if !o.All {
			if len(targets) > 0 && !targets[rec.Target] {
				continue
			}
			if o.OlderThan > 0 && now.Sub(time.Unix(rec.BuiltAt, 0)) < o.OlderThan {
				continue
			}
		}
		final, err := safeJoin(w.Dir, rel)
		if err != nil {
			res.Skipped = append(res.Skipped, rel)
			continue
		}
		info, err := os.Lstat(final)
		if os.IsNotExist(err) {
			if !o.DryRun {
				delete(w.State.Files, rel)
			}
			res.Skipped = append(res.Skipped, rel)
			continue
		}
		if err != nil {
			res.Skipped = append(res.Skipped, rel)
			continue
		}
		if info.Mode()&os.ModeSymlink != 0 {
			res.Skipped = append(res.Skipped, rel)
			continue
		}
		sum, err := hashFile(final)
		if err != nil {
			res.Skipped = append(res.Skipped, rel)
			continue
		}
		if sum != rec.SHA256 && !o.Force {
			res.Modified = append(res.Modified, rel)
			continue
		}
		if !o.DryRun {
			if err := os.Remove(final); err != nil {
				res.Skipped = append(res.Skipped, rel)
				continue
			}
			delete(w.State.Files, rel)
			parents = append(parents, filepath.Dir(final))
		}
		res.Deleted = append(res.Deleted, rel)
		res.Bytes += rec.Size
	}
	if !o.DryRun {
		for _, p := range parents {
			removeEmptyParents(w.Dir, p)
		}
		if o.All && len(res.Modified) == 0 && len(res.Skipped) == 0 {
			_ = os.RemoveAll(w.TmpDir)
			_ = os.RemoveAll(w.BackupDir)
			_ = os.Remove(w.StatePath)
			w.State = State{Schema: stateSchema, Files: map[string]FileRecord{}}
			return res, nil
		}
		if err := w.saveState(); err != nil {
			return res, err
		}
	}
	return res, nil
}

func removeEmptyParents(root, dir string) {
	for dir != root && len(dir) > len(root) {
		entries, err := os.ReadDir(dir)
		if err != nil || len(entries) > 0 {
			return
		}
		_ = os.Remove(dir)
		dir = filepath.Dir(dir)
	}
}
