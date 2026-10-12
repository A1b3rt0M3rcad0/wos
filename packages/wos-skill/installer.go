package wosskill

import (
	"crypto/rand"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const markerFile = ".wos-skill.json"
const packageName = "@a1b3rt0m3rcad0/wos-skill"

var skillNames = []string{
	"wos-coordination", "wos-delegation", "wos-continuity", "wos-review", "wos-workspace",
}

// The Go service embeds the same source used by the standalone npm skill installer.
//
//go:embed skills package.json
var assets embed.FS

type Options struct {
	Agent   string
	Project string
	Dir     string
	Global  bool
	DryRun  bool
}

type Result struct {
	Target  string `json:"target"`
	Name    string `json:"name"`
	Version string `json:"version"`
	Action  string `json:"action"`
	DryRun  bool   `json:"dryRun"`
}

type manifest struct {
	Schema  int               `json:"schema"`
	Package string            `json:"package"`
	Name    string            `json:"name"`
	Version string            `json:"version"`
	Files   map[string]string `json:"files"`
}

type planned struct {
	result  Result
	prior   *manifest
	files   map[string][]byte
	digests map[string]string
}

func Version() (string, error) {
	var data struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	}
	b, err := assets.ReadFile("package.json")
	if err != nil {
		return "", err
	}
	if err := json.Unmarshal(b, &data); err != nil {
		return "", err
	}
	if data.Name != packageName || data.Version == "" {
		return "", errors.New("invalid embedded WOS skill metadata")
	}
	return data.Version, nil
}

const help = `Usage: wos skill <install|update|uninstall|list|doctor|version> [options]
  --agent codex|claude|hermes|openclaw|generic|all  (default: codex)
  --project PATH    Project root (default: current directory)
  --global          Use native user discovery directories
  --dir PATH        Explicit skills directory (required for generic)
  --dry-run         Preview all changes without writing
  --help            Show this help
The skill bundles are embedded in the WOS Go binary. No Node.js, npm,
network connection, WOS server or API credential is needed for installation.
Installing a skill does not connect MCP or grant agent authority.`

// Run provides the native `wos skill ...` subcommands without starting the server.
func Run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || (len(args) == 1 && (args[0] == "--help" || args[0] == "-h" || args[0] == "help")) {
		fmt.Fprintln(stdout, help)
		return 0
	}
	if len(args) == 1 && args[0] == "version" {
		v, err := Version()
		if err != nil {
			fmt.Fprintln(stderr, "wos skill:", err)
			return 1
		}
		fmt.Fprintln(stdout, v)
		return 0
	}
	command := args[0]
	set := flag.NewFlagSet("wos skill "+command, flag.ContinueOnError)
	set.SetOutput(stderr)
	o := Options{}
	set.StringVar(&o.Agent, "agent", "codex", "agent runtime")
	set.StringVar(&o.Project, "project", "", "project root")
	set.StringVar(&o.Dir, "dir", "", "explicit skills directory")
	set.BoolVar(&o.Global, "global", false, "install for current user")
	set.BoolVar(&o.DryRun, "dry-run", false, "show actions without writing")
	if err := set.Parse(args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Fprintln(stdout, help)
			return 0
		}
		return 2
	}
	if len(set.Args()) != 0 {
		fmt.Fprintln(stderr, "wos skill: unexpected positional arguments")
		return 2
	}
	if command != "install" && command != "update" && command != "uninstall" && command != "list" && command != "doctor" {
		fmt.Fprintln(stderr, "wos skill: unknown command:", command)
		fmt.Fprintln(stderr, help)
		return 2
	}
	results, err := Operate(command, o)
	if err != nil {
		fmt.Fprintln(stderr, "wos skill:", err)
		return 1
	}
	if err := json.NewEncoder(stdout).Encode(results); err != nil {
		fmt.Fprintln(stderr, "wos skill:", err)
		return 1
	}
	return 0
}

