package woscli

// Central installation management deliberately reuses the protected schema-2
// document store privately. A store belongs to one profile, not to the caller's
// current directory, and its destination is never inferred from that directory.
import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/signing"
)

type installation struct {
	SchemaVersion int      `json:"schema_version"`
	ActiveProfile string   `json:"active_profile,omitempty"`
	Profiles      []string `json:"profiles"`
}

func installationDataDir() (string, error) {
	if value := os.Getenv("WOS_DATA_DIR"); value != "" {
		if !filepath.IsAbs(value) {
			return "", usage("WOS_DATA_DIR must be an absolute path")
		}
		return filepath.Clean(value), nil
	}
	home, e := os.UserHomeDir()
	if e != nil {
		return "", e
	}
	switch runtime.GOOS {
	case "windows":
		base := os.Getenv("LOCALAPPDATA")
		if !filepath.IsAbs(base) {
			return "", fmt.Errorf("LOCALAPPDATA must be an absolute directory; set WOS_DATA_DIR explicitly")
		}
		return filepath.Join(base, "WOS"), nil
	case "darwin":
		return filepath.Join(home, "Library", "Application Support", "wos"), nil
	default:
		base := os.Getenv("XDG_DATA_HOME")
		if base == "" {
			base = filepath.Join(home, ".local", "share")
		}
		if !filepath.IsAbs(base) {
			return "", fmt.Errorf("XDG_DATA_HOME must be absolute; set WOS_DATA_DIR explicitly")
		}
		return filepath.Join(base, "wos"), nil
	}
}

func openInstallation(create bool) (*Workspace, error) {
	path, e := installationDataDir()
	if e != nil {
		return nil, e
	}
	// Refuse symlink ancestors before creating anything. os.Root then confines
	// every subsequent access, including publication and metadata replacement.
	for current := path; ; current = filepath.Dir(current) {
		info, e := os.Lstat(current)
		if e == nil && (!info.IsDir() || info.Mode()&os.ModeSymlink != 0) {
			return nil, fmt.Errorf("installation data path must contain only real directories")
		}
		if e != nil && !os.IsNotExist(e) {
			return nil, e
		}
		if filepath.Dir(current) == current {
			break
		}
	}
	if create {
		if e = os.MkdirAll(path, 0700); e != nil {
			return nil, e
		}
	}
	w, e := OpenWorkspace(path)
	if e != nil {
		if os.IsNotExist(e) {
			return nil, usage("installation is not initialized; run wos init")
		}
		return nil, e
	}
	if e = validateInstallationDirectory(w.canonicalPath); e != nil {
		w.Close()
		return nil, e
	}
	return w, nil
}

func installationLock(ctx context.Context, w *Workspace) (func(), error) {
	return w.LockV2(ctx, "installation", "")
}

func readInstallation(w *Workspace) (installation, []byte, error) {
	var value installation
	raw, e := w.ReadV2("installation.json")
	if e != nil {
		return value, nil, e
	}
	if e = signing.DecodeStrict(raw, &value, MaxLocalDocumentV2); e != nil {
		return value, nil, fmt.Errorf("invalid installation metadata: %w", e)
	}
	if value.SchemaVersion != 1 || value.Profiles == nil {
		return value, nil, fmt.Errorf("unsupported installation metadata")
	}
	seen := map[string]bool{}
	active := value.ActiveProfile == ""
	for _, name := range value.Profiles {
		if !validProfileName(name) || seen[strings.ToLower(name)] {
			return value, nil, fmt.Errorf("invalid or case-colliding installation profile")
		}
		seen[strings.ToLower(name)] = true
		active = active || name == value.ActiveProfile
	}
	if !active {
		return value, nil, fmt.Errorf("active profile is absent from installation")
	}
	return value, raw, nil
}

func writeInstallation(w *Workspace, value installation, before []byte) error {
	sort.Strings(value.Profiles)
	raw, e := json.MarshalIndent(value, "", "  ")
	if e != nil {
		return e
	}
	if before == nil {
		return w.writeRawV2("installation.json", raw, "", true)
	}
	return w.writeRawV2("installation.json", raw, signing.Digest(before), false)
}

