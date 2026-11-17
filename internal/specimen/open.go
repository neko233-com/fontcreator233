package specimen

import (
	"os/exec"
	"runtime"
)

// Open launches the platform browser for an HTML file.
func Open(path string) error {
	name, args := openCommand(runtime.GOOS, path)
	return exec.Command(name, args...).Start()
}

func openCommand(goos, path string) (string, []string) {
	switch goos {
	case "windows":
		return "cmd", []string{"/c", "start", "", path}
	case "darwin":
		return "open", []string{path}
	default:
		return "xdg-open", []string{path}
	}
}
