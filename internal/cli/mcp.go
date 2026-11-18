package cli

import (
	"context"
	"fmt"

	"github.com/neko233-com/fontcreator233/internal/mcp"
)

func (a *app) cmdMCP(args []string) int {
	if len(args) > 0 {
		return a.fail("mcp", 1, fmt.Errorf("mcp takes no arguments"))
	}
	server := &mcp.Server{}
	if err := server.Serve(context.Background(), a.io.In, a.io.Out); err != nil {
		fmt.Fprintf(a.io.Err, "fontcreator233 mcp: %v\n", err)
		return 2
	}
	return 0
}
