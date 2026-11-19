package importexport

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/neko233-com/fontcreator233/internal/build"
	"github.com/neko233-com/fontcreator233/internal/recipe"
	"github.com/neko233-com/fontcreator233/internal/style"
	"github.com/neko233-com/fontcreator233/internal/workspace"
)

// BundleIndex is the bundle.json schema.
type BundleIndex struct {
	Schema    string            `json:"schema"`
	Generator string            `json:"generator"`
	CreatedAt string            `json:"createdAt"`
	Recipe    string            `json:"recipe"`
	Family    string            `json:"family"`
	Style     string            `json:"style"`
	License   string            `json:"license"`
	Files     map[string]string `json:"files"`
}

// ExportResult summarizes a bundle export.
type ExportResult struct {
	Zip   string `json:"zip"`
	Files int    `json:"files"`
	Bytes int64  `json:"bytes"`
}

// ImportResult summarizes a bundle import.
type ImportResult struct {
	Dir   string   `json:"dir"`
	Files []string `json:"files"`
	Built bool     `json:"built"`
}

const bundleSchema = "fontcreator233/bundle/v1"
const bundleTime = "2026-01-01T00:00:00Z"

// ExportBundle builds a recipe and packs it into a deterministic zip.
func ExportBundle(ctx context.Context, recipePath, outZip string, targets []string) (ExportResult, error) {
	r, err := recipe.Load(recipePath)
	if err != nil {
		return ExportResult{}, err
	}
	tmp, err := os.MkdirTemp("", "fontcreator233-bundle-")
	if err != nil {
		return ExportResult{}, err
	}
	defer os.RemoveAll(tmp)
	r.Output.Specimen = true
	r.Output.Open = false
	if len(targets) > 0 {
		r.Output.Targets = targets
	}
	if _, err := build.Build(ctx, r, build.Options{OutDir: tmp, Targets: r.Output.Targets, SplitCJK: r.Output.SplitCJK, NoOpen: true}, nil); err != nil {
		return ExportResult{}, err
	}
	recipeData, err := os.ReadFile(recipePath)
	if err != nil {
		return ExportResult{}, err
	}
	if r.Glyphs.Overrides != "" {
		src := r.Glyphs.Overrides
		if !filepath.IsAbs(src) && r.SourcePath != "" {
			src = filepath.Join(filepath.Dir(r.SourcePath), src)
		}
		if err := copyDir(src, filepath.Join(tmp, "glyphs")); err != nil {
			return ExportResult{}, fmt.Errorf("bundle: copy glyph overrides: %w", err)
		}
		r.Glyphs.Overrides = "glyphs"
		recipeData, err = yaml.Marshal(r)
		if err != nil {
			return ExportResult{}, err
		}
	}
	if err := os.WriteFile(filepath.Join(tmp, "recipe.yaml"), recipeData, 0o644); err != nil {
		return ExportResult{}, err
	}
	index := BundleIndex{
		Schema:    bundleSchema,
		Generator: "fontcreator233 0.7.0",
		CreatedAt: bundleTime,
		Recipe:    "recipe.yaml",
		Family:    r.Meta.Family,
		Style:     r.Meta.Style,
		License:   r.Meta.License,
		Files:     map[string]string{},
	}
	err = filepath.WalkDir(tmp, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, err := filepath.Rel(tmp, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if rel == "bundle.json" {
			return nil
		}
		sum, err := hashFile(path)
		if err != nil {
			return err
		}
		index.Files[rel] = sum
		return nil
	})
	if err != nil {
		return ExportResult{}, err
	}
	data, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return ExportResult{}, err
	}
	if err := os.WriteFile(filepath.Join(tmp, "bundle.json"), append(data, '\n'), 0o644); err != nil {
		return ExportResult{}, err
	}
	if err := os.MkdirAll(filepath.Dir(outZip), 0o755); err != nil {
		return ExportResult{}, err
	}
	files, err := writeZip(tmp, outZip)
	if err != nil {
		return ExportResult{}, err
	}
	info, _ := os.Stat(outZip)
	size := int64(0)
	if info != nil {
		size = info.Size()
	}
	return ExportResult{Zip: outZip, Files: files, Bytes: size}, nil
}