// Operate performs preflight on every destination before making any changes.
// Concurrent modification of a skills directory by other processes is unsupported.
func Operate(command string, o Options) ([]Result, error) {
	switch command {
	case "install", "update", "uninstall", "list", "doctor":
	default:
		return nil, fmt.Errorf("unknown skill command: %s", command)
	}
	bases, err := destinations(o)
	if err != nil {
		return nil, err
	}
	version, err := Version()
	if err != nil {
		return nil, err
	}
	plan := make([]planned, 0, len(bases)*len(skillNames))
	for _, base := range bases {
		if err := safeDirectory(base); err != nil {
			return nil, err
		}
		for _, name := range skillNames {
			target := filepath.Join(base, name)
			prior, err := inspect(target, name)
			if err != nil {
				return nil, err
			}
			files, digests, err := bundledFiles(name)
			if err != nil {
				return nil, err
			}
			action := "install"
			switch {
			case command == "uninstall":
				if prior == nil {
					action = "absent"
				} else {
					action = "remove"
				}
			case command == "list" || command == "doctor":
				if prior == nil {
					action = "absent"
				} else {
					action = "installed"
				}
			case prior != nil && prior.Version == version && maps.Equal(prior.Files, digests):
				action = "unchanged"
			case prior != nil:
				action = "update"
			}
			plan = append(plan, planned{result: Result{Target: target, Name: name, Version: version, Action: action, DryRun: o.DryRun}, prior: prior, files: files, digests: digests})
		}
	}
	if !o.DryRun && command != "list" && command != "doctor" {
		for _, item := range plan {
			if item.result.Action == "absent" || item.result.Action == "unchanged" {
				continue
			}
			// Revalidate just before each mutation to avoid overwriting observed edits.
			current, err := inspect(item.result.Target, item.result.Name)
			if err != nil {
				return nil, err
			}
			if !sameManifest(current, item.prior) {
				return nil, fmt.Errorf("destination changed during installation: %s", item.result.Target)
			}
			if item.result.Action == "remove" {
				if err := os.RemoveAll(item.result.Target); err != nil {
					return nil, err
				}
				continue
			}
			if err := installBundle(item, version); err != nil {
				return nil, err
			}
		}
	}
	results := make([]Result, 0, len(plan))
	for _, item := range plan {
		results = append(results, item.result)
	}
	return results, nil
}

func destinations(o Options) ([]string, error) {
	agent := o.Agent
	if agent == "" {
		agent = "codex"
	}
	valid := map[string]bool{"codex": true, "claude": true, "hermes": true, "openclaw": true, "generic": true, "all": true}
	if !valid[agent] {
		return nil, fmt.Errorf("unknown agent: %s", agent)
	}
	if o.Dir != "" && (o.Global || agent == "all") {
		return nil, errors.New("--dir cannot be combined with --global or --agent all")
	}
	if o.Dir != "" {
		path, err := filepath.Abs(o.Dir)
		if err != nil {
			return nil, err
		}
		return []string{path}, nil
	}
	if agent == "generic" {
		return nil, errors.New("generic agent requires --dir")
	}
	base := o.Project
	if o.Global {
		var err error
		base, err = os.UserHomeDir()
		if err != nil {
			return nil, err
		}
	} else if base == "" {
		var err error
		base, err = os.Getwd()
		if err != nil {
			return nil, err
		}
	}
	base, err := filepath.Abs(base)
	if err != nil {
		return nil, err
	}
	paths := map[string]string{
		"codex":    filepath.Join(base, ".agents", "skills"),
		"claude":   filepath.Join(base, ".claude", "skills"),
		"hermes":   filepath.Join(base, ".hermes", "skills"),
		"openclaw": filepath.Join(base, "skills"),
	}
	if o.Global {
		paths["openclaw"] = filepath.Join(base, ".openclaw", "skills")
	}
	if agent != "all" {
		return []string{paths[agent]}, nil
	}
	return []string{paths["claude"], paths["codex"], paths["hermes"], paths["openclaw"]}, nil
}

func safeDirectory(path string) error {
	path, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	for {
		info, err := os.Lstat(path)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		if err == nil && (info.Mode()&os.ModeSymlink != 0 || !info.IsDir()) {
			return fmt.Errorf("unsafe directory (symlink or non-directory): %s", path)
		}
		parent := filepath.Dir(path)
		if parent == path {
			return nil
		}
		path = parent
	}
}