func installationSelect(value installation, explicit, environment string) (string, error) {
	selected := explicit
	if selected == "" {
		selected = environment
	}
	if selected == "" {
		selected = value.ActiveProfile
	}
	if selected == "" && len(value.Profiles) == 1 {
		selected = value.Profiles[0]
	}
	if selected == "" {
		return "", usage("profile_required: use wos profile use NAME, --profile NAME, or WOS_PROFILE")
	}
	for _, name := range value.Profiles {
		if name == selected {
			return name, nil
		}
	}
	return "", usage("profile not found; profile names are case-sensitive")
}

func openCentralProfile(w *Workspace, name string) (*Workspace, error) {
	path := filepath.Join("profiles", name)
	if !validProfileName(name) {
		return nil, fmt.Errorf("invalid profile name")
	}
	if e := w.check(path); e != nil {
		return nil, e
	}
	return OpenWorkspace(filepath.Join(w.canonicalPath, path))
}

func runCentral(ctx context.Context, o options) (Output, error) {
	result := Output{Operation: strings.Join(o.args, " ")}
	if o.args[0] == "init" {
		if len(o.args) != 1 || o.values["workspace-schema"] != "" {
			return result, usage("wos init initializes the installation; use explicit --workspace for legacy initialization")
		}
		if o.values["from"] != "" {
			return migrateCentral(ctx, o)
		}
		w, e := openInstallation(true)
		if e != nil {
			return result, e
		}
		defer w.Close()
		release, e := installationLock(ctx, w)
		if e != nil {
			return result, e
		}
		defer release()
		value, _, e := readInstallation(w)
		if os.IsNotExist(e) {
			value = installation{SchemaVersion: 1, Profiles: []string{}}
			e = writeInstallation(w, value, nil)
		}
		if e != nil {
			return result, e
		}
		result.Data = map[string]any{"initialized": true, "data_dir": w.canonicalPath, "profiles": value.Profiles, "active_profile": value.ActiveProfile}
		return result, nil
	}
	w, e := openInstallation(false)
	if e != nil {
		return result, e
	}
	defer w.Close()
	if o.args[0] == "profile" {
		return centralProfileCommand(ctx, w, o)
	}
	if o.args[0] == "project" || o.args[0] == "workspace" {
		return result, usage("project/workspace commands require explicit --workspace; use wos init or wos init --from PATH for central installation")
	}
	release, e := installationLock(ctx, w)
	if e != nil {
		return result, e
	}
	value, _, e := readInstallation(w)
	var name string
	if e == nil {
		name, e = installationSelect(value, o.values["profile"], os.Getenv("WOS_PROFILE"))
	}
	var profile *Workspace
	if e == nil {
		profile, e = openCentralProfile(w, name)
	}
	release()
	if e != nil {
		return result, e
	}
	defer profile.Close()
	if e = refuseMigratedSource(profile); e != nil {
		return result, e
	}
	// Copy the resolved selection into this invocation. Changes to the persisted
	// active profile cannot redirect a command while it is in flight.
	o.values["profile"] = name
	return runWorkspaceV2(ctx, profile, o)
}

