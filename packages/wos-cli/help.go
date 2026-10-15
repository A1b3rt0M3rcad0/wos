package woscli

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// CommandCatalog is a local discovery contract; it never grants authority.
type CommandCatalog struct {
	SchemaVersion int            `json:"schema_version"`
	Version       string         `json:"version"`
	Commit        string         `json:"commit"`
	Groups        []CommandGroup `json:"groups"`
	Notes         []string       `json:"notes"`
}

type CommandGroup struct {
	Name     string   `json:"name"`
	Summary  string   `json:"summary"`
	Commands []string `json:"commands"`
}

func localCatalog(service bool) CommandCatalog {
	groups := []CommandGroup{
		{"help", "Local help and versioned CLI discovery; no credentials required", []string{"wos --help", "wos <group> --help", "wos help <group>", "wos catalog --output json"}},
		{"version", "Product identity", []string{"wos version --output json"}},
		{"init", "Initialize legacy schema 1; schema 2 uses project init", []string{"wos init --server URL --namespace ID --outcome ID --credential-env NAME"}},
		{"project", "Initialize an explicit signed schema-2 workspace", []string{"wos project init --server URL --server-id ID --namespace ID --outcome ID"}},
		{"profile", "Named schema-2 identity and protected credential references", []string{"wos profile create NAME --file FILE --server URL --server-id ID --issuer-fingerprint SHA256 --enrollment ID", "wos profile list", "wos profile inspect NAME", "wos profile trust NAME --issuer-fingerprint SHA256"}},
		{"auth", "Authenticate the selected destination and Principal", []string{"wos --profile NAME auth status --output json"}},
		{"capabilities", "Read supported server contracts", []string{"wos --profile NAME capabilities --output json"}},
		{"doctor", "Schema-2 destination, issuer, local key and pending-intent diagnostics", []string{"wos --profile NAME doctor --output json"}},
		{"work", "Execution; use the explicitly selected workspace protocol", []string{"wos --profile NAME work checkout --next", "wos --profile NAME work show ID --for-agent", "wos --profile NAME work sign ID", "wos --profile NAME work send ID", "wos --profile NAME work finish ID", "wos --profile NAME work recover --all-pending", "wos --profile NAME work resolution-status ID", "wos --profile NAME work resolve ID --reason REASON", "wos --profile NAME work renew ID", "wos --profile NAME work resume ID --takeover", "wos --profile NAME work keepalive ID --foreground"}},
		{"review", "Independent signed review; a different profile alone is not a different Principal", []string{"wos --profile NAME review checkout --next", "wos --profile NAME review show ID --for-agent", "wos --profile NAME review sign ID", "wos --profile NAME review send ID", "wos --profile NAME review finish ID", "wos --profile NAME review recover --all-pending", "wos --profile NAME review resolution-status ID", "wos --profile NAME review resolve ID --reason REASON"}},
		{"protocol", "Explicit Namespace protocol selection; activation requires an authorized operator", []string{"wos --profile NAME protocol get", "wos --profile NAME protocol preflight", "wos --profile NAME protocol set --phase signed_contracts_v2 --version VERSION --writers-drained --reason REASON", "wos --profile NAME protocol recover"}},
		{"outcome", "Legacy schema-1 Outcome queries and creation; context is not a CLI verb", []string{"wos outcome get [ID] --output json", "wos outcome list --output json", "wos outcome create --file FILE"}},
		{"objective", "Legacy schema-1 typed planning commands", []string{"wos objective get ID", "wos objective list", "wos objective create --file FILE"}},
		{"roadmap", "Legacy schema-1 typed roadmap commands", []string{"wos roadmap get ID", "wos roadmap create --file FILE", "wos roadmap publish --file FILE", "wos roadmap activate --file FILE"}},
		{"contract", "Legacy schema-1 contract queries; signed contracts use work/review show", []string{"wos contract list", "wos contract get ID", "wos contract history ID", "wos contract revoke ID --version VERSION --reason REASON"}},
		{"workspace", "Explicit migration; preserve pending intentions", []string{"wos workspace migrate --to 2 --profile NAME --dry-run"}},
	}
	if service {
		groups = append(groups,
			CommandGroup{"server", "Standalone WOS service", []string{"wos server"}},
			CommandGroup{"skill", "Install bundled agent skills through the primary wos executable", []string{"wos skill install --agent codex", "wos skill update --agent codex", "wos skill list", "wos skill doctor --agent codex", "wos skill uninstall --agent codex"}},
			CommandGroup{"config", "Validate local service configuration", []string{"wos config validate"}},
			CommandGroup{"db", "Service-side database operations; restore targets a new SQLite file", []string{"wos db migrate", "wos db backup FILE", "wos db restore SOURCE DESTINATION"}},
			CommandGroup{"issuer", "Explicit service-side issuer recovery", []string{"wos issuer recover --file FILE"}},
			CommandGroup{"mcp", "Local-authentication MCP stdio service", []string{"wos mcp stdio"}},
		)
	}
	return CommandCatalog{1, Version, Commit, groups, []string{
		"Install: npm install -g wosctl@latest; public operations use wos <command>.",
		"Select --workspace PATH and --profile NAME explicitly for signed schema-2 calls.",
		"init, project init and skill installation never activate signed_contracts_v2.",
		"Preserve the original payload, idempotency key, expected versions and receipts on timeout.",
		"Evidence and evidence links do not evaluate criteria or achieve Outcomes.",
		"Detailed flags and legacy operations: docs/signed-workspace-profiles.md and packages/wos-cli/README.md.",
	}}
}

