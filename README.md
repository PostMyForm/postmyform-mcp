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

For a temporary shell session, read the token without echoing it or placing
the value in shell history:

```bash
read -rsp "PostMyForm API token: " POSTMYFORM_API_TOKEN
echo
export POSTMYFORM_API_TOKEN
```

The credential is not an MCP tool argument.

After the MCP client session is complete, remove the token from the shell
environment:

```bash
unset POSTMYFORM_API_TOKEN
```

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

## Installation

PostMyForm MCP is distributed as a standalone native executable.

Supported release targets:

- Linux amd64.
- Linux arm64.
- macOS amd64.
- macOS arm64.
- Windows amd64.

Each release also includes:

- `SHA256SUMS`;
- one CycloneDX JSON SBOM for each target;
- `LICENSE`;
- `THIRD_PARTY_LICENSES.txt`;
- GitHub artifact attestations for the files listed in `SHA256SUMS`.

The normal installation path does not require Go, Node.js, Python, the
PostMyForm CLI, or the private PostMyForm application repository.

### Linux

Example for amd64:

```bash
gh release download \
  --repo PostMyForm/postmyform-mcp \
  --pattern postmyform-mcp-linux-amd64 \
  --pattern SHA256SUMS

sha256sum -c --ignore-missing SHA256SUMS

sudo install -m 0755 \
  postmyform-mcp-linux-amd64 \
  /usr/local/bin/postmyform-mcp
```

For Linux arm64, use `postmyform-mcp-linux-arm64`.

Verify the installed version:

```bash
postmyform-mcp --version
```

### macOS

Download the correct macOS artifact and `SHA256SUMS` from the GitHub Release.

Use:

- `postmyform-mcp-darwin-amd64` for Intel macOS;
- `postmyform-mcp-darwin-arm64` for Apple silicon.

Verify the checksum, then install the binary:

```bash
sudo install -m 0755 \
  postmyform-mcp-darwin-arm64 \
  /usr/local/bin/postmyform-mcp
```

Verify the installed version:

```bash
postmyform-mcp --version
```

Do not disable macOS security controls to run PostMyForm MCP.

### Windows

Download:

```text
postmyform-mcp-windows-amd64.exe
SHA256SUMS
```

from the GitHub Release.

Verify the SHA-256 digest against the matching entry in `SHA256SUMS`.

Rename the executable to `postmyform-mcp.exe` if desired and place it in a
directory included in your `PATH`.

Verify the installed version:

```powershell
postmyform-mcp.exe --version
```

Do not disable Windows security controls to run PostMyForm MCP.

## Release verification

Verify the downloaded artifact against `SHA256SUMS` before installation.

If GitHub CLI is installed, also verify release provenance:

```bash
gh attestation verify postmyform-mcp-linux-amd64 \
  --repo PostMyForm/postmyform-mcp
```

Replace the file name with the artifact for your platform.

A valid attestation verifies that the artifact was produced by the
PostMyForm MCP GitHub Actions release workflow.

## Version

Show the installed server version with either command:

```bash
postmyform-mcp version
postmyform-mcp --version
```

Release versions use stable semantic-version tags such as `v0.1.0`.

The reported version must match the installed release artifact.

PostMyForm MCP supports PostMyForm public API version `0.6.0`, as documented
by the pinned `api/openapi.json` snapshot.

The MVP is built and tested with the Go Model Context Protocol SDK version
`v1.7.0` and uses local stdio transport.

## Upgrade

Check the installed version:

```bash
postmyform-mcp --version
```

Check the latest published release:

```bash
gh release view \
  --repo PostMyForm/postmyform-mcp \
  --json tagName \
  --jq .tagName
```

To upgrade, download the newer artifact for your platform, verify it, and
replace the installed executable using the same installation method.

PostMyForm MCP does not update itself automatically.

## Uninstall

On Linux or macOS, if you installed the server in `/usr/local/bin`:

```bash
sudo rm /usr/local/bin/postmyform-mcp
```

On Windows, delete `postmyform-mcp.exe` from its installation directory.

If you added a directory to `PATH` only for PostMyForm MCP, remove that
directory from `PATH` after deleting the executable.

## Build from source

Building from source is optional and intended for development.

Requirements:

- Go 1.27.1 or later in the Go 1.27 release line.

Build:

```bash
go build -trimpath -buildvcs=false -o postmyform-mcp ./cmd/postmyform-mcp
```


## Agent configuration

