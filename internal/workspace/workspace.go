// Package workspace provides versioned build output state, staging, locking,
// and conservative cleanup.
package workspace

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const stateSchema = "fontcreator233/workspace/v1"

// Options controls workspace behavior.
type Options struct {
	NoAutoClean bool
	KeepHistory int
	Force       bool
}

// FileRecord is one managed output file.
type FileRecord struct {
	SHA256  string `json:"sha256"`
	Size    int64  `json:"size"`
	Family  string `json:"family"`
	Style   string `json:"style"`
	Target  string `json:"target"`
	BuildID string `json:"buildId"`
	BuiltAt int64  `json:"builtAt"`
}

// BuildRecord is one committed build.
type BuildRecord struct {
	ID         string    `json:"id"`
	Recipe     string    `json:"recipe,omitempty"`
	Family     string    `json:"family"`
	Style      string    `json:"style"`
	Targets    []string  `json:"targets,omitempty"`
	StartedAt  time.Time `json:"startedAt"`
	FinishedAt time.Time `json:"finishedAt"`
	Files      []string  `json:"files"`
}

// State is the persisted workspace manifest.
type State struct {
	Schema    string                `json:"schema"`
	CreatedAt string                `json:"createdAt"`
	Files     map[string]FileRecord `json:"files"`
	Builds    []BuildRecord         `json:"builds"`
}

// BuildMeta identifies one staged build.
type BuildMeta struct {
	ID      string
	Recipe  string
	Family  string
	Style   string
	Targets []string
}

// Stage is an isolated directory for one build.
type Stage struct {
	Root    string
	meta    BuildMeta
	started time.Time
	ws      *Workspace
}

// Path resolves a relative path inside the stage.
func (s *Stage) Path(rel string) string {
	if rel == "" {
		return s.Root
	}
	return filepath.Join(s.Root, filepath.FromSlash(rel))
}

// Workspace is one output directory.
type Workspace struct {
	Dir       string
	MetaDir   string
	StatePath string
	TmpDir    string
	BackupDir string
	LockPath  string
	State     State

	opts     Options
	mu       sync.Mutex
	lockHeld bool
}

// Open loads or creates the workspace for dir.
func Open(dir string, opts Options) (*Workspace, error) {
	if dir == "" {
		dir = "."
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(abs, 0o755); err != nil {
		return nil, err
	}
	meta := filepath.Join(abs, ".fontcreator233")
	w := &Workspace{
		Dir:       abs,
		MetaDir:   meta,
		StatePath: filepath.Join(meta, "state.json"),
		TmpDir:    filepath.Join(meta, "tmp"),
		BackupDir: filepath.Join(meta, "backup"),
		LockPath:  filepath.Join(meta, "lock"),
		opts:      opts,
	}
	for _, d := range []string{w.MetaDir, w.TmpDir, w.BackupDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return nil, err
		}
	}
	if opts.KeepHistory <= 0 {
		w.opts.KeepHistory = 100
	}
	if err := w.loadState(); err != nil {
		return nil, err
	}
	return w, nil
}

// OpenReadOnly loads workspace state without creating directories. It is used
// by status so a read-only query does not leave a dist/.fontcreator233 tree
// behind.
func OpenReadOnly(dir string) (*Workspace, error) {
	if dir == "" {
		dir = "."
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	meta := filepath.Join(abs, ".fontcreator233")
	w := &Workspace{
		Dir:       abs,
		MetaDir:   meta,
		StatePath: filepath.Join(meta, "state.json"),
		TmpDir:    filepath.Join(meta, "tmp"),
		BackupDir: filepath.Join(meta, "backup"),
		LockPath:  filepath.Join(meta, "lock"),
		opts:      Options{KeepHistory: 100},
	}
	if _, err := os.Stat(abs); os.IsNotExist(err) {
		w.State = State{Schema: stateSchema, Files: map[string]FileRecord{}}
		return w, nil
	}
	if err := w.loadState(); err != nil {
		return nil, err
	}
	return w, nil
}

func (w *Workspace) loadState() error {
	data, err := os.ReadFile(w.StatePath)
	if os.IsNotExist(err) {
		w.State = State{
			Schema:    stateSchema,
			CreatedAt: time.Now().UTC().Format(time.RFC3339),
			Files:     map[string]FileRecord{},
			Builds:    []BuildRecord{},
		}
		return nil
	}
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, &w.State); err != nil {
		return fmt.Errorf("workspace state: %w", err)
	}
	if w.State.Schema != stateSchema {
		return fmt.Errorf("workspace state: unsupported schema %q", w.State.Schema)
	}
	if w.State.Files == nil {
		w.State.Files = map[string]FileRecord{}
	}
	return nil
}

func (w *Workspace) saveState() error {
	data, err := json.MarshalIndent(w.State, "", "  ")
	if err != nil {
		return err
	}
	tmp := w.StatePath + ".tmp"
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	if _, err := f.Write(append(data, '\n')); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmp, w.StatePath); err != nil {
		return err
	}
	return nil
}

