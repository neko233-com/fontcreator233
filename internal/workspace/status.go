package workspace

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
)

// Status summarizes a workspace.
type Status struct {
	Dir          string   `json:"dir"`
	Files        int      `json:"files"`
	Builds       int      `json:"builds"`
	Staging      int      `json:"staging"`
	Bytes        int64    `json:"bytes"`
	StagingBytes int64    `json:"stagingBytes"`
	Reclaimable  int64    `json:"reclaimable"`
	Modified     []string `json:"modified"`
	Missing      []string `json:"missing"`
}

// Status inspects state and scratch directories.
func (w *Workspace) Status() (Status, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	st := Status{Dir: w.Dir, Files: len(w.State.Files), Builds: len(w.State.Builds)}
	for rel, rec := range w.State.Files {
		final, err := safeJoin(w.Dir, rel)
		if err != nil {
			st.Missing = append(st.Missing, rel)
			continue
		}
		info, err := os.Lstat(final)
		if os.IsNotExist(err) {
			st.Missing = append(st.Missing, rel)
			continue
		}
		if err != nil || info.Mode()&os.ModeSymlink != 0 {
			st.Modified = append(st.Modified, rel)
			continue
		}
		st.Bytes += info.Size()
		sum, err := hashFile(final)
		if err != nil || sum != rec.SHA256 {
			st.Modified = append(st.Modified, rel)
			continue
		}
		st.Reclaimable += info.Size()
	}
	if entries, err := os.ReadDir(w.TmpDir); err == nil {
		for _, e := range entries {
			st.Staging++
			_ = filepath.WalkDir(filepath.Join(w.TmpDir, e.Name()), func(path string, d fs.DirEntry, err error) error {
				if err != nil || d.IsDir() {
					return nil
				}
				if info, err := d.Info(); err == nil {
					st.StagingBytes += info.Size()
				}
				return nil
			})
		}
	}
	st.Reclaimable += st.StagingBytes
	sort.Strings(st.Modified)
	sort.Strings(st.Missing)
	return st, nil
}