PostMyForm MCP is a local stdio server.

A compatible MCP client must:

1. start the local `postmyform-mcp` executable;
2. communicate with it over stdin and stdout;
3. provide `POSTMYFORM_API_TOKEN` in the child-process environment;
4. optionally provide `POSTMYFORM_API_BASE_URL` for an approved alternate
   environment.

The API token belongs to the MCP server process. It is not an MCP tool
argument, and the model does not need the credential value.

Use an absolute executable path when the MCP client requires one.

Do not:

- put the bearer token in command arguments;
- commit the token to source control;
- store the token in public dotfiles;
- let the model choose or modify the API base URL;
- disable TLS verification;
- expose the server through a generic HTTP proxy;
- run model-provided shell commands to configure the server.

For WSL, install and run the Linux binary inside WSL and use a path that is
visible to the MCP client running in that WSL environment.

The MCP server does not need access to your website source code or to the
private PostMyForm application repository.


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

For an approved alternate API environment, the same file can also contain:

```text
POSTMYFORM_API_BASE_URL=https://example.invalid/api/v1
```

Keep this file private. Do not commit it to source control.

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
      POSTMYFORM_API_BASE_URL: "${POSTMYFORM_API_BASE_URL}"
    enabled: true
    supports_parallel_tool_calls: false
```

`POSTMYFORM_API_BASE_URL` is optional. Omit it to use the production API.

Reload MCP servers after changing the configuration:

```text
/reload-mcp
```

Hermes prefixes imported MCP tool names with the configured server name.


## Codex CLI

Codex supports local stdio MCP servers directly.

Provide `POSTMYFORM_API_TOKEN` in the environment that starts Codex. Use the
safe temporary-shell method described in the Security model section instead of
placing the token value in a command argument or shell-history entry.

Add this to:

```text
~/.codex/config.toml
```

Example:

```toml
[mcp_servers.postmyform]
command = "/absolute/path/to/postmyform-mcp"
env_vars = ["POSTMYFORM_API_TOKEN", "POSTMYFORM_API_BASE_URL"]
required = true
default_tools_approval_mode = "writes"
```

`POSTMYFORM_API_BASE_URL` is optional. If it is not present in the parent
environment, PostMyForm MCP uses the production API.

Codex can also store explicit MCP environment values in its local
configuration. The example above uses inherited environment-variable names so
the API token does not need to be written into `config.toml`.

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

Provide `POSTMYFORM_API_TOKEN` in the environment that starts Claude Code. Use
the safe temporary-shell method described in the Security model section.

A project-scoped `.mcp.json` can reference environment variables without
storing the token value:

```json
{
  "mcpServers": {
    "postmyform": {
      "command": "/absolute/path/to/postmyform-mcp",
      "args": [],
      "env": {
        "POSTMYFORM_API_TOKEN": "${POSTMYFORM_API_TOKEN}",
        "POSTMYFORM_API_BASE_URL": "${POSTMYFORM_API_BASE_URL}"
      }
    }
  }
}
```

`POSTMYFORM_API_BASE_URL` is optional. Omit it from the configuration to use
the production API.

Project-scoped MCP configuration requires user approval in normal interactive
Claude Code sessions.

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

## Release process

Stable releases use tags in the form `vMAJOR.MINOR.PATCH`.

A release tag must identify a reviewed commit that is reachable from the
repository's `main` branch.

The release workflow fails closed. Before publishing artifacts, it runs:

- Go formatting verification;
- `go vet`;
- the complete test suite;
- the race-enabled test suite;
- pinned OpenAPI contract verification;
- compatibility comparison with the published PostMyForm API contract;
- generated-model regeneration and diff verification;
- `govulncheck`;
- native builds for all supported release targets;
- a second build of each target and byte-for-byte comparison;
- embedded release-version verification.

Release binaries are built with:

```text
CGO_ENABLED=0
-trimpath
-buildvcs=false
```

The release version is embedded at build time and must match the Git tag.

For each supported target, the workflow generates a CycloneDX JSON SBOM. It
also includes the project license, bundled third-party license texts, and a
`SHA256SUMS` file.

GitHub artifact attestations provide release provenance. The MVP does not use
a separate binary-signing key. Consumers can verify both the SHA-256 digest
and the GitHub attestation before installation.

A failed quality, security, contract, reproducibility, version, SBOM, or
provenance step prevents release publication.

PostMyForm MCP has no silent or automatic update mechanism.


## License

Apache License 2.0.
