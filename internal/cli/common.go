package cli

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/neko233-com/fontcreator233/internal/agentschema"
	"golang.org/x/term"
)

type stringList []string

func (s *stringList) String() string { return strings.Join(*s, ",") }

func (s *stringList) Set(v string) error {
	for _, part := range strings.Split(v, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			*s = append(*s, part)
		}
	}
	return nil
}

type commonFlags struct {
	json, noOpen, noTUI, plain, quiet bool
}

func (a *app) newFlagSet(name string, common *commonFlags) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(a.io.Err)
	fs.BoolVar(&common.json, "json", false, "emit one JSON envelope on stdout")
	fs.BoolVar(&common.noOpen, "no-open", false, "do not open the specimen")
	fs.BoolVar(&common.noTUI, "no-tui", false, "disable the interactive TUI")
	fs.BoolVar(&common.plain, "plain", false, "plain line progress")
	fs.BoolVar(&common.quiet, "quiet", false, "suppress non-error output")
	fs.Usage = func() {
		fmt.Fprintf(a.io.Err, "fontcreator233 %s\n\nUsage: fontcreator %s [flags] [arguments]\n\nFlags:\n", agentschema.Version, name)
		fs.PrintDefaults()
	}
	return fs
}

// parseFlexible lets flags appear before or after positional arguments.
func parseFlexible(fs *flag.FlagSet, args []string, valueFlags map[string]bool) error {
	var flags, pos []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			pos = append(pos, args[i+1:]...)
			break
		}
		if strings.HasPrefix(arg, "-") && arg != "-" {
			flags = append(flags, arg)
			name := strings.TrimLeft(arg, "-")
			if eq := strings.IndexByte(name, '='); eq >= 0 {
				name = name[:eq]
			}
			if valueFlags[name] && !strings.Contains(arg, "=") {
				if i+1 >= len(args) {
					return fmt.Errorf("flag --%s requires a value", name)
				}
				i++
				flags = append(flags, args[i])
			}
			continue
		}
		pos = append(pos, arg)
	}
	return fs.Parse(append(flags, pos...))
}

func (a *app) interactive(common commonFlags) bool {
	if common.json || common.plain || common.noTUI || common.quiet || isCI() {
		return false
	}
	if os.Getenv("NO_COLOR") != "" || strings.EqualFold(os.Getenv("TERM"), "dumb") {
		return false
	}
	f, ok := a.io.Err.(*os.File)
	return ok && term.IsTerminal(int(f.Fd()))
}

func hasJSON(args []string) bool {
	for _, a := range args {
		if a == "--json" || strings.HasPrefix(a, "--json=") {
			return true
		}
	}
	return false
}
