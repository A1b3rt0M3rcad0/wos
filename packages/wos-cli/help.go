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
		{"init", "Initialize a private, CWD-independent installation", []string{"wos init", "wos init --from PATH --writers-drained --dry-run", "wos init --from PATH --writers-drained"}},
		{"project", "Legacy explicit-workspace initialization; select --workspace PATH", []string{"wos --workspace PATH project init --server URL --server-id ID --namespace ID --outcome ID"}},
		{"profile", "Isolated destinations and protected identities; selection is pinned per invocation", []string{"wos profile create NAME --server URL --server-id ID --namespace ID --issuer-fingerprint SHA256 --credential-ref env:TOKEN --private-key-ref env:KEY --enrollment ID", "wos profile create NAME --server URL --server-id ID --namespace ID --issuer-fingerprint SHA256 --token-stdin --generate-signing-key --enrollment ID", "wos profile list", "wos profile show NAME", "wos profile use NAME", "wos profile remove NAME", "wos --profile NAME profile recover", "wos --profile NAME profile trust --issuer-fingerprint SHA256"}},
		{"auth", "Authenticate the selected destination and Principal", []string{"wos --profile NAME auth status --output json"}},
		{"capabilities", "Read supported server contracts", []string{"wos --profile NAME capabilities --output json"}},
		{"inspect", "Read coherent workspace context without acquiring authority", []string{"wos --profile NAME --outcome OUTCOME_ID inspect get --file QUERY.json", "wos --profile NAME --outcome OUTCOME_ID inspect cards --file QUERY.json", "wos --profile NAME --outcome OUTCOME_ID inspect timeline --file QUERY.json", "wos --profile NAME --outcome OUTCOME_ID inspect note NOTE_ID"}},
		{"planning", "Typed planning/documentary intentions with durable exact replay; never execution or signed review", []string{"wos --profile NAME --outcome OUTCOME_ID planning apply --command append_entity_note --file COMMAND.json", "wos --profile NAME --outcome OUTCOME_ID planning apply --command create_plan_items --file COMMAND.json", "wos --profile NAME --outcome OUTCOME_ID planning recover"}},
		{"doctor", "Schema-2 destination, issuer, local key and pending-intent diagnostics", []string{"wos --profile NAME doctor --output json"}},
		{"work", "Acquire Tasks, inspect bounded context, record durable material, and deliver", []string{"wos work available --limit 25", "wos work next", "wos work checkout TASK_REF", "wos work list", "wos work context [CONTRACT_REF]", "wos work status [CONTRACT_REF]", "wos work history --work TASK_ID --outcome OUTCOME_ID --order desc", "wos work evidence list", "wos work artifact list", "wos work log add --type progress --description TEXT", "wos work evidence add --description TEXT --uri URI --criterion CRITERION_REF", "wos work artifact add --name NAME --uri URI --artifact-type TYPE", "wos work progress set --summary TEXT --next-action TEXT", "wos work result set --summary TEXT --completion auto", "wos work correction add --finding FINDING_ID --summary TEXT", "wos work check [CONTRACT_REF]", "wos work finish [CONTRACT_REF]", "wos work recover --all-pending", "wos work resolution-status CONTRACT_ID", "wos work resolve CONTRACT_ID --reason REASON", "wos work renew CONTRACT_REF", "wos work resume CONTRACT_REF", "wos work keepalive --all-active --foreground"}},
		{"review", "Independent review; another profile alone is not another Principal", []string{"wos review next", "wos review checkout CASE_ID --version VERSION", "wos review list", "wos review context [CONTRACT_REF]", "wos review status [CONTRACT_REF]", "wos review evidence add --description TEXT --uri URI --criterion CRITERION_REF", "wos review decision set --decision approved --reason TEXT", "wos review assessment add --criterion CRITERION_REF --result met --rationale TEXT", "wos review assessment list", "wos review assessment set --criterion CRITERION_REF --result met --rationale TEXT", "wos review assessment remove --criterion CRITERION_REF", "wos review decision set --decision changes_requested --reason TEXT", "wos review finding add --criterion CRITERION_REF --description TEXT", "wos review finding add --requirement ORIGINAL_REQUIREMENT_REF --description TEXT", "wos review check [CONTRACT_REF]", "wos review finish [CONTRACT_REF]", "wos review recover --all-pending", "wos review resolution-status CONTRACT_ID", "wos review resolve CONTRACT_ID --reason REASON"}},
		{"protocol", "Explicit Namespace protocol selection; activation requires an authorized operator", []string{"wos --profile NAME protocol get", "wos --profile NAME protocol preflight", "wos --profile NAME protocol set --phase signed_contracts_v2 --version VERSION --writers-drained --reason REASON", "wos --profile NAME protocol recover"}},
		{"outcome", "Current scoped Outcome discovery; planning apply exposes typed creation without execution authority", []string{"wos outcome get [REF] --output json", "wos outcome list --output json", "wos --workspace PATH outcome create --file FILE"}},
		{"objective", "Legacy schema-1 typed planning commands", []string{"wos --workspace PATH objective get ID", "wos --workspace PATH objective list", "wos --workspace PATH objective create --file FILE"}},
		{"roadmap", "Legacy schema-1 typed roadmap commands", []string{"wos --workspace PATH roadmap get ID", "wos --workspace PATH roadmap create --file FILE", "wos --workspace PATH roadmap publish --file FILE", "wos --workspace PATH roadmap activate --file FILE"}},
		{"contract", "Legacy schema-1 contract queries; signed contracts use work/review show", []string{"wos --workspace PATH contract list", "wos --workspace PATH contract get ID", "wos --workspace PATH contract history --work WORK_ID", "wos --workspace PATH contract revoke ID --version VERSION --reason REASON"}},
		{"workspace", "Explicit migration; preserve pending intentions", []string{"wos --workspace PATH workspace migrate --to 2 --profile NAME --dry-run"}},
	}
	if service {
		groups = append(groups,
			CommandGroup{"server", "Standalone WOS service", []string{"wos server"}},
			CommandGroup{"skill", "Install bundled agent skills through the primary wos executable", []string{"wos skill install --agent codex", "wos skill update --agent codex", "wos skill list", "wos skill doctor --agent codex", "wos skill uninstall --agent codex"}},
			CommandGroup{"config", "Validate local service configuration", []string{"wos config validate", "wos config validate --deployment=shared"}},
			CommandGroup{"db", "Service-side database operations; restore targets a new SQLite file", []string{"wos db migrate", "wos db backup FILE", "wos db restore SOURCE DESTINATION"}},
			CommandGroup{"issuer", "Explicit service-side issuer recovery", []string{"wos issuer recover --file FILE"}},
			CommandGroup{"mcp", "Local-authentication MCP stdio service", []string{"wos mcp stdio"}},
		)
	}
	return CommandCatalog{1, Version, Commit, groups, []string{
		"This catalog describes the current binary; compare wos version and server capabilities when using a published package.",
		"Profile selection: --profile, WOS_PROFILE, persistent active profile; one invocation stays on its original destination.",
		"WOS_DATA_DIR overrides the private installation root; --workspace PATH explicitly selects legacy compatibility.",
		"init, profile creation and skill installation never activate signed_contracts_v2 or grant authority.",
		"Omit a contract reference only when exactly one local contract matches; ambiguity fails without mutation.",
		"Use typed commands for material; never edit internal operational documents.",
		"Execution log/evidence/artifact additions are durable; result/progress and review material are local preparation until return.",
		"finish reports delivery disposition separately from Task completion and local cleanup.",
		"Preserve the original payload, idempotency key, expected versions and receipts on timeout.",
		"Evidence and evidence links do not evaluate criteria or achieve Outcomes.",
		"Set an approved review decision before adding its criterion assessments; terminal resolution requires an exact contract ID.",
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
