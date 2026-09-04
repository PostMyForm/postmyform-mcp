# PostMyForm MCP

The official Model Context Protocol server for the PostMyForm public API.

PostMyForm MCP lets AI agents manage forms, fields, and generated form snippets through seven narrow MCP tools.

The server uses only the public PostMyForm API. It does not access PostMyForm databases, private application code, internal routes, local files, or a shell.

## Status

This repository is under active MVP development.

The MVP transport is local stdio. Your MCP client starts `postmyform-mcp` as a child process. The server then calls the PostMyForm public API over HTTPS.

A hosted MCP service is not part of the MVP.

## Tools

Read-only tools:

- `list_forms` — list forms visible to the configured API credential.
- `get_form` — get one form by UUID.
- `get_form_fields` — get the complete ordered field configuration.
- `get_form_snippet` — get generated starter HTML as data. The server does not execute it.

State-changing tools:

- `create_form` — create a form.
- `update_form` — update selected form properties.
- `replace_form_fields` — replace the complete ordered field collection.

`update_form` distinguishes an omitted success redirect from an explicit redirect clear.

`replace_form_fields` replaces the complete field collection. It is not an incremental field edit operation.

## Security model

The PostMyForm API credential belongs to the MCP server process.

Set it with:

```bash
export POSTMYFORM_API_TOKEN='your-api-token'
```

The credential is not an MCP tool argument.

The server does not expose:

- API tokens.
- Authorization headers.
- Arbitrary HTTP methods, URLs, headers, or request bodies.
- Shell commands.
- Local file paths or file access.
- Generic proxy tools.

Do not put `POSTMYFORM_API_TOKEN` in a command argument.

Do not commit API credentials to source control.

The server:

- Uses the operating system TLS trust store.
- Requires HTTPS for remote API endpoints.
- Allows plain HTTP only for loopback testing.
- Uses bounded HTTP timeouts and response sizes.
- Treats API responses as untrusted input.
- Sanitizes errors before they cross the MCP boundary.
- Does not automatically retry state-changing operations.

PostMyForm remains authoritative for authentication, authorization, API scopes, organization boundaries, plan limits, concurrency rules, and audit behavior.

## API endpoint

The production API is used by default:

```text
https://postmyform.com/api/v1
```

For an approved alternate environment, set:

```bash
export POSTMYFORM_API_BASE_URL='https://example.invalid/api/v1'
```

Do not let an agent provide or change the API base URL through a tool call.

## Build from source

The MVP is currently built from source during development.

Requirements:

- Go 1.27.1 or later in the Go 1.27 release line.

Build:

```bash
go build -trimpath -buildvcs=false -o postmyform-mcp ./cmd/postmyform-mcp
```

The release process will later publish native binaries for:

- Linux amd64.
- Linux arm64.
- macOS amd64.
- macOS arm64.
- Windows amd64.

## Hermes Agent

Hermes supports local stdio MCP servers directly.

Store the secret in:

```text
~/.hermes/.env
```

Example:

```text
POSTMYFORM_API_TOKEN=your-api-token
```

Then add the server to:

```text
~/.hermes/config.yaml
```

Example:

```yaml
mcp_servers:
  postmyform:
    command: "/absolute/path/to/postmyform-mcp"
    args: []
    env:
      POSTMYFORM_API_TOKEN: "${POSTMYFORM_API_TOKEN}"
    enabled: true
    supports_parallel_tool_calls: false
```

Reload MCP servers after changing the configuration:

```text
/reload-mcp
```

Hermes prefixes imported MCP tool names with the configured server name.

## Codex CLI

Codex supports local stdio MCP servers directly.

Export the token in the environment that starts Codex:

```bash
export POSTMYFORM_API_TOKEN='your-api-token'
```

Add this to:

```text
~/.codex/config.toml
```

Example:

```toml
[mcp_servers.postmyform]
command = "/absolute/path/to/postmyform-mcp"
env_vars = ["POSTMYFORM_API_TOKEN"]
required = true
default_tools_approval_mode = "writes"
```

Check the configured server with:

```bash
codex mcp list
```

Inside the Codex terminal UI, use:

```text
/mcp
```

The `writes` approval mode asks before tools that are not marked read-only.

## Claude Code

Claude Code supports local stdio MCP servers directly.

Export the token before starting Claude Code:

```bash
export POSTMYFORM_API_TOKEN='your-api-token'
```

A project-scoped `.mcp.json` can reference that environment variable without storing the token:

```json
{
  "mcpServers": {
    "postmyform": {
      "command": "/absolute/path/to/postmyform-mcp",
      "args": [],
      "env": {
        "POSTMYFORM_API_TOKEN": "${POSTMYFORM_API_TOKEN}"
      }
    }
  }
}
```

Project-scoped MCP configuration requires user approval in normal interactive Claude Code sessions.

Check MCP status with:

```bash
claude mcp list
```

Inside Claude Code, use:

```text
/mcp
```

Do not commit a real token into `.mcp.json`.

## Other MCP clients

The MVP server uses standard local stdio MCP transport.

A client must be able to:

1. Start a local executable.
2. Communicate with it over MCP stdio.
3. Pass `POSTMYFORM_API_TOKEN` to the child process environment.

Clients that require only remote Streamable HTTP cannot connect directly to the MVP server.

A hosted Streamable HTTP transport can be considered separately after the local stdio MVP.

## Development

Run all tests:

```bash
go test ./...
```

Run static analysis:

```bash
go vet ./...
```

Build all packages:

```bash
go build ./...
```

The test suite covers:

- The seven public API operations.
- Request methods and paths.
- Response content type and shape validation.
- Bounded responses.
- Network and timeout failures.
- Untrusted TLS rejection.
- Credential redaction.
- No automatic mutation retries.
- MCP tool inventory.
- MCP input and output schemas.
- Read-only and state-changing annotations.
- PATCH omission and explicit redirect clear semantics.
- Complete field replacement.
- Rejection of generic HTTP, credential, shell, and file inputs.
- Real child-process stdio startup and MCP discovery.
- Fail-closed startup when `POSTMYFORM_API_TOKEN` is missing.

## Public API contract

The repository contains a pinned snapshot of the published PostMyForm OpenAPI contract:

```text
api/openapi.json
```

Its expected SHA-256 value is recorded in:

```text
api/openapi.sha256
```

Generated API models use `oapi-codegen` with explicit nullable handling.

The MCP server maintains its own small API client. It does not depend on the PostMyForm CLI and does not use a shared MVP SDK.

## License

Apache License 2.0.
