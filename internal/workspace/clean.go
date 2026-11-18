package workspace

import (
	"io/fs"
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
	if !o.All && len(o.Targets) == 0 && o.OlderThan == 0 {
		return w.cleanScratch(o)
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

// cleanScratch removes staging/backup scratch and prunes history. It never
// touches managed output files, so plain `clean` is safe to run routinely.
func (w *Workspace) cleanScratch(o CleanOptions) (CleanResult, error) {
	res := CleanResult{DryRun: o.DryRun}
	for _, dir := range []string{w.TmpDir, w.BackupDir} {
		entries, err := os.ReadDir(dir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return res, err
		}
		for _, e := range entries {
			rel := filepath.ToSlash(filepath.Join(filepath.Base(dir), e.Name()))
			res.Deleted = append(res.Deleted, rel)
			if info, err := e.Info(); err == nil {
				if e.IsDir() {
					_ = filepath.WalkDir(filepath.Join(dir, e.Name()), func(path string, d fs.DirEntry, err error) error {
						if err == nil && !d.IsDir() {
							if fi, err := d.Info(); err == nil {
								res.Bytes += fi.Size()
							}
						}
						return nil
					})
				} else {
					res.Bytes += info.Size()
				}
			}
		}
		if !o.DryRun {
			_ = os.RemoveAll(dir)
			_ = os.MkdirAll(dir, 0o755)
		}
	}
	if !o.DryRun {
		if w.pruneHistory() {
			if err := w.saveState(); err != nil {
				return res, err
			}
		}
	}
	sort.Strings(res.Deleted)
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
