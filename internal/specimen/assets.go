package specimen

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
)

//go:embed webassets
var webFS embed.FS

//go:embed webassets/specimen.html
var specimenTemplate []byte

const assetsVersion = "1"

// EnsureAssets writes the embedded CSS/JS/fonts/licenses into dest.
// assetsDir, when non-empty, overrides outDir/assets.
func EnsureAssets(outDir, assetsDir string) (string, error) {
	root := filepath.Join(outDir, "assets")
	if assetsDir != "" {
		root = assetsDir
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return "", err
	}
	hashes := map[string]string{}
	err := fs.WalkDir(webFS, "webassets", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel := path[len("webassets/"):]
		if rel == "specimen.html" {
			return nil
		}
		data, err := webFS.ReadFile(path)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(data)
		hashes[rel] = hex.EncodeToString(sum[:])
		dest := filepath.Join(root, filepath.FromSlash(rel))
		if existing, err := os.ReadFile(dest); err == nil {
			if sha256.Sum256(existing) == sum {
				return nil
			}
		}
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return err
		}
		return os.WriteFile(dest, data, 0o644)
	})
	if err != nil {
		return "", err
	}
	keys := make([]string, 0, len(hashes))
	for k := range hashes {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	ordered := make(map[string]string, len(hashes))
	for _, k := range keys {
		ordered[k] = hashes[k]
	}
	versions := map[string]any{
		"version": assetsVersion,
		"files":   ordered,
	}
	raw, err := json.MarshalIndent(versions, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(root, "versions.json"), append(raw, '\n'), 0o644); err != nil {
		return "", err
	}
	return assetsVersion, nil
}

func readEmbedded(path string) ([]byte, error) {
	return webFS.ReadFile(path)
}
