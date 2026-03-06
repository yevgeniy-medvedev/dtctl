# dtctl - Dynatrace Platform CLI (v{{.Version}})

## Commands
dtctl uses a verb-noun pattern: `dtctl <verb> <resource> [flags]`

### Core Operations
- `dtctl get <resource>` - List resources
- `dtctl describe <resource> <id>` - Show details
- `dtctl apply -f <file>` - Create/update from YAML (idempotent)
- `dtctl diff -f <file>` - Compare local vs remote
- `dtctl query --query "<DQL>"` - Execute DQL queries
- `dtctl delete <resource> <id>` - Delete resources

### Best Practices
- Always use `--agent` flag for structured JSON output
- Use `dtctl diff` before `dtctl apply` in production
- Use `dtctl apply --dry-run` to preview changes
- Use `dtctl commands --brief -o json` for machine-readable command listing

### Resource Aliases
wf=workflows, dash/db=dashboards, nb=notebooks, bkt=buckets