// Begin acquires the workspace lock, prunes stale scratch, and creates a stage.
func (w *Workspace) Begin(meta BuildMeta) (*Stage, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if err := w.acquireLock(); err != nil {
		return nil, err
	}
	if !w.opts.NoAutoClean {
		w.autoClean()
	}
	if meta.ID == "" {
		meta.ID = fmt.Sprintf("%d-%d", time.Now().UnixNano(), os.Getpid())
	}
	root := filepath.Join(w.TmpDir, meta.ID)
	if err := os.MkdirAll(root, 0o755); err != nil {
		w.releaseLock()
		return nil, err
	}
	return &Stage{Root: root, meta: meta, started: time.Now(), ws: w}, nil
}

// Abort removes a stage and releases the lock.
func (w *Workspace) Abort(stage *Stage) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if stage != nil {
		_ = os.RemoveAll(stage.Root)
	}
	w.releaseLock()
	return nil
}

// Commit atomically moves a stage into the output directory and updates state.
func (w *Workspace) Commit(stage *Stage) ([]string, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if stage == nil {
		return nil, fmt.Errorf("workspace: nil stage")
	}
	newFiles, err := scanStage(stage.Root)
	if err != nil {
		return nil, err
	}
	if len(newFiles) == 0 {
		return nil, fmt.Errorf("workspace: stage has no files")
	}
	oldForFamily := map[string]FileRecord{}
	for rel, rec := range w.State.Files {
		if rec.Family == stage.meta.Family && rec.Style == stage.meta.Style {
			oldForFamily[rel] = rec
		}
	}
	backupRoot := filepath.Join(w.BackupDir, stage.meta.ID)
	type move struct {
		rel     string
		final   string
		backup  string
		existed bool
		moved   bool
	}
	var moves []move
	rollback := func() {
		for i := len(moves) - 1; i >= 0; i-- {
			m := moves[i]
			if m.existed {
				_ = os.Remove(m.final)
				if _, err := os.Stat(m.backup); err == nil {
					_ = os.MkdirAll(filepath.Dir(m.final), 0o755)
					_ = os.Rename(m.backup, m.final)
				}
				continue
			}
			if m.moved {
				_ = os.Remove(m.final)
			}
		}
		_ = w.loadState()
	}
	fail := func(err error) ([]string, error) {
		rollback()
		return nil, err
	}
	rels := make([]string, 0, len(newFiles))
	for rel := range newFiles {
		rels = append(rels, rel)
	}
	sort.Strings(rels)

	// Move existing outputs into backup before publishing anything. All
	// backups stay recorded so rollback can restore the previous build even
	// when a failure happens before the next file is moved.
	for _, rel := range rels {
		final, err := safeJoin(w.Dir, rel)
		if err != nil {
			return fail(err)
		}
		m := move{rel: rel, final: final}
		info, err := os.Lstat(final)
		switch {
		case os.IsNotExist(err):
		case err != nil:
			return fail(err)
		case info.Mode()&os.ModeSymlink != 0:
			return fail(fmt.Errorf("workspace: refusing to replace symlink %s", rel))
		case !info.Mode().IsRegular():
			return fail(fmt.Errorf("workspace: refusing to replace non-regular file %s", rel))
		default:
			m.existed = true
			rec, tracked := w.State.Files[rel]
			if !tracked && !w.opts.Force {
				return fail(fmt.Errorf("workspace: refusing to overwrite untracked file %s (use force)", rel))
			}
			if tracked {
				sum, err := hashFile(final)
				if err != nil {
					return fail(err)
				}
				if sum != rec.SHA256 && !w.opts.Force {
					return fail(fmt.Errorf("workspace: refusing to overwrite modified file %s (use force)", rel))
				}
			}
			m.backup = filepath.Join(backupRoot, filepath.FromSlash(rel))
			if err := os.MkdirAll(filepath.Dir(m.backup), 0o755); err != nil {
				return fail(err)
			}
			if err := os.Rename(final, m.backup); err != nil {
				return fail(err)
			}
		}
		moves = append(moves, m)
	}
	for i := range moves {
		m := &moves[i]
		if err := os.MkdirAll(filepath.Dir(m.final), 0o755); err != nil {
			return fail(err)
		}
		staged := stage.Path(m.rel)
		if err := os.Rename(staged, m.final); err != nil {
			// Cross-device fallback.
			if err := copyFile(staged, m.final); err != nil {
				return fail(err)
			}
			_ = os.Remove(staged)
		}
		m.moved = true
	}

	// Stale files from the same family are moved to backup first too, so a
	// failed state write leaves the previous build exactly as it was.
	now := time.Now().Unix()
	for rel, rec := range oldForFamily {
		if _, still := newFiles[rel]; still {
			continue
		}
		final, err := safeJoin(w.Dir, rel)
		if err != nil {
			continue
		}
		sum, err := hashFile(final)
		if err != nil || sum != rec.SHA256 {
			continue
		}
		backup := filepath.Join(backupRoot, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(backup), 0o755); err != nil {
			return fail(err)
		}
		if err := os.Rename(final, backup); err != nil {
			return fail(err)
		}
		delete(w.State.Files, rel)
		moves = append(moves, move{rel: rel, final: final, backup: backup, existed: true, moved: true})
	}
	for rel, staged := range newFiles {
		rec := FileRecord{
			SHA256:  staged.sha,
			Size:    staged.size,
			Family:  stage.meta.Family,
			Style:   stage.meta.Style,
			Target:  inferTarget(rel),
			BuildID: stage.meta.ID,
			BuiltAt: now,
		}
		w.State.Files[rel] = rec
	}
	files := make([]string, 0, len(newFiles))
	for rel := range newFiles {
		files = append(files, rel)
	}
	sort.Strings(files)
	w.State.Builds = append(w.State.Builds, BuildRecord{
		ID:         stage.meta.ID,
		Recipe:     stage.meta.Recipe,
		Family:     stage.meta.Family,
		Style:      stage.meta.Style,
		Targets:    append([]string(nil), stage.meta.Targets...),
		StartedAt:  stage.started,
		FinishedAt: time.Now(),
		Files:      files,
	})
	w.pruneHistory()
	if err := w.saveState(); err != nil {
		return fail(err)
	}
	_ = os.RemoveAll(stage.Root)
	_ = os.RemoveAll(backupRoot)
	w.releaseLock()
	return files, nil
}