func writeZip(root, outPath string) (int, error) {
	var names []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		names = append(names, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		return 0, err
	}
	sort.Strings(names)
	f, err := os.Create(outPath)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	zw := zip.NewWriter(f)
	fixed := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for _, name := range names {
		header := &zip.FileHeader{Name: name, Method: zip.Deflate, Modified: fixed}
		header.SetMode(0o644)
		w, err := zw.CreateHeader(header)
		if err != nil {
			return 0, err
		}
		in, err := os.Open(filepath.Join(root, filepath.FromSlash(name)))
		if err != nil {
			return 0, err
		}
		_, err = io.Copy(w, in)
		_ = in.Close()
		if err != nil {
			return 0, err
		}
	}
	if err := zw.Close(); err != nil {
		return 0, err
	}
	return len(names), nil
}

// ImportBundle safely extracts and validates a bundle.
func ImportBundle(ctx context.Context, zipPath, outDir string, buildAfter, force bool) (ImportResult, error) {
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return ImportResult{}, err
	}
	defer zr.Close()
	const maxTotal = 512 << 20
	const maxFile = 256 << 20
	const maxFiles = 20000
	var total int64
	if len(zr.File) > maxFiles {
		return ImportResult{}, fmt.Errorf("bundle: too many files")
	}
	tmp, err := os.MkdirTemp("", "fontcreator233-import-")
	if err != nil {
		return ImportResult{}, err
	}
	defer os.RemoveAll(tmp)
	type entry struct {
		name string
		path string
	}
	var entries []entry
	seen := map[string]bool{}
	for _, f := range zr.File {
		name, err := bundleRel(f.Name)
		if err != nil {
			return ImportResult{}, fmt.Errorf("bundle: %w", err)
		}
		mode := f.Mode()
		if mode&os.ModeSymlink != 0 {
			return ImportResult{}, fmt.Errorf("bundle: symlink %q rejected", name)
		}
		if f.FileInfo().IsDir() {
			continue
		}
		if seen[name] {
			return ImportResult{}, fmt.Errorf("bundle: duplicate entry %q", name)
		}
		seen[name] = true
		if f.UncompressedSize64 > maxFile {
			return ImportResult{}, fmt.Errorf("bundle: file %q too large", name)
		}
		dest := filepath.Join(tmp, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return ImportResult{}, err
		}
		rc, err := f.Open()
		if err != nil {
			return ImportResult{}, err
		}
		out, err := os.Create(dest)
		if err != nil {
			_ = rc.Close()
			return ImportResult{}, err
		}
		// Trust actual bytes, not the central directory's declared size.
		n, copyErr := io.Copy(out, io.LimitReader(rc, maxFile+1))
		_ = out.Close()
		_ = rc.Close()
		if copyErr != nil {
			return ImportResult{}, copyErr
		}
		if n > maxFile {
			return ImportResult{}, fmt.Errorf("bundle: file %q exceeds size limit", name)
		}
		total += n
		if total > maxTotal {
			return ImportResult{}, fmt.Errorf("bundle: uncompressed size exceeds limit")
		}
		entries = append(entries, entry{name: name, path: dest})
	}
	indexData, err := os.ReadFile(filepath.Join(tmp, "bundle.json"))
	if err != nil {
		return ImportResult{}, fmt.Errorf("bundle: missing bundle.json")
	}
	var index BundleIndex
	if err := json.Unmarshal(indexData, &index); err != nil {
		return ImportResult{}, fmt.Errorf("bundle: %w", err)
	}
	if index.Schema != bundleSchema {
		return ImportResult{}, fmt.Errorf("bundle: unsupported schema %q", index.Schema)
	}
	for rel, want := range index.Files {
		clean, err := bundleRel(rel)
		if err != nil {
			return ImportResult{}, fmt.Errorf("bundle: index: %w", err)
		}
		if !seen[clean] {
			return ImportResult{}, fmt.Errorf("bundle: index lists missing file %s", rel)
		}
		got, err := hashFile(filepath.Join(tmp, filepath.FromSlash(clean)))
		if err != nil || got != want {
			return ImportResult{}, fmt.Errorf("bundle: checksum mismatch for %s", rel)
		}
	}
	for name := range seen {
		if name == "bundle.json" {
			continue
		}
		if _, ok := index.Files[name]; !ok {
			return ImportResult{}, fmt.Errorf("bundle: unindexed file %s", name)
		}
	}
	if index.License == "OFL-1.1" {
		if _, err := os.Stat(filepath.Join(tmp, "OFL.txt")); err != nil {
			return ImportResult{}, fmt.Errorf("bundle: OFL.txt missing for OFL font")
		}
	}
	tmpRecipe := filepath.Join(tmp, "recipe.yaml")
	if r, err := recipe.Load(tmpRecipe); err != nil {
		return ImportResult{}, fmt.Errorf("bundle: recipe: %w", err)
	} else if baseEnabled(r) && index.License != "OFL-1.1" {
		return ImportResult{}, fmt.Errorf("bundle: base-derived fonts must be OFL-1.1")
	}
	if entries, err := os.ReadDir(outDir); err == nil && len(entries) > 0 && !force {
		return ImportResult{}, fmt.Errorf("bundle: output directory %s is not empty (use --force)", outDir)
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return ImportResult{}, err
	}
	var files []string
	for _, e := range entries {
		dst := filepath.Join(outDir, filepath.FromSlash(e.name))
		if _, err := os.Stat(dst); err == nil && !force {
			return ImportResult{}, fmt.Errorf("bundle: refusing to overwrite %s (use --force)", e.name)
		}
		if err := copyFile(e.path, dst); err != nil {
			return ImportResult{}, err
		}
		files = append(files, dst)
	}
	res := ImportResult{Dir: outDir, Files: files}
	if buildAfter {
		recipePath := filepath.Join(outDir, "recipe.yaml")
		r, err := recipe.Load(recipePath)
		if err != nil {
			return res, fmt.Errorf("bundle: rebuild recipe: %w", err)
		}
		r.Output.Open = false
		ws, err := workspace.Open(outDir, workspace.Options{Force: true})
		if err != nil {
			return res, err
		}
		if _, _, err := build.BuildToWorkspace(ctx, ws, workspace.BuildMeta{
			Recipe:  recipePath,
			Family:  r.Meta.Family,
			Style:   style.ResolveStyle(r),
			Targets: r.Output.Targets,
		}, r, build.Options{Targets: r.Output.Targets, SplitCJK: r.Output.SplitCJK, NoOpen: true}, nil); err != nil {
			return res, err
		}
		res.Built = true
	}
	return res, nil
}

func bundleRel(name string) (string, error) {
	name = strings.ReplaceAll(name, "\\", "/")
	if name == "" || strings.HasPrefix(name, "/") || path.IsAbs(name) {
		return "", fmt.Errorf("unsafe path %q", name)
	}
	clean := path.Clean(name)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") || strings.ContainsAny(clean, ":\x00") {
		return "", fmt.Errorf("unsafe path %q", name)
	}
	return clean, nil
}

func baseEnabled(r *recipe.Recipe) bool {
	return r.Coverage.Base.CJK != "none" || r.Coverage.Base.Kana ||
		r.Coverage.Base.Hangul || r.Coverage.Base.Symbols
}

func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		if rel == "." {
			return os.MkdirAll(dst, 0o755)
		}
		if d.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlink rejected: %s", p)
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		return copyFile(p, target)
	})
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
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

func hashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
