package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const testToken = "pmf_stdio_secret_777777"

type requestRecord struct {
	method        string
	path          string
	authorization string
}

func TestServerStartsAndServesToolsOverStdio(t *testing.T) {
	t.Parallel()

	requests := make(chan requestRecord, 1)

	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests <- requestRecord{
			method:        r.Method,
			path:          r.URL.Path,
			authorization: r.Header.Get("Authorization"),
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	defer apiServer.Close()

	repoRoot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("filepath.Abs() error = %v", err)
	}

	command := exec.Command("go", "run", "./cmd/postmyform-mcp")
	command.Dir = repoRoot
	command.Env = append(
		withoutEnv(os.Environ(), "POSTMYFORM_API_TOKEN", "POSTMYFORM_API_BASE_URL"),
		"POSTMYFORM_API_TOKEN="+testToken,
		"POSTMYFORM_API_BASE_URL="+apiServer.URL,
	)

	for _, arg := range command.Args {
		if strings.Contains(arg, testToken) {
			t.Fatal("credential appeared in child-process arguments")
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client := mcp.NewClient(
		&mcp.Implementation{
			Name:    "postmyform-mcp-stdio-test",
			Version: "test",
		},
		nil,
	)

	session, err := client.Connect(
		ctx,
		&mcp.CommandTransport{
			Command:           command,
			TerminateDuration: 2 * time.Second,
		},
		nil,
	)
	if err != nil {
		t.Fatalf("client.Connect() error = %v", err)
	}
	defer session.Close()

	tools, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("ListTools() error = %v", err)
	}
	if len(tools.Tools) != 7 {
		t.Fatalf("tool count = %d, want 7", len(tools.Tools))
	}

	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "list_forms",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatalf("CallTool(list_forms) protocol error = %v", err)
	}
	if result.IsError {
		t.Fatalf("CallTool(list_forms) returned tool error: %v", result.Content)
	}

	encodedResult, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("json.Marshal(result) error = %v", err)
	}
	if bytes.Contains(encodedResult, []byte(testToken)) {
		t.Fatal("credential appeared in MCP result")
	}

	select {
	case request := <-requests:
		if request.method != http.MethodGet {
			t.Fatalf("API method = %s, want GET", request.method)
		}
		if request.path != "/forms" {
			t.Fatalf("API path = %q, want /forms", request.path)
		}
		if request.authorization != "Bearer "+testToken {
			t.Fatalf("API Authorization header was not the process-owned credential")
		}
	case <-ctx.Done():
		t.Fatal("fixture API did not receive list_forms request")
	}
}

func withoutEnv(env []string, names ...string) []string {
	result := make([]string, 0, len(env))

	for _, entry := range env {
		keep := true
		for _, name := range names {
			if strings.HasPrefix(entry, name+"=") {
				keep = false
				break
			}
		}
		if keep {
			result = append(result, entry)
		}
	}

	return result
}

func TestServerFailsClosedWithoutCredential(t *testing.T) {
	t.Parallel()

	repoRoot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("filepath.Abs() error = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	command := exec.CommandContext(ctx, "go", "run", "./cmd/postmyform-mcp")
	command.Dir = repoRoot
	command.Env = withoutEnv(
		os.Environ(),
		"POSTMYFORM_API_TOKEN",
		"POSTMYFORM_API_BASE_URL",
	)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr

	err = command.Run()
	if err == nil {
		t.Fatal("server started without POSTMYFORM_API_TOKEN")
	}

	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}

	const want = "postmyform-mcp: POSTMYFORM_API_TOKEN is required\n"
	if !strings.HasPrefix(stderr.String(), want) {
		t.Fatalf("stderr = %q, want prefix %q", stderr.String(), want)
	}
}
