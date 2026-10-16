package woscli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/signing"
)

const centralMigrationMarker = ".wos/central-migration.json"
const centralRetirementMarker = ".wos/central-retired.json"

type centralMigrationFile struct {
	SourcePath string `json:"source_path"`
	Profile    string `json:"profile"`
	StorePath  string `json:"store_path"`
	Digest     string `json:"sha256"`
	Bytes      int    `json:"bytes"`
}
type centralMigrationManifest struct {
	SchemaVersion int                    `json:"schema_version"`
	Source        string                 `json:"source"`
	Destination   string                 `json:"destination"`
	Profiles      []string               `json:"profiles"`
	Files         []centralMigrationFile `json:"files"`
}

func refuseMigratedSource(w *Workspace) error {
	if _, e := w.root.Lstat(centralRetirementMarker); e == nil {
		return usage("profile was removed; select an active profile")
	} else if !os.IsNotExist(e) {
		return e
	}
	_, e := w.root.Lstat(centralMigrationMarker)
	if os.IsNotExist(e) {
		return nil
	}
	if e != nil {
		return e
	}
	return usage("legacy source is fenced by a central migration; use the central installation, or resume wos init --from PATH --writers-drained")
}

// Unlike a normal recovery read, inventory is strictly read-only: incomplete
// publication pairs, symlinks, hardlinks and unrecognized special files fail.
func migrationRead(w *Workspace, path string) ([]byte, error) {
	if e := w.check(path); e != nil {
		return nil, e
	}
	info, e := w.root.Lstat(path)
	if e != nil {
		return nil, e
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("migration source must contain only regular files")
	}
	file, e := w.root.Open(path)
	if e != nil {
		return nil, e
	}
	defer file.Close()
	if e = validateRegularV2(file); e != nil {
		return nil, e
	}
	raw, e := io.ReadAll(io.LimitReader(file, MaxLocalDocumentV2+1))
	if e == nil && len(raw) > MaxLocalDocumentV2 {
		e = fmt.Errorf("migration document exceeds protected document bound")
	}
	return raw, e
}