// HandleHelp handles only explicit local discovery. It runs before workspace or
// credential access, including when global profile flags precede --help.
func HandleHelp(args []string, stdout, stderr io.Writer, service bool) (bool, int) {
	help, catalog := false, false
	filtered := make([]string, 0, len(args))
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			help = true
		} else {
			filtered = append(filtered, arg)
		}
	}
	o, err := parseOptions(filtered)
	if err == nil && len(o.args) > 0 && (o.args[0] == "help" || o.args[0] == "catalog") {
		help, catalog = true, o.args[0] == "catalog"
		o.args = o.args[1:]
	}
	if !help {
		return false, 0
	}
	if err != nil {
		fmt.Fprintln(stderr, err)
		return true, 2
	}
	value := localCatalog(service)
	if catalog && len(o.args) != 0 {
		fmt.Fprintln(stderr, "use wos catalog --output json; use wos help <group> for group help")
		return true, 2
	}
	if len(o.args) > 0 {
		found := false
		for _, group := range value.Groups {
			if group.Name == o.args[0] {
				value.Groups, found = []CommandGroup{group}, true
				break
			}
		}
		if !found {
			fmt.Fprintln(stderr, "unknown help group; use wos --help or wos catalog --output json")
			return true, 2
		}
	}
	if o.values["output"] == "json" {
		err = json.NewEncoder(stdout).Encode(Output{Operation: "catalog", Data: value})
	} else {
		var text strings.Builder
		fmt.Fprintf(&text, "WOS %s (CLI catalog schema %d)\nUsage: wos <command> [flags]\n\n", value.Version, value.SchemaVersion)
		for _, group := range value.Groups {
			fmt.Fprintf(&text, "%s — %s\n", group.Name, group.Summary)
			for _, command := range group.Commands {
				fmt.Fprintf(&text, "  %s\n", command)
			}
		}
		fmt.Fprintln(&text)
		for _, note := range value.Notes {
			fmt.Fprintln(&text, note)
		}
		_, err = io.WriteString(stdout, text.String())
	}
	if err != nil {
		fmt.Fprintln(stderr, "cannot write help:", err)
		return true, 6
	}
	return true, 0
}
