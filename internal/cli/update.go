package cli

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/neko233-com/fontcreator233/internal/agentschema"
)

// mirrorPrefixes are tried in order; first reachable wins. Each entry is prepended to
// the full GitHub URL, so "" hits GitHub directly and the rest go through proxies
// (ghproxy style: https://ghproxy.net/https://api.github.com/...).
var mirrorPrefixes = []string{
	"",
	"https://ghproxy.net/",
	"https://mirror.ghproxy.com/",
	"https://ghfast.top/",
}

const (
	repoOwner   = "neko233-com"
	repoName    = "fontcreator233"
	apiBase     = "https://api.github.com"
	releaseBase = "https://github.com"
)

func (a *app) cmdUpdate(args []string) int {
	var common commonFlags
	var checkOnly bool
	var force bool
	fs := a.newFlagSet("update", &common)
	fs.BoolVar(&checkOnly, "check", false, "only check for a newer version, don't download")
	fs.BoolVar(&force, "force", false, "re-download even if up-to-date")
	if err := parseFlexible(fs, args, nil); err != nil {
		a.json = common.json
		return a.fail("update", 1, err)
	}
	a.json = common.json

	latest, err := a.fetchLatestTag()
	if err != nil {
		return a.fail("update", 2, fmt.Errorf("checking release: %w", err))
	}

	cur := strings.TrimPrefix(agentschema.Version, "v")
	newVer := strings.TrimPrefix(latest, "v")
	cmp := compareSemver(cur, newVer)

	if a.json {
		a.writeJSON(okMap("update", map[string]any{
			"current": cur, "latest": newVer, "upToDate": cmp >= 0,
		}))
	}
	if cmp >= 0 && !force {
		if !a.json {
			fmt.Fprintf(a.io.Out, "fontcreator233 is up-to-date (%s)\n", latest)
		}
		return 0
	}
	if checkOnly {
		if !a.json {
			fmt.Fprintf(a.io.Out, "New version available: %s -> %s\n", agentschema.Version, latest)
		}
		return 0
	}

	if !a.json {
		fmt.Fprintf(a.io.Out, "Updating %s -> %s ...\n", agentschema.Version, latest)
	}
	if err := a.doUpdate(latest); err != nil {
		return a.fail("update", 2, err)
	}
	if !a.json {
		fmt.Fprintf(a.io.Out, "Done. Restart to use %s.\n", latest)
	}
	return 0
}

func (a *app) fetchLatestTag() (string, error) {
	pageTarget := fmt.Sprintf("%s/%s/%s/releases/latest", releaseBase, repoOwner, repoName)
	apiTarget := fmt.Sprintf("%s/repos/%s/%s/releases/latest", apiBase, repoOwner, repoName)

	var lastErr error
	for _, prefix := range mirrorPrefixes {
		// The /releases/latest 302 is not rate-limited and works through plain proxies.
		if tag, err := a.tagViaRedirect(prefix + pageTarget); err == nil {
			return tag, nil
		} else {
			lastErr = err
		}
		// Fall back to the REST API (may be rate-limited to 403).
		if tag, err := a.httpGetLatest(prefix + apiTarget); err == nil {
			return tag, nil
		} else {
			lastErr = err
		}
	}
	return "", lastErr
}

// tagViaRedirect resolves the newest tag from the un-followed 302 of /releases/latest.
func (a *app) tagViaRedirect(url string) (string, error) {
	client := &http.Client{
		Timeout: 20 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	req, _ := http.NewRequestWithContext(a.ctx, "GET", url, nil)
	req.Header.Set("User-Agent", "fontcreator233-update/"+agentschema.Version)
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	loc := resp.Header.Get("Location")
	if loc == "" {
		return "", fmt.Errorf("HTTP %d without Location", resp.StatusCode)
	}
	const marker = "/releases/tag/"
	idx := strings.LastIndex(loc, marker)
	if idx < 0 {
		return "", fmt.Errorf("unexpected redirect target %q", loc)
	}
	tag := strings.TrimSuffix(loc[idx+len(marker):], "/")
	if tag == "" {
		return "", fmt.Errorf("empty tag in %q", loc)
	}
	return tag, nil
}

func (a *app) httpGetLatest(url string) (string, error) {
	client := &http.Client{Timeout: 15 * time.Second}
	req, _ := http.NewRequestWithContext(a.ctx, "GET", url, nil)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "fontcreator233-update/"+agentschema.Version)
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	var body struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", err
	}
	if body.TagName == "" {
		return "", errors.New("empty tag_name in release payload")
	}
	return body.TagName, nil
}

func compareSemver(a, b string) int {
	pa := parseSemver(a)
	pb := parseSemver(b)
	for i := 0; i < 3; i++ {
		if pa[i] < pb[i] {
			return -1
		}
		if pa[i] > pb[i] {
			return 1
		}
	}
	return 0
}

func parseSemver(s string) [3]int {
	s = strings.TrimPrefix(strings.TrimPrefix(s, "v"), "V")
	parts := strings.SplitN(s, ".", 3)
	out := [3]int{}
	for i := 0; i < len(parts) && i < 3; i++ {
		n, _ := strconv.Atoi(parts[i])
		out[i] = n
	}
	return out
}

