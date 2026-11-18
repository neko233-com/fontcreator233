package specimen

import (
	"net/url"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// Open launches the platform browser for an HTML file.
func Open(path string) error {
	name, args := openCommand(runtime.GOOS, path)
	return exec.Command(name, args...).Start()
}

func openCommand(goos, path string) (string, []string) {
	switch goos {
	case "windows":
		// Avoid cmd.exe entirely: paths can contain '&', '%', '^', or '!'.
		// FileProtocolHandler receives a file URL through CreateProcess, so
		// shell metacharacters are never interpreted.
		abs, err := filepath.Abs(path)
		if err != nil {
			abs = path
		}
		p := filepath.ToSlash(abs)
		if !strings.HasPrefix(p, "/") {
			p = "/" + p
		}
		u := url.URL{Scheme: "file", Path: p}
		return "rundll32.exe", []string{"url.dll,FileProtocolHandler", u.String()}
	case "darwin":
		return "open", []string{path}
	default:
		return "xdg-open", []string{path}
	}
}