func centralMigrationInventory(source *Workspace, destination string) (centralMigrationManifest, map[string][]byte, error) {
	manifest := centralMigrationManifest{SchemaVersion: 1, Source: source.canonicalPath, Destination: destination, Profiles: []string{}, Files: []centralMigrationFile{}}
	contents := map[string][]byte{}
	projectPath, e := source.DocumentPath(".wos/project")
	if e != nil {
		return manifest, contents, e
	}
	projectRaw, e := migrationRead(source, projectPath)
	if os.IsNotExist(e) {
		return manifest, contents, usage("central migration requires a schema-2 source; reconcile schema-1 intentions and use explicit --workspace PATH workspace migrate --to 2 first")
	}
	if e != nil {
		return manifest, contents, e
	}
	var project ProjectV2
	if e = DecodeV2Document(projectRaw, &project); e != nil {
		return manifest, contents, e
	}
	if e = project.Validate(); e != nil {
		return manifest, contents, e
	}
	names, e := source.ProfileNamesV2()
	if e != nil {
		return manifest, contents, e
	}
	if len(names) == 0 {
		return manifest, contents, usage("source has no profiles to migrate")
	}
	manifest.Profiles = names
	total := 0
	appendFile := func(profile, from, to string, raw []byte) error {
		total += len(raw)
		if len(manifest.Files) >= 10000 || total > 64<<20 {
			return fmt.Errorf("migration inventory exceeds 10000 files or 64 MiB; preserve source and migrate smaller stores")
		}
		manifest.Files = append(manifest.Files, centralMigrationFile{SourcePath: from, Profile: profile, StorePath: to, Digest: signing.Digest(raw), Bytes: len(raw)})
		contents[profile+"/"+to] = raw
		return nil
	}
	for _, name := range names {
		if e = appendFile(name, projectPath, projectPath, projectRaw); e != nil {
			return manifest, contents, e
		}
		base := filepath.Join(".wos/profiles", name)
		var walk func(string) error
		walk = func(path string) error {
			entries, e := source.readDirectoryV2(path)
			if e != nil {
				return e
			}
			sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
			for _, entry := range entries {
				next := filepath.Join(path, entry.Name())
				if entry.Type()&os.ModeSymlink != 0 || strings.Contains(entry.Name(), ".tmp-") {
					return fmt.Errorf("source contains symlink or incomplete publication; recover it before migrating")
				}
				if entry.IsDir() {
					if e = walk(next); e != nil {
						return e
					}
					continue
				}
				raw, e := migrationRead(source, next)
				if e != nil {
					return e
				}
				if e = appendFile(name, next, next, raw); e != nil {
					return e
				}
			}
			return nil
		}
		if e = walk(base); e != nil {
			return manifest, contents, e
		}
		profilePath, e := source.DocumentPath(filepath.Join(base, "profile"))
		if e != nil {
			return manifest, contents, e
		}
		var profile ProfileV2
		if e = DecodeV2Document(contents[name+"/"+profilePath], &profile); e != nil {
			return manifest, contents, e
		}
		if e = profile.Validate(); e != nil {
			return manifest, contents, e
		}
		if profile.Name != name || profile.Binding.ServerID != project.Connection.ExpectedServerID || profile.Binding.ServerOrigin != project.Connection.ServerURL || profile.Binding.NamespaceID != project.Scope.NamespaceID {
			return manifest, contents, fmt.Errorf("source profile/project immutable binding differs")
		}
		resolver := defaultSecretResolverV2(source)
		token, e := resolver.Read(profile.Authentication.CredentialRef)
		if e != nil {
			return manifest, contents, fmt.Errorf("profile %s protected credential unavailable; supply its original secret reference before migration", name)
		}
		e = profile.VerifyBinding(token)
		clear(token)
		if e != nil {
			return manifest, contents, e
		}
		for _, ref := range []string{profile.Authentication.CredentialRef, profile.Signing.PrivateKeyRef} {
			if strings.HasPrefix(ref, "mounted:") {
				path := strings.TrimPrefix(ref, "mounted:")
				if rel, e := filepath.Rel(destination, path); e == nil && (rel == "." || filepath.IsLocal(rel)) {
					return manifest, contents, fmt.Errorf("mounted secret would be inside the central installation; preserve source and reprovision the protected reference explicitly")
				}
				secret, e := resolver.Read(ref)
				clear(secret)
				if e != nil {
					return manifest, contents, fmt.Errorf("protected mounted reference cannot be preserved safely: %w", e)
				}
			}
		}
		for _, file := range manifest.Files {
			if file.Profile != name || filepath.Dir(file.StorePath) != filepath.Join(base, "contract") {
				continue
			}
			idText := strings.TrimSuffix(strings.TrimSuffix(filepath.Base(file.StorePath), ".yaml"), ".yml")
			id, e := d.ParseID(idText)
			if e != nil {
				return manifest, contents, fmt.Errorf("unrecognized contract filename preserved in source")
			}
			var contract ContractFileV2
			if e = DecodeV2Document(contents[name+"/"+file.StorePath], &contract); e != nil {
				return manifest, contents, fmt.Errorf("only verified signed contracts may migrate; reconcile legacy/unfinished records at the source: %w", e)
			}
			view, e := contract.VerifyIssued(profile)
			if e != nil || view.Authority.ContractID != id.String() {
				return manifest, contents, fmt.Errorf("source contract signature or filename binding differs")
			}
		}
	}
	return manifest, contents, nil
}

