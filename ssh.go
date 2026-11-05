package main

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"os"
	"time"

	"golang.org/x/crypto/ssh"
)

// Everything we need to know to log into one server.
type Host struct {
	Addr        string // where:  "127.0.0.1:2222"
	User        string // who:    "agent"
	KeyPath     string // key:    lab\keys\agent_key
	Fingerprint string // ID card we expect: "SHA256:..."
}

// What we send back to the AI.
type Result struct {
	ExitCode int    `json:"exitCode" jsonschema:"0 means success"`
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
}

func runCommand(h Host, command string, timeout time.Duration) (Result, error) {
	// STEP 1: pick up the key.
	keyBytes, err := os.ReadFile(h.KeyPath)
	if err != nil {
		return Result{}, fmt.Errorf("read key: %w", err)
	}
	signer, err := ssh.ParsePrivateKey(keyBytes)
	if err != nil {
		return Result{}, fmt.Errorf("parse key: %w", err)
	}

	// STEP 2: settings: who am I, which key, and how to check the server's ID.
	config := &ssh.ClientConfig{
		User:              h.User,
		Auth:              []ssh.AuthMethod{ssh.PublicKeys(signer)},
		HostKeyAlgorithms: []string{ssh.KeyAlgoED25519},
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			got := ssh.FingerprintSHA256(key)
			if got != h.Fingerprint {
				return fmt.Errorf("host key mismatch: expected %s, got %s", h.Fingerprint, got)
			}
			return nil // nil = "OK, I trust you"
		},
		Timeout: 10 * time.Second,
	}

	// STEP 3: knock on the door.
	client, err := ssh.Dial("tcp", h.Addr, config)
	if err != nil {
		return Result{}, fmt.Errorf("connect: %w", err)
	}
	defer client.Close() // "when this function ends, hang up"

	// STEP 4: open a session (one session = one command) and collect the output.
	session, err := client.NewSession()
	if err != nil {
		return Result{}, fmt.Errorf("session: %w", err)
	}
	defer session.Close()

	var stdout, stderr bytes.Buffer
	session.Stdout = &stdout // normal output goes here
	session.Stderr = &stderr // error messages go here

	// STEP 5: run it on another goroutine so we can stop waiting.
	// session.Run blocks until the command exits. Buffered so the goroutine
	// can always finish and exit, even if nobody reads the result.
	done := make(chan error, 1)
	go func() { done <- session.Run(command) }()

	select {
	case err = <-done:
		// finished in time: fall through to interpreting the result
	case <-time.After(timeout):
		// Closing a session alone waits for the remote process, so signal it
		// first, then drop the whole connection.
		_ = session.Signal(ssh.SIGKILL)
		_ = session.Close()
		_ = client.Close()
		return Result{}, fmt.Errorf("command exceeded %s and was killed", timeout)
	}

	// STEP 6: turn what happened into a Result.
	res := Result{Stdout: stdout.String(), Stderr: stderr.String()}
	var exitErr *ssh.ExitError
	if errors.As(err, &exitErr) {
		res.ExitCode = exitErr.ExitStatus() // the command ran but failed (e.g. file not found)
	} else if err != nil {
		return Result{}, fmt.Errorf("run: %w", err) // something else broke (e.g. connection lost)
	}
	return res, nil
}
