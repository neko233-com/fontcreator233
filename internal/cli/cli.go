// Package cli implements the fontcreator command-line interface.
package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
)

// IO is the injectable command environment.
type IO struct {
	In  io.Reader
	Out io.Writer
	Err io.Writer
}

type app struct {
	io   IO
	json bool
	ctx  context.Context
}

// Run executes one command and returns its process exit code.
func Run(args []string, io IO) int {
	if io.In == nil {
		io.In = os.Stdin
	}
	if io.Out == nil {
		io.Out = os.Stdout
	}
	if io.Err == nil {
		io.Err = os.Stderr
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	a := &app{io: io, json: hasJSON(args), ctx: ctx}
	if len(args) == 0 {
		a.usage()
		return 1
	}
	cmd := args[0]
	rest := args[1:]
	if cmd == "help" || cmd == "--help" || cmd == "-h" {
		a.usage()
		return 0
	}
	if cmd == "--version" {
		cmd = "version"
	}
	switch cmd {
	case "version":
		return a.cmdVersion(rest)
	case "update":
		return a.cmdUpdate(rest)
	case "schema":
		return a.cmdSchema(rest)
	case "describe":
		return a.cmdDescribe(rest)
	case "init":
		return a.cmdInit(rest)
	case "validate":
		return a.cmdValidate(rest)
	case "build":
		return a.cmdBuild(rest)
	case "build-all":
		return a.cmdBuildAll(rest)
	case "inspect":
		return a.cmdInspect(rest)
	case "learn":
		return a.cmdLearn(rest)
	case "specimen":
		return a.cmdSpecimen(rest)
	case "library":
		return a.cmdLibrary(rest)
	case "clean":
		return a.cmdClean(rest)
	case "status":
		return a.cmdStatus(rest)
	case "import":
		return a.cmdImport(rest)
	case "export":
		return a.cmdExport(rest)
	case "mcp":
		return a.cmdMCP(rest)
	default:
		return a.fail(cmd, 1, fmt.Errorf("unknown command %q", cmd))
	}
}

func (a *app) usage() {
	fmt.Fprint(a.io.Err, `fontcreator233 — deterministic, agent-native font creator

Usage:
  fontcreator <command> [flags] [arguments]

Commands:
  build <recipe>            build a recipe
  build-all --dir <dir>     build several recipes in parallel
  validate <recipe>         validate a recipe
  inspect <font>            inspect a TTF/OTF
  learn <font>              deep-report a font and write a starter recipe
  specimen <font>           regenerate the HTML specimen
  init                      write a starter recipe
  describe                  print the agent contract
  schema                    print the recipe JSON Schema
  library list|export       built-in skeleton library
  clean / status            workspace management
  import font|bundle        import a font or bundle
  export glyphs|bundle      export glyphs or a bundle
  mcp                       MCP stdio server
  version                   print the version
  update                    check and install a newer version

Common flags:
  --json --no-open --no-tui --plain --quiet
`)
}

type errorInfo struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

type errorEnvelope struct {
	OK      bool      `json:"ok"`
	Command string    `json:"command"`
	Error   errorInfo `json:"error"`
}

func (a *app) fail(command string, code int, err error) int {
	if code == 0 {
		code = 1
	}
	if a.json {
		a.writeJSON(errorEnvelope{
			OK:      false,
			Command: command,
			Error:   errorInfo{Code: errorCode(code), Message: err.Error()},
		})
	}
	if !errors.Is(err, flag.ErrHelp) {
		fmt.Fprintf(a.io.Err, "fontcreator233 %s: %v\n", command, err)
	}
	return code
}

func errorCode(code int) string {
	switch code {
	case 1:
		return "user_error"
	case 130:
		return "canceled"
	default:
		return "internal_error"
	}
}