func centralProfileCommand(ctx context.Context, w *Workspace, o options) (Output, error) {
	result := Output{Operation: strings.Join(o.args, " ")}
	if len(o.args) < 2 || len(o.args) > 3 {
		return result, usage("profile create|list|show|use|remove|recover|trust [NAME]")
	}
	release, e := installationLock(ctx, w)
	if e != nil {
		return result, e
	}
	defer release()
	value, before, e := readInstallation(w)
	if e != nil {
		return result, e
	}
	if o.args[1] == "list" {
		if len(o.args) != 2 {
			return result, usage("profile list takes no name")
		}
		items := []any{}
		for _, name := range value.Profiles {
			store, e := openCentralProfile(w, name)
			if e != nil {
				return result, e
			}
			profile, e := store.LoadProfileV2(name)
			store.Close()
			if e != nil {
				return result, e
			}
			items = append(items, map[string]any{"name": name, "active": value.ActiveProfile == name, "server": profile.Binding.ServerOrigin, "namespace_id": profile.Binding.NamespaceID, "principal_id": profile.Binding.PrincipalID, "pending_count": len(profile.Local.PendingOperations)})
		}
		result.Data = map[string]any{"active_profile": value.ActiveProfile, "items": items}
		return result, nil
	}
	name := o.values["profile"]
	if len(o.args) == 3 {
		if name != "" && name != o.args[2] {
			return result, usage("profile selectors differ")
		}
		name = o.args[2]
	}
	if o.args[1] == "create" {
		return createCentralProfile(ctx, w, value, before, o, name)
	}
	name, e = installationSelect(value, name, os.Getenv("WOS_PROFILE"))
	if e != nil {
		return result, e
	}
	store, e := openCentralProfile(w, name)
	if e != nil {
		return result, e
	}
	defer store.Close()
	switch o.args[1] {
	case "use", "show":
		if o.args[1] == "use" {
			if e = refuseMigratedSource(store); e != nil {
				return result, e
			}
		}
		profile, e := store.LoadProfileV2(name)
		if e != nil {
			return result, e
		}
		if o.args[1] == "use" {
			value.ActiveProfile = name
			if e = writeInstallation(w, value, before); e != nil {
				return result, e
			}
		}
		result.Data = map[string]any{"profile": compactProfileV2(profile), "active": value.ActiveProfile == name, "authorization": "local configuration only; every operation verifies current server authorization"}
		return result, nil
	case "remove":
		if value.ActiveProfile == name {
			return result, usage("active profile cannot be removed; use another profile first")
		}
		return removeCentralProfile(ctx, w, store, value, before, name)
	case "inspect", "recover", "trust":
		o.values["profile"] = name
		return profileCommandV2(ctx, store, o, defaultSecretResolverV2(store))
	default:
		return result, usage("profile create|list|show|use|remove|inspect|recover|trust [NAME]")
	}
}

func removeCentralProfile(ctx context.Context, installationStore, store *Workspace, value installation, before []byte, name string) (Output, error) {
	result := Output{Operation: "profile remove"}
	// Allow explicit retry after a metadata CAS failure. Ordinary operations
	// still honor the retirement marker before and after acquiring their lock.
	store.migrationAccess = true
	release, e := store.LockV2(ctx, name, "")
	if e != nil {
		return result, e
	}
	defer release()
	profile, _, token, e := loadProfileClientV2(store, name, defaultSecretResolverV2(store))
	if e != nil {
		return result, e
	}
	clear(token)
	if len(profile.Local.PendingOperations) != 0 {
		return result, usage("pending intentions must be reconciled before profile removal")
	}
	entries, e := store.readDirectoryV2(filepath.Join(".wos/profiles", name, "contract"))
	if e != nil && !os.IsNotExist(e) {
		return result, e
	}
	if len(entries) != 0 {
		return result, usage("local contracts must be confirmed before profile removal")
	}
	if _, e = store.root.Lstat(centralRetirementMarker); os.IsNotExist(e) {
		e = store.writeRawV2(centralRetirementMarker, []byte(`{"retired":true}`), "", true)
	}
	if e != nil {
		return result, e
	}
	for i, item := range value.Profiles {
		if item == name {
			value.Profiles = append(value.Profiles[:i], value.Profiles[i+1:]...)
			break
		}
	}
	if e = writeInstallation(installationStore, value, before); e != nil {
		return result, e
	}
	// Preserve all receipts, history, and profile material. Removal changes
	// selection, never remote credentials or shared keyring entries.
	result.Data = map[string]any{"removed": name, "remote_identity_revoked": false, "secrets_removed": false, "local_history_preserved": true}
	return result, nil
}

