package main

import (
	"context"
	"fmt"
	"os"

	"github.com/PostMyForm/postmyform-mcp/internal/api"
	"github.com/PostMyForm/postmyform-mcp/internal/mcpserver"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var version = "dev"

func main() {
	if len(os.Args) == 2 {
		switch os.Args[1] {
		case "version", "--version":
			fmt.Println(version)
			return
		}
	}

	client, err := api.NewClient(
		os.Getenv("POSTMYFORM_API_BASE_URL"),
		os.Getenv("POSTMYFORM_API_TOKEN"),
		0,
	)
	if err != nil {
		fmt.Fprintln(os.Stderr, "postmyform-mcp:", err)
		os.Exit(1)
	}

	server := mcpserver.New(version, client)

	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		fmt.Fprintln(os.Stderr, "postmyform-mcp:", err)
		os.Exit(1)
	}
}