func migrateCentral(ctx context.Context, o options) (Output, error) {
	result := Output{Operation: "init migrate"}
	if !filepath.IsAbs(o.values["from"]) {
		return result, usage("--from requires an absolute legacy workspace path")
	}
	path, e := installationDataDir()
	if e != nil {
		return result, e
	}
	source, e := OpenWorkspace(o.values["from"])
	if e != nil {
		return result, e
	}
	defer source.Close()
	source.migrationAccess = true
	if rel, e := filepath.Rel(source.canonicalPath, path); e == nil && (rel == "." || filepath.IsLocal(rel)) {
		return result, usage("central data directory must be outside the legacy source")
	}
	if rel, e := filepath.Rel(path, source.canonicalPath); e == nil && (rel == "." || filepath.IsLocal(rel)) {
		return result, usage("legacy source must be outside the central data directory")
	}
	manifest, contents, e := centralMigrationInventory(source, path)
	if e != nil {
		return result, e
	}
	raw, e := json.Marshal(manifest)
	if e != nil {
		return result, e
	}
	digest := signing.Digest(raw)
	result.Data = map[string]any{"dry_run": o.values["dry-run"] == "true", "profiles": manifest.Profiles, "files": len(manifest.Files), "inventory_sha256": digest, "originals_preserved": true, "source_writers_must_be_drained": true}
	if o.values["dry-run"] == "true" {
		return result, nil
	}
	if o.values["writers-drained"] != "true" {
		return result, usage("migration requires --writers-drained after stopping every old CLI writer; original binaries cannot honor the migration fence")
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
	value, before, e := readInstallation(w)
	if os.IsNotExist(e) {
		value = installation{SchemaVersion: 1, Profiles: []string{}}
		before = nil
	} else if e != nil {
		return result, e
	}
	marker, markerErr := migrationReadPublished(source, centralMigrationMarker)
	if markerErr != nil && !os.IsNotExist(markerErr) {
		return result, markerErr
	}
	if markerErr == nil && !bytes.Equal(marker, raw) {
		return result, fmt.Errorf("frozen migration source or destination changed; original fence and documents preserved")
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
		for _, name := range manifest.Profiles {
			if strings.EqualFold(entry.Name(), name) && entry.Name() != name {
				return result, fmt.Errorf("destination profile directory collides by case; original source preserved")
			}
		}
	}
	for _, name := range manifest.Profiles {
		for _, existing := range value.Profiles {
			if strings.EqualFold(existing, name) && (existing != name || markerErr != nil) {
				return result, fmt.Errorf("destination profile collision; existing profile preserved")
			}
		}
		if e = w.check(filepath.Join("profiles", name)); e != nil {
			return result, e
		}
		if _, e = w.root.Lstat(filepath.Join("profiles", name)); e == nil && markerErr != nil {
			return result, fmt.Errorf("destination profile directory already exists; preserved source was not fenced")
		} else if e != nil && !os.IsNotExist(e) {
			return result, e
		}
	}
	// Acquire all original lock identities, including every extant contract.
	// Bound contention: migration must refuse an active writer, never wait for
	// an agent to finish and assume that it has actually been drained.
	lockCtx, cancel := context.WithTimeout(ctx, 250*time.Millisecond)
	defer cancel()
	locks := []func(){}
	defer func() {
		for i := len(locks) - 1; i >= 0; i-- {
			locks[i]()
		}
	}()
	for _, name := range manifest.Profiles {
		unlock, e := source.LockV2(lockCtx, name, "")
		if e != nil {
			return result, fmt.Errorf("source profile is busy; drain writers before migration: %w", e)
		}
		locks = append(locks, unlock)
		for _, entry := range manifest.Files {
			if entry.Profile == name && filepath.Dir(entry.StorePath) == filepath.Join(".wos/profiles", name, "contract") {
				id := strings.TrimSuffix(strings.TrimSuffix(filepath.Base(entry.StorePath), ".yaml"), ".yml")
				unlock, e = source.LockV2(lockCtx, name, id)
				if e != nil {
					return result, fmt.Errorf("source contract is busy; drain writers before migration: %w", e)
				}
				locks = append(locks, unlock)
			}
		}
	}
	locked, lockedContents, e := centralMigrationInventory(source, path)
	if e != nil {
		return result, e
	}
	lockedRaw, e := json.Marshal(locked)
	if e != nil || !bytes.Equal(raw, lockedRaw) {
		return result, fmt.Errorf("source changed while acquiring migration locks; no migration published")
	}
	contents = lockedContents
	if markerErr != nil {
		if e = source.writeRawV2(centralMigrationMarker, raw, "", true); e != nil {
			return result, e
		}
	}
	// The source is now fenced before destination publication. Interrupted
	// copies resume only from this exact manifest; no intention is re-signed,
	// replayed, discarded, or rebound to another identity.
	for _, name := range manifest.Profiles {
		stage := filepath.Join("migration-staging", strings.TrimPrefix(digest, "sha256:"), name)
		final := filepath.Join("profiles", name)
		if e = w.Mkdir(stage); e != nil {
			return result, e
		}
		stageStore, e := OpenWorkspace(filepath.Join(w.canonicalPath, stage))
		if e != nil {
			return result, e
		}
		for _, file := range manifest.Files {
			if file.Profile != name {
				continue
			}
			content := contents[name+"/"+file.StorePath]
			e = migrationCopyFrozenFile(stageStore, file.StorePath, content)
			if e != nil {
				stageStore.Close()
				return result, e
			}
		}
		stageStore.Close()
		if e = w.Mkdir("profiles"); e != nil {
			return result, e
		}
		if _, e = w.root.Lstat(final); os.IsNotExist(e) {
			e = w.root.Rename(stage, final)
			if e == nil {
				e = w.syncV2Directory("profiles")
			}
		} else if e == nil {
			// An interrupted attempt may already have published this store.
			store, openErr := openCentralProfile(w, name)
			if openErr != nil {
				return result, openErr
			}
			for _, file := range manifest.Files {
				if file.Profile != name {
					continue
				}
				existing, readErr := migrationRead(store, file.StorePath)
				if readErr != nil || signing.Digest(existing) != file.Digest {
					store.Close()
					return result, fmt.Errorf("published destination differs; edited material preserved")
				}
			}
			store.Close()
		}
		if e != nil {
			return result, e
		}
		present := false
		for _, existing := range value.Profiles {
			present = present || name == existing
		}
		if !present {
			value.Profiles = append(value.Profiles, name)
		}
	}
	if value.ActiveProfile == "" && len(value.Profiles) == 1 {
		value.ActiveProfile = value.Profiles[0]
	}
	if e = writeInstallation(w, value, before); e != nil {
		return result, e
	}
	result.Committed = true
	result.RequiresAction = "use the central installation; preserved source is fenced against supported CLI writes"
	return result, nil
}

func migrationCopyStage(path string, expected []byte) string {
	hash := strings.TrimPrefix(signing.Digest([]byte(path+"\x00"+signing.Digest(expected))), "sha256:")
	return path + ".tmp-create-" + hash[:24]
}

func migrationReadPublished(w *Workspace, path string) ([]byte, error) {
	raw, e := migrationRead(w, path)
	if e != nil && !os.IsNotExist(e) {
		// Operational recovery can normalize an exact recognized initial-create
		// alias pair; reject special files before attempting an ordinary open.
		if info, statErr := w.root.Lstat(path); statErr == nil && info.Mode().IsRegular() {
			return w.ReadV2(path)
		}
	}
	return raw, e
}

// A deterministic stage is bound to both the frozen target path and bytes.
// Recover only an unchanged constructor prefix or an exact recognized alias;
// edits are preserved. No partially written document is ever published.
func migrationCopyFrozenFile(w *Workspace, path string, expected []byte) error {
	stage := migrationCopyStage(path, expected)
	current, e := migrationRead(w, path)
	if e != nil && !os.IsNotExist(e) {
		// This exact stage is known from the frozen manifest. Do not depend on
		// the general reader's bounded directory scan to discover its alias.
		targetInfo, targetStatErr := w.root.Lstat(path)
		if info, statErr := w.root.Lstat(stage); statErr == nil && info.Mode().IsRegular() && targetStatErr == nil && targetInfo.Mode().IsRegular() {
			staged, stagedInfo, stageErr := w.readStageV2(stage, 2)
			published, publishedInfo, publishedErr := w.readStageV2(path, 2)
			if stageErr == nil && publishedErr == nil && os.SameFile(stagedInfo, publishedInfo) {
				if !bytes.Equal(staged, expected) || !bytes.Equal(published, expected) {
					return fmt.Errorf("migration published-pair edit preserved")
				}
				if e = w.root.Remove(stage); e != nil {
					return e
				}
				if e = w.syncV2Directory(filepath.Dir(path)); e != nil {
					return e
				}
				current, e = migrationRead(w, path)
			}
		}
	}
	if e == nil {
		if !bytes.Equal(current, expected) {
			return fmt.Errorf("migration staging edit preserved; frozen content differs")
		}
		// A duplicate single-link technical stage can remain if another reader
		// already normalized a published pair. Remove only identical bytes.
		duplicate, stageErr := migrationRead(w, stage)
		if os.IsNotExist(stageErr) {
			return nil
		}
		if stageErr != nil {
			return stageErr
		}
		if !bytes.Equal(duplicate, expected) {
			return fmt.Errorf("migration technical-stage edit preserved")
		}
		if e = w.root.Remove(stage); e != nil {
			return e
		}
		return w.syncV2Directory(filepath.Dir(path))
	}
	if !os.IsNotExist(e) {
		return e
	}
	staged, e := migrationRead(w, stage)
	if os.IsNotExist(e) {
		return w.publishRawV2(path, stage, expected, nil)
	}
	if e != nil {
		return e
	}
	if !bytes.HasPrefix(expected, staged) {
		return fmt.Errorf("migration technical-stage edit preserved; constructor prefix differs")
	}
	if !bytes.Equal(staged, expected) {
		info, e := w.root.Lstat(stage)
		if e != nil {
			return e
		}
		file, e := w.root.OpenFile(stage, os.O_WRONLY, 0600)
		if e != nil {
			return e
		}
		opened, e := file.Stat()
		latest, readErr := migrationRead(w, stage)
		if e != nil || readErr != nil || !os.SameFile(info, opened) || !bytes.Equal(latest, staged) {
			file.Close()
			return fmt.Errorf("migration constructor stage changed; observed edit preserved")
		}
		_, e = file.WriteAt(expected[len(staged):], int64(len(staged)))
		if e == nil {
			e = file.Sync()
		}
		closeErr := file.Close()
		if e != nil {
			return e
		}
		if closeErr != nil {
			return closeErr
		}
	}
	if e = w.check(path); e != nil {
		return e
	}
	if e = w.root.Link(stage, path); e != nil {
		return e
	}
	if e = w.root.Remove(stage); e != nil {
		return e
	}
	return w.syncV2Directory(filepath.Dir(path))
}