func inspect(root, expectedName string) (*manifest, error) {
	info, err := os.Lstat(root)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("unmanaged or unsafe skill: %s", root)
	}
	if err := safeDirectory(root); err != nil {
		return nil, err
	}
	marker := filepath.Join(root, markerFile)
	mInfo, err := os.Lstat(marker)
	if err != nil || !mInfo.Mode().IsRegular() {
		return nil, fmt.Errorf("unmanaged skill, preserving it: %s", root)
	}
	raw, err := os.ReadFile(marker)
	if err != nil {
		return nil, err
	}
	var m manifest
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, fmt.Errorf("invalid ownership manifest %s: %w", root, err)
	}
	if m.Schema != 1 || m.Package != packageName || m.Name != expectedName || m.Version == "" || m.Files == nil {
		return nil, fmt.Errorf("invalid ownership manifest: %s", root)
	}
	actual, err := directoryDigests(root)
	if err != nil {
		return nil, err
	}
	if !maps.Equal(actual, m.Files) {
		return nil, fmt.Errorf("locally modified skill, preserving it: %s", root)
	}
	return &m, nil
}

func directoryDigests(root string) (map[string]string, error) {
	result := map[string]string{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == root {
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlink in skill: %s", path)
		}
		if entry.IsDir() {
			return nil
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("unsupported skill entry: %s", path)
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if rel == markerFile {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		hash := sha256.Sum256(data)
		result[filepath.ToSlash(rel)] = hex.EncodeToString(hash[:])
		return nil
	})
	return result, err
}

func bundledFiles(name string) (map[string][]byte, map[string]string, error) {
	prefix := "skills/" + name
	files := map[string][]byte{}
	digests := map[string]string{}
	err := fs.WalkDir(assets, prefix, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("invalid embedded skill file: %s", path)
		}
		rel := strings.TrimPrefix(path, prefix+"/")
		if rel == path || rel == markerFile {
			return fmt.Errorf("invalid embedded skill path: %s", path)
		}
		data, err := assets.ReadFile(path)
		if err != nil {
			return err
		}
		hash := sha256.Sum256(data)
		files[rel] = data
		digests[rel] = hex.EncodeToString(hash[:])
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	if len(files) == 0 || len(files["SKILL.md"]) == 0 {
		return nil, nil, fmt.Errorf("incomplete embedded skill: %s", name)
	}
	return files, digests, nil
}

func sameManifest(a, b *manifest) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return a.Schema == b.Schema && a.Package == b.Package && a.Name == b.Name && a.Version == b.Version && maps.Equal(a.Files, b.Files)
}

func randomSuffix() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func installBundle(item planned, version string) error {
	target := item.result.Target
	parent := filepath.Dir(target)
	if err := safeDirectory(parent); err != nil {
		return err
	}
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return err
	}
	if err := safeDirectory(parent); err != nil {
		return err
	}
	suffix, err := randomSuffix()
	if err != nil {
		return err
	}
	stage, backup := target+".stage-"+suffix, target+".backup-"+suffix
	if err := os.Mkdir(stage, 0o755); err != nil {
		return err
	}
	defer os.RemoveAll(stage)
	paths := make([]string, 0, len(item.files))
	for p := range item.files {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	for _, p := range paths {
		path := filepath.Join(stage, filepath.FromSlash(p))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(path, item.files[p], 0o644); err != nil {
			return err
		}
	}
	m := manifest{Schema: 1, Package: packageName, Name: item.result.Name, Version: version, Files: item.digests}
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(stage, markerFile), append(b, '\n'), 0o644); err != nil {
		return err
	}
	if err := safeDirectory(parent); err != nil {
		return err
	}
	if item.prior != nil {
		if err := os.Rename(target, backup); err != nil {
			return err
		}
	}
	if err := os.Rename(stage, target); err != nil {
		if item.prior != nil {
			if recoveryErr := os.Rename(backup, target); recoveryErr != nil {
				return fmt.Errorf("install failed: %v; restore previous bundle manually from %s: %w", err, backup, recoveryErr)
			}
		}
		return err
	}
	if item.prior != nil {
		if err := os.RemoveAll(backup); err != nil {
			return fmt.Errorf("installed but could not remove previous backup %s: %w", backup, err)
		}
	}
	return nil
}