func (a *app) doUpdate(tag string) error {
	assetName, archiveName, ext := archiveFor(tag)

	target := fmt.Sprintf("%s/%s/%s/releases/download/%s/%s",
		releaseBase, repoOwner, repoName, tag, archiveName)
	var lastErr error
	for _, prefix := range mirrorPrefixes {
		url := prefix + target
		data, sha, err := a.httpGetArchive(url, url+".sha256")
		if err != nil {
			lastErr = err
			continue
		}
		if !verifySHA(data, sha) {
			lastErr = fmt.Errorf("sha256 mismatch from %s", url)
			continue
		}
		binPath, err := extractBinary(data, archiveName, ext, assetName)
		if err != nil {
			lastErr = err
			continue
		}
		return replaceSelf(binPath)
	}
	return fmt.Errorf("all mirrors failed: %w", lastErr)
}

func archiveFor(tag string) (assetName, archiveName, ext string) {
	// assetName: binary inside archive, archiveName: download filename
	suffix := runtime.GOOS + "-" + runtime.GOARCH
	var asset string
	if runtime.GOOS == "windows" {
		asset = "fontcreator.exe"
	} else {
		asset = "fontcreator"
	}
	return asset, fmt.Sprintf("fontcreator233-%s-%s.tar.gz", tag, suffix), "tar.gz"
}

func (a *app) httpGetArchive(url, shaURL string) ([]byte, string, error) {
	client := &http.Client{Timeout: 60 * time.Second}
	// checksum first; a proxy may return an HTML error page, so only trust a 64-char hex token
	sha := ""
	if shaURL != "" {
		if data, err := httpGetBytes(client, shaURL, a.ctx); err == nil {
			for _, f := range strings.Fields(string(data)) {
				if len(f) == 64 && isHex(f) {
					sha = f
					break
				}
			}
		}
	}
	data, err := httpGetBytes(client, url, a.ctx)
	if err != nil {
		return nil, "", err
	}
	return data, sha, nil
}

func isHex(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		switch {
		case c >= '0' && c <= '9', c >= 'a' && c <= 'f', c >= 'A' && c <= 'F':
		default:
			return false
		}
	}
	return true
}

func httpGetBytes(client *http.Client, url string, ctx context.Context) ([]byte, error) {
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	req.Header.Set("User-Agent", "fontcreator233-update")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

func verifySHA(data []byte, expected string) bool {
	if expected == "" {
		return true // no checksum available — still allow
	}
	sum := sha256.Sum256(data)
	got := hex.EncodeToString(sum[:])
	return strings.EqualFold(got, expected)
}

func extractBinary(data []byte, archiveName, ext, assetName string) (string, error) {
	tmpDir, err := os.MkdirTemp("", "fontcreator-update-*")
	if err != nil {
		return "", err
	}

	switch {
	case strings.HasSuffix(archiveName, ".tar.gz") || ext == "tar.gz":
		return extractTarGZ(data, tmpDir, assetName)
	case strings.HasSuffix(archiveName, ".zip") || ext == "zip":
		return extractZip(data, tmpDir, assetName)
	default:
		// raw binary
		out := filepath.Join(tmpDir, assetName)
		if err := os.WriteFile(out, data, 0o755); err != nil {
			return "", err
		}
		return out, nil
	}
}

func extractTarGZ(data []byte, outDir, assetName string) (string, error) {
	gzr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	defer gzr.Close()
	tr := tar.NewReader(gzr)
	var candidates []string
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
		if hdr.FileInfo().IsDir() {
			continue
		}
		base := filepath.Base(hdr.Name)
		if base == assetName {
			dst := filepath.Join(outDir, assetName)
			f, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
			if err != nil {
				return "", err
			}
			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return "", err
			}
			f.Close()
			return dst, nil
		}
		candidates = append(candidates, base)
	}
	sort.Strings(candidates)
	if len(candidates) == 0 {
		return "", errors.New("archive is empty")
	}
	return "", fmt.Errorf("binary %q not found in archive; candidates: %s", assetName, strings.Join(candidates, ", "))
}

func extractZip(data []byte, outDir, assetName string) (string, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", err
	}
	for _, f := range zr.File {
		if filepath.Base(f.Name) != assetName {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return "", err
		}
		dst := filepath.Join(outDir, assetName)
		out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
		if err != nil {
			rc.Close()
			return "", err
		}
		_, err = io.Copy(out, rc)
		rc.Close()
		out.Close()
		if err != nil {
			return "", err
		}
		return dst, nil
	}
	return "", fmt.Errorf("binary %q not found in zip", assetName)
}

func replaceSelf(newBin string) error {
	self, err := os.Executable()
	if err != nil {
		return err
	}
	backup := self + ".bak"
	// try direct replace first; if running binary is locked (Windows), use rename trick
	if err := os.Rename(self, backup); err == nil {
		defer os.Remove(backup)
	} else {
		// on Windows, can't delete/replace running exe — copy to .new and instruct user
		dst := self + ".new"
		if err := copyFile(newBin, dst); err != nil {
			return fmt.Errorf("cannot replace running binary: %w (new binary at %s)", err, dst)
		}
		fmt.Fprintf(os.Stderr, "fontcreator233-update: cannot replace running binary on Windows.\n")
		fmt.Fprintf(os.Stderr, "  New version saved at %s\n", dst)
		fmt.Fprintf(os.Stderr, "  Close this shell, rename .exe.new -> .exe\n")
		return nil
	}
	return copyFile(newBin, self)
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	st, _ := in.Stat()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, st.Mode())
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}
