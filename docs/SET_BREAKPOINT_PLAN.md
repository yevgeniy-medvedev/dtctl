# POC start

We're gonna implement support for setting a breakpoint using the Live Debugger.
The goal is to have a POC.
The minimal requirements for a POC is to at least:
1. Get the current workspace the user has, by using getOrCreateWorkspace GraphQL (or maybe getOrCreateWorkspaceV2, you need to check). We'll use the clientName parameter and set it with "dtctl".
2. Update the workspace's filters in order to choose the servers we want to debug. This will be done by updateWorkspaceV2 graphql.
3. Set a breakpoint by setting the filename and line number. This will be done with CreateRuleV2 graphql.

So, here are example commands:
if I want to debug all the servers from the k8s.namespace.name=prod I would start with:

dtctl debug --filters k8s.namespace.name=prod

I should be able to set multiple filters seperated with a comma.

And then I should be able to place a breakpoint by:

dtctl debug --breakpoint OrderController.java:306

This will place a breakpoint on file OrderController.java line 306.

You should show the return output of the graphql requests, think about how you should do that.

You can see all the details of the graphql and code usage in the folder devobs-vs-code-plugin where I cloned a repository from a different project that does all these things. Take inspiration and learn from it.

---

# Implementation Progress (2026-03-08)

## Status

POC implementation is complete for the scope defined above.

## Completed

### 1) New `debug` command added

- File: `cmd/debug.go`
- Command shape:
	- `dtctl debug --filters key=value[,key=value...]`
	- `dtctl debug --breakpoint File.java:line`
	- `dtctl get breakpoints`
- Validations implemented:
	- At least one of `--filters` or `--breakpoint` is required.
	- Filter parsing requires valid `key=value` pairs.
	- Breakpoint parsing requires `File.java:line` with positive integer line.

### 2) GraphQL Live Debugger handler implemented

- File: `pkg/resources/livedebugger/livedebugger.go`
- Implemented operations:
	- `GetOrCreateWorkspace(...)` using `getOrCreateUserWorkspaceV2` (clientName=`dtctl`)
	- `UpdateWorkspaceFilters(...)` using `updateWorkspaceV2`
	- `CreateBreakpoint(...)` using `createRuleV2`
	- `DeleteBreakpoint(...)` using `deleteRuleV2`
	- `DeleteAllBreakpoints(...)` using `deleteAllRulesFromWorkspaceV2` with `DumpFrame` rule type
- Additional support implemented:
	- Environment URL -> GraphQL endpoint resolution (`/platform/dob/graphql`)
	- Org ID extraction from environment host
	- Response parsing for workspace ID
	- Mutable rule ID generation

### 2.1) `--filters` payload mapping (important)

- CLI filters are intentionally sent as `labels` in `updateWorkspaceV2` input.
- `filters` is sent as an empty array.

Example command:

`dtctl debug --filters k8s.container.name=credit-card-order-service,dt.kubernetes.workload.name=credit-card-order-service`

Expected GraphQL `filterSets` payload shape:

```json
[
	{
		"labels": [
			{
				"field": "k8s.container.name",
				"values": [
					"credit-card-order-service"
				]
			},
			{
				"field": "dt.kubernetes.workload.name",
				"values": [
					"credit-card-order-service"
				]
			}
		],
		"filters": []
	}
]
```

### 3) Safety checks added for mutating actions

- In `cmd/debug.go`:
	- `OperationUpdate` check before filter update
	- `OperationCreate` check before breakpoint creation

### 4) GraphQL response output wired

- Default mode (no `-v/--verbose`):
	- no raw GraphQL payloads printed on successful operations
	- errors are still returned and shown
- Verbose mode (`-v` / `--verbose`):
	- prints full GraphQL JSON responses for debug operations (`getOrCreateWorkspaceV2`, `updateWorkspaceV2`, `createRuleV2`, `getWorkspaceRules`)

### 5) Unit tests added

- File: `cmd/debug_test.go`
- Covered:
	- `parseFilters` success/error cases
	- `parseBreakpoint` success/error cases
	- `get breakpoints` command registration
	- workspace-rules to table-row extraction (`id`, `filename`, `line number`, `active`)

### 6) Breakpoint listing added

- New command:
	- `dtctl get breakpoints`
- Implemented GraphQL query:
	- `GetWorkspaceRules(...)` in `pkg/resources/livedebugger/livedebugger.go`
- Output behavior:
	- non-verbose: table view (`id`, `filename`, `line number`, `active`)
	- verbose: raw GraphQL output
- `active` is derived from `is_disabled` (`active = !is_disabled`).

### 7) Breakpoint deletion added

- New commands:
	- `dtctl delete breakpoint <id>`
	- `dtctl delete breakpoint File.java:line`
	- `dtctl delete breakpoint --all`
- Behavior:
	- accepts a mutable breakpoint ID directly
	- resolves `File.java:line` by listing workspace breakpoints, then deletes all matches one by one
	- supports confirmation prompts, `-y/--yes`, and `--dry-run`

## Notes / Follow-up (Optional Hardening)

- Potential next iteration:
	- Add integration/e2e tests around the actual GraphQL flow.
	- Add optional filters/sorting for breakpoint listing (if needed beyond POC).

## Requirement Coverage Checklist

| Requirement | Status | Implemented In |
|---|---|---|
| Add `dtctl debug` command | ✅ | `cmd/debug.go` |
| Require at least one of `--filters` / `--breakpoint` | ✅ | `cmd/debug.go` |
| Parse multiple comma-separated filters | ✅ | `parseFilters` in `cmd/debug.go` |
| Parse `File.java:line` breakpoint format | ✅ | `parseBreakpoint` in `cmd/debug.go` |
| List workspace breakpoints (`get breakpoints`) | ✅ | `cmd/debug.go`, `cmd/get.go`, `GetWorkspaceRules` in `pkg/resources/livedebugger/livedebugger.go` |
| Delete breakpoint by ID or file:line | ✅ | `cmd/delete_breakpoints.go`, `DeleteBreakpoint` in `pkg/resources/livedebugger/livedebugger.go` |
| Delete all workspace breakpoints | ✅ | `cmd/delete_breakpoints.go`, `DeleteAllBreakpoints` in `pkg/resources/livedebugger/livedebugger.go` |
| Get/create workspace with clientName `dtctl` | ✅ | `GetOrCreateWorkspace` in `pkg/resources/livedebugger/livedebugger.go` |
| Update workspace filters via GraphQL | ✅ | `UpdateWorkspaceFilters` in `pkg/resources/livedebugger/livedebugger.go` |
| Create breakpoint rule via GraphQL | ✅ | `CreateBreakpoint` in `pkg/resources/livedebugger/livedebugger.go` |
| Quiet by default on successful debug GraphQL operations | ✅ | `cmd/debug.go` |
| Show raw GraphQL payloads only in verbose mode | ✅ | `cmd/debug.go` (`-v` / `--verbose`) |
| Apply safety checks on mutating operations | ✅ | `cmd/debug.go` (`OperationUpdate`, `OperationCreate`) |
| Add unit tests for debug parsing/registration helpers | ✅ | `cmd/debug_test.go` |
