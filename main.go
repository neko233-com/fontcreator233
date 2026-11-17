package main

import (
	"os"

	"github.com/neko233-com/fontcreator233/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:], cli.IO{In: os.Stdin, Out: os.Stdout, Err: os.Stderr}))
}