type stagedFile struct {
	sha  string
	size int64
}

func scanStage(root string) (map[string]stagedFile, error) {
	out := map[string]stagedFile{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if d.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("workspace: symlink in stage: %s", path)
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		sum, size, err := hashFileSize(path)
		if err != nil {
			return err
		}
		out[rel] = stagedFile{sha: sum, size: size}
		return nil
	})
	return out, err
}

func (w *Workspace) autoClean() {
	// Staging older than 24h.
	if entries, err := os.ReadDir(w.TmpDir); err == nil {
		for _, e := range entries {
			info, err := e.Info()
			if err == nil && time.Since(info.ModTime()) > 24*time.Hour {
				_ = os.RemoveAll(filepath.Join(w.TmpDir, e.Name()))
			}
		}
	}
	// Backups older than 7d.
	if entries, err := os.ReadDir(w.BackupDir); err == nil {
		for _, e := range entries {
			info, err := e.Info()
			if err == nil && time.Since(info.ModTime()) > 7*24*time.Hour {
				_ = os.RemoveAll(filepath.Join(w.BackupDir, e.Name()))
			}
		}
	}
	if w.pruneHistory() {
		_ = w.saveState()
	}
}

func (w *Workspace) pruneHistory() bool {
	if len(w.State.Builds) == 0 {
		return false
	}
	cutoff := time.Now().Add(-30 * 24 * time.Hour)
	var kept []BuildRecord
	changed := false
	for _, b := range w.State.Builds {
		if b.FinishedAt.Before(cutoff) {
			changed = true
			continue
		}
		kept = append(kept, b)
	}
	if len(kept) > w.opts.KeepHistory {
		changed = true
		kept = kept[len(kept)-w.opts.KeepHistory:]
	}
	if changed {
		w.State.Builds = kept
	}
	return changed
}

func inferTarget(rel string) string {
	parts := strings.Split(filepath.ToSlash(rel), "/")
	for _, part := range parts {
		switch part {
		case "unity":
			return "unity"
		case "unreal":
			return "unreal"
		case "web":
			return "web"
		case "system":
			return "system"
		}
	}
	return "universal"
}

func safeJoin(root, rel string) (string, error) {
	rel = filepath.FromSlash(rel)
	if filepath.IsAbs(rel) {
		return "", fmt.Errorf("workspace: absolute path %q", rel)
	}
	final := filepath.Join(root, rel)
	check, err := filepath.Rel(root, final)
	if err != nil || check == ".." || strings.HasPrefix(check, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("workspace: path escapes output directory: %q", rel)
	}
	// Reject symlinked ancestors so a tracked path cannot redirect writes or
	// deletes outside the workspace. The root itself may be a symlink chosen
	// by the user; only components below it are checked.
	current := root
	parts := strings.Split(check, string(filepath.Separator))
	for i, part := range parts {
		if part == "" || part == "." {
			continue
		}
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if os.IsNotExist(err) {
			break
		}
		if err != nil {
			return "", err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("workspace: symlink in output path: %q", rel)
		}
		if i < len(parts)-1 && !info.IsDir() {
			return "", fmt.Errorf("workspace: path component is not a directory: %q", rel)
		}
	}
	return final, nil
}

func hashFile(path string) (string, error) {
	sum, _, err := hashFileSize(path)
	return sum, err
}

func hashFileSize(path string) (string, int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", 0, err
	}
	defer f.Close()
	h := sha256.New()
	n, err := io.Copy(h, f)
	if err != nil {
		return "", 0, err
	}
	return hex.EncodeToString(h.Sum(nil)), n, nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}

func writeJSON(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}