func createCentralProfile(ctx context.Context, w *Workspace, value installation, before []byte, o options, name string) (Output, error) {
	result := Output{Operation: "profile create"}
	if !validProfileName(name) {
		return result, usage("profile create NAME requires a valid profile name")
	}
	for _, existing := range value.Profiles {
		if strings.EqualFold(existing, name) {
			return result, usage("profile name already reserved")
		}
	}
	entries, e := w.readDirectoryV2("profiles")
	if e != nil && !os.IsNotExist(e) {
		return result, e
	}
	seenNames := map[string]bool{}
	for _, entry := range entries {
		folded := strings.ToLower(entry.Name())
		if !validProfileName(entry.Name()) || !entry.IsDir() || entry.Type()&os.ModeSymlink != 0 || seenNames[folded] {
			return result, fmt.Errorf("invalid central profile directory preserved")
		}
		seenNames[folded] = true
		if strings.EqualFold(entry.Name(), name) && entry.Name() != name {
			return result, usage("profile name collides by case with preserved local material")
		}
	}
	if o.values["file"] != "" {
		return result, usage("central profile creation uses protected enrollment flags; use wos init --from PATH to import an existing profile")
	}
	for _, ref := range []string{o.values["credential-ref"], o.values["private-key-ref"]} {
		if strings.HasPrefix(ref, "mounted:") {
			path := strings.TrimPrefix(ref, "mounted:")
			if relative, e := filepath.Rel(w.canonicalPath, path); e == nil && (relative == "." || filepath.IsLocal(relative)) {
				return result, usage("mounted secrets must remain outside the central installation directory")
			}
		}
	}
	server, e := exactOrigin(o.values["server"])
	if e != nil {
		return result, e
	}
	serverID, e := d.ParseID(o.values["server-id"])
	if e != nil {
		return result, usage("profile create requires the administrator-approved --server-id")
	}
	namespace, e := d.ParseID(o.values["namespace"])
	if e != nil {
		return result, usage("profile create requires --namespace")
	}
	var outcome d.ID
	if o.values["outcome"] != "" {
		outcome, e = d.ParseID(o.values["outcome"])
		if e != nil {
			return result, e
		}
	}
	path := filepath.Join("profiles", name)
	// A removed store is fenced and retained until this name is explicitly
	// provisioned again. Archive it without deleting any local evidence.
	if e = w.check(filepath.Join(path, centralRetirementMarker)); e != nil {
		return result, e
	}
	if _, e = w.root.Lstat(filepath.Join(path, centralRetirementMarker)); e == nil {
		id, e := newLocalIDV2()
		if e != nil {
			return result, e
		}
		if e = w.Mkdir("retired-profiles"); e != nil {
			return result, e
		}
		if e = w.root.Rename(path, filepath.Join("retired-profiles", name+"-"+id.String())); e != nil {
			return result, e
		}
		if e = w.syncV2Directory("retired-profiles"); e != nil {
			return result, e
		}
	} else if !os.IsNotExist(e) {
		return result, e
	}
	if e = w.Mkdir(path); e != nil {
		return result, e
	}
	store, e := openCentralProfile(w, name)
	if e != nil {
		return result, e
	}
	defer store.Close()
	project := ProjectV2{SchemaVersion: 2, Kind: "WOSProject", Name: name, Connection: ProjectConnectionV2{ServerURL: server, ExpectedServerID: serverID}, Scope: WorkspaceScope{NamespaceID: namespace, DefaultOutcomeID: outcome}, Workspace: ProjectLayoutV2{ProfilesDirectory: "profiles"}}
	if e = project.Validate(); e != nil {
		return result, e
	}
	if existing, e := store.LoadProjectV2(); e == nil {
		if existing != project {
			return result, fmt.Errorf("unfinished onboarding destination differs; existing material preserved")
		}
	} else if os.IsNotExist(e) {
		if e = store.CreateV2(".wos/project.yaml", project); e != nil {
			return result, e
		}
	} else {
		return result, e
	}
	o.values["profile"] = name
	o.args = []string{"profile", "create", name}
	result, commandErr := profileCommandV2(ctx, store, o, defaultSecretResolverV2(store))
	// Enrollment can commit remotely before the response is lost. Publish the
	// exact pending profile even on failure, so profile recover can reconcile it.
	if _, e = store.LoadProfileV2(name); e == nil {
		value.Profiles = append(value.Profiles, name)
		if value.ActiveProfile == "" {
			value.ActiveProfile = name
		}
		if e = writeInstallation(w, value, before); e != nil {
			return result, e
		}
	}
	return result, commandErr
}
