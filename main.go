package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) == 2 && (os.Args[1] == "version" || os.Args[1] == "--version") {
		fmt.Println("fontcreator233 development")
		return
	}
	fmt.Fprintln(os.Stderr, "fontcreator233: CLI not wired yet")
	os.Exit(2)
}
