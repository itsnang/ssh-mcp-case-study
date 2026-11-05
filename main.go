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

// RunInput is what the agent supplies to run_command.
type RunInput struct {
	Command        string `json:"command" jsonschema:"the command to run, e.g. uptime or df -h"`
	TimeoutSeconds int    `json:"timeoutSeconds,omitempty" jsonschema:"seconds before the command is killed; defaults to 15"`
}

const maxTimeout = 120 * time.Second

var lab = Host{
	Addr:        "127.0.0.1:2222",
	User:        "agent",
	KeyPath:     `lab\keys\agent_key`,
	Fingerprint: "SHA256:XrPKA+qQ+DY1WncJtZLtZ6jxx00WVEiFBt2nEv5RYKQ",
}

func ping(ctx context.Context, req *mcp.CallToolRequest, in PingInput) (*mcp.CallToolResult, PingOutput, error) {
	return nil, PingOutput{Message: "pong at " + time.Now().Format(time.RFC3339)}, nil
}

func add(ctx context.Context, req *mcp.CallToolRequest, in AddInput) (*mcp.CallToolResult, AddOutput, error) {
	return nil, AddOutput{Sum: in.A + in.B}, nil
}

func runCommandTool(ctx context.Context, req *mcp.CallToolRequest, in RunInput) (*mcp.CallToolResult, Result, error) {
	timeout := 15 * time.Second
	if in.TimeoutSeconds > 0 {
		timeout = time.Duration(in.TimeoutSeconds) * time.Second
	}
	if timeout > maxTimeout {
		timeout = maxTimeout
	}

	res, err := runCommand(lab, in.Command, timeout)
	if err != nil {
		return nil, Result{}, err
	}
	return nil, res, nil
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

	mcp.AddTool(server, &mcp.Tool{
		Name: "run_command",
		Description: "Run a command on the lab Linux server over SSH. Returns stdout, stderr and exitCode. " +
			"Commands are non-interactive; anything still running at the timeout is killed.",
	}, runCommandTool)

	log.Println("server ready") // stderr, not stdout
	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Fatal(err)
	}
}
