package workspace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func openWS(t *testing.T) *Workspace {
	t.Helper()
	w, err := Open(t.TempDir(), Options{KeepHistory: 5})
	if err != nil {
		t.Fatal(err)
	}
	return w
}

func commitFiles(t *testing.T, w *Workspace, family, style, id string, files map[string]string) []string {
	t.Helper()
	stage, err := w.Begin(BuildMeta{ID: id, Family: family, Style: style, Targets: []string{"universal"}})
	if err != nil {
		t.Fatal(err)
	}
	for rel, content := range files {
		path := stage.Path(rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	committed, err := w.Commit(stage)
	if err != nil {
		t.Fatal(err)
	}
	return committed
}

func TestCommitAndState(t *testing.T) {
	w := openWS(t)
	committed := commitFiles(t, w, "Test", "Regular", "b1", map[string]string{
		"Test-Regular.ttf":    "font",
		"assets/specimen.css": "css",
	})
	if len(committed) != 2 {
		t.Fatalf("committed = %v", committed)
	}
	data, err := os.ReadFile(filepath.Join(w.Dir, "Test-Regular.ttf"))
	if err != nil || string(data) != "font" {
		t.Fatalf("font = %q err=%v", data, err)
	}
	if len(w.State.Files) != 2 || len(w.State.Builds) != 1 {
		t.Fatalf("state = %#v", w.State)
	}
	if _, err := os.Stat(w.LockPath); !os.IsNotExist(err) {
		t.Fatalf("lock still exists: %v", err)
	}
	if _, err := os.Stat(stagePath(w, "b1")); !os.IsNotExist(err) {
		t.Fatalf("stage still exists")
	}
}

func TestStaleFileRemovedUserFileKept(t *testing.T) {
	w := openWS(t)
	commitFiles(t, w, "Family", "Regular", "b1", map[string]string{
		"Family-Regular.ttf":  "a",
		"Family-Regular.woff": "b",
	})
	user := filepath.Join(w.Dir, "notes.txt")
	if err := os.WriteFile(user, []byte("user"), 0o644); err != nil {
		t.Fatal(err)
	}
	commitFiles(t, w, "Family", "Regular", "b2", map[string]string{
		"Family-Regular.ttf": "a2",
	})
	if _, err := os.Stat(filepath.Join(w.Dir, "Family-Regular.woff")); !os.IsNotExist(err) {
		t.Fatalf("stale woff not removed: %v", err)
	}
	if _, err := os.Stat(user); err != nil {
		t.Fatalf("user file removed: %v", err)
	}
}

func TestModifiedManagedFileKept(t *testing.T) {
	w := openWS(t)
	commitFiles(t, w, "Family", "Regular", "b1", map[string]string{
		"Family-Regular.ttf":  "a",
		"Family-Regular.woff": "b",
	})
	woff := filepath.Join(w.Dir, "Family-Regular.woff")
	if err := os.WriteFile(woff, []byte("user edit"), 0o644); err != nil {
		t.Fatal(err)
	}
	commitFiles(t, w, "Family", "Regular", "b2", map[string]string{
		"Family-Regular.ttf": "a2",
	})
	data, err := os.ReadFile(woff)
	if err != nil || string(data) != "user edit" {
		t.Fatalf("modified file lost: %q %v", data, err)
	}
}

func TestLock(t *testing.T) {
	w := openWS(t)
	stage, err := w.Begin(BuildMeta{ID: "b1", Family: "F", Style: "R"})
	if err != nil {
		t.Fatal(err)
	}
	w2, err := Open(w.Dir, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w2.Begin(BuildMeta{ID: "b2"}); err == nil {
		t.Fatal("second begin succeeded")
	}
	if err := w.Abort(stage); err != nil {
		t.Fatal(err)
	}
	if _, err := w.Begin(BuildMeta{ID: "b3"}); err != nil {
		t.Fatal(err)
	}
}

func TestStaleLockRecovered(t *testing.T) {
	w := openWS(t)
	old := lockInfo{PID: 999999, Host: "test", Started: time.Now().Add(-time.Hour)}
	if err := writeJSON(w.LockPath, old); err != nil {
		t.Fatal(err)
	}
	if _, err := w.Begin(BuildMeta{ID: "b1"}); err != nil {
		t.Fatalf("stale lock not recovered: %v", err)
	}
}

func TestClean(t *testing.T) {
	w := openWS(t)
	commitFiles(t, w, "Family", "Regular", "b1", map[string]string{
		"Family-Regular.ttf":  "a",
		"Family-Regular.woff": "b",
	})
	user := filepath.Join(w.Dir, "user.txt")
	if err := os.WriteFile(user, []byte("user"), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := w.Clean(CleanOptions{DryRun: true})
	if err != nil || len(res.Deleted) != 2 || res.Bytes == 0 {
		t.Fatalf("dry run = %#v err=%v", res, err)
	}
	if _, err := os.Stat(filepath.Join(w.Dir, "Family-Regular.ttf")); err != nil {
		t.Fatalf("dry run deleted: %v", err)
	}
	res, err = w.Clean(CleanOptions{All: true})
	if err != nil || len(res.Deleted) != 2 {
		t.Fatalf("clean = %#v err=%v", res, err)
	}
	if _, err := os.Stat(user); err != nil {
		t.Fatalf("user file removed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(w.Dir, "Family-Regular.ttf")); !os.IsNotExist(err) {
		t.Fatalf("font still exists: %v", err)
	}
}

func TestCleanSkipsModifiedUnlessForced(t *testing.T) {
	w := openWS(t)
	commitFiles(t, w, "Family", "Regular", "b1", map[string]string{"Family-Regular.ttf": "a"})
	path := filepath.Join(w.Dir, "Family-Regular.ttf")
	if err := os.WriteFile(path, []byte("edited"), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := w.Clean(CleanOptions{All: true})
	if err != nil || len(res.Modified) != 1 || len(res.Deleted) != 0 {
		t.Fatalf("clean = %#v err=%v", res, err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal("modified file deleted")
	}
	res, err = w.Clean(CleanOptions{All: true, Force: true})
	if err != nil || len(res.Deleted) != 1 {
		t.Fatalf("force clean = %#v err=%v", res, err)
	}
}

func TestCleanRejectsEscape(t *testing.T) {
	w := openWS(t)
	commitFiles(t, w, "Family", "Regular", "b1", map[string]string{"Family-Regular.ttf": "a"})
	w.State.Files["../evil.ttf"] = FileRecord{SHA256: strings.Repeat("0", 64), Size: 1, Family: "Family", Style: "Regular"}
	if err := w.saveState(); err != nil {
		t.Fatal(err)
	}
	res, err := w.Clean(CleanOptions{All: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range res.Deleted {
		if strings.Contains(p, "evil") {
			t.Fatalf("escaped path deleted: %v", res.Deleted)
		}
	}
}

func TestStatus(t *testing.T) {
	w := openWS(t)
	commitFiles(t, w, "Family", "Regular", "b1", map[string]string{"Family-Regular.ttf": "a"})
	st, err := w.Status()
	if err != nil {
		t.Fatal(err)
	}
	if st.Files != 1 || st.Bytes != 1 {
		t.Fatalf("status = %#v", st)
	}
	if err := os.WriteFile(filepath.Join(w.Dir, "Family-Regular.ttf"), []byte("edited"), 0o644); err != nil {
		t.Fatal(err)
	}
	st, err = w.Status()
	if err != nil || len(st.Modified) != 1 {
		t.Fatalf("status = %#v err=%v", st, err)
	}
}

func stagePath(w *Workspace, id string) string {
	return filepath.Join(w.TmpDir, id)
}
