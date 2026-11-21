package library

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestLatinStarterPacks guards the four professional family templates. A
// missing or malformed SVG would silently fall back to the parametric skeleton,
// which changes the family's voice without a build error.
func TestLatinStarterPacks(t *testing.T) {
	root := filepath.Join("..", "..", "assets", "latin")
	packs := []string{"hei", "song", "puhui", "mono"}

	for _, pack := range packs {
		dir := filepath.Join(root, pack)
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatalf("%s: %v", pack, err)
		}
		seen := map[string]bool{}
		for _, entry := range entries {
			name := entry.Name()
			if entry.IsDir() || !strings.HasSuffix(strings.ToLower(name), ".svg") {
				continue
			}
			seen[name] = true
			data, err := os.ReadFile(filepath.Join(dir, name))
			if err != nil {
				t.Fatalf("%s/%s: %v", pack, name, err)
			}
			ov, err := ParseSVG(data, filepath.Join(dir, name))
			if err != nil {
				t.Fatalf("%s/%s: %v", pack, name, err)
			}
			if ov.Advance <= 0 {
				t.Fatalf("%s/%s: advance = %d", pack, name, ov.Advance)
			}
			if len(ov.Path) == 0 || len(ov.Path[0]) == 0 {
				t.Fatalf("%s/%s: empty outline", pack, name)
			}
			if pack == "mono" && ov.Advance != 600 {
				t.Fatalf("%s/%s: mono advance = %d, want 600", pack, name, ov.Advance)
			}
		}
		if len(seen) < 90 {
			t.Fatalf("%s: only %d SVG glyphs", pack, len(seen))
		}
		if !seen["U+0067.svg"] {
			t.Fatalf("%s: missing U+0067.svg", pack)
		}
	}
}
