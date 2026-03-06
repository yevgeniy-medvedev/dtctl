# dtctl - Dynatrace Platform CLI (v{{.Version}})

## Available Commands
dtctl uses a verb-noun pattern: `dtctl <verb> <resource> [flags]`

### Core Verbs
- `get` - List resources (workflows, dashboards, slos, settings, ...)
- `describe` - Show resource details
- `apply -f <file>` - Create/update from YAML (idempotent)
- `diff` - Compare local vs remote state
- `query` - Execute DQL queries
- `wait` - Wait for conditions (CI/CD)
- `watch` - Real-time monitoring
- `history` - Version history
- `restore` - Restore previous version
- `delete` - Delete resources

### Quick Reference
- Always use `--agent` flag for structured JSON output
- Use `dtctl commands` for full command catalog
- Use `dtctl apply --dry-run` before applying changes
- Use `dtctl diff` before `dtctl apply` in production

### Resource Aliases
wf=workflows, dash/db=dashboards, nb=notebooks, bkt=buckets

### Getting Help
- `dtctl commands --brief -o json` for machine-readable command listing
- `dtctl <command> --help` for detailed usage
