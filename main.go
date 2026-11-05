package main

import (
	"context"
	"log"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// PingInput is empty: the ping tool takes no arguments.
type PingInput struct{}

// PingOutput is what the agent receives back.
type PingOutput struct {
	Message string `json:"message"`
}

// AddInput is the homework tool: two numbers to add.
type AddInput struct {
	A float64 `json:"a" jsonschema:"the first number"`
	B float64 `json:"b" jsonschema:"the second number"`
}

// AddOutput carries the result.
type AddOutput struct {
	Sum float64 `json:"sum" jsonschema:"the sum of a and b"`
}

func ping(ctx context.Context, req *mcp.CallToolRequest, in PingInput) (*mcp.CallToolResult, PingOutput, error) {
	return nil, PingOutput{Message: "pong at " + time.Now().Format(time.RFC3339)}, nil
}

func add(ctx context.Context, req *mcp.CallToolRequest, in AddInput) (*mcp.CallToolResult, AddOutput, error) {
	return nil, AddOutput{Sum: in.A + in.B}, nil
}

func main() {
	server := mcp.NewServer(&mcp.Implementation{Name: "ssh-agent-mcp", Version: "0.1.0"}, nil)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "ping",
		Description: "Check that the server is running.",
	}, ping)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "add",
		Description: "Add two numbers and return the sum.",
	}, add)

	log.Println("server ready") // stderr, not stdout
	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Fatal(err)
	}
}
