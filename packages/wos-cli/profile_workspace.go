package woscli

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func (w *Workspace) ProfileNamesV2() ([]string, error) {
	path := ".wos/profiles"
	if e := w.check(path); e != nil {
		return nil, e
	}
	directory, e := w.root.Open(path)
	if os.IsNotExist(e) {
		return []string{}, nil
	}
	if e != nil {
		return nil, e
	}
	defer directory.Close()
	entries, e := directory.ReadDir(-1)
	if e != nil {
		return nil, e
	}
	seen := map[string]bool{}
	names := []string{}
	for _, entry := range entries {
		if !validProfileName(entry.Name()) || !entry.IsDir() || entry.Type()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("invalid or ambiguous profile directory")
		}
		folded := strings.ToLower(entry.Name())
		if seen[folded] {
			return nil, fmt.Errorf("profile names collide by case")
		}
		seen[folded] = true
		if e = w.check(filepath.Join(path, entry.Name())); e != nil {
			return nil, e
		}
		names = append(names, entry.Name())
	}
	sort.Strings(names)
	return names, nil
}
func (w *Workspace) SelectProfileV2(explicit string, environment string) (string, error) {
	names, e := w.ProfileNamesV2()
	if e != nil {
		return "", e
	}
	selected := explicit
	if selected == "" {
		selected = environment
	}
	if selected == "" {
		if len(names) != 1 {
			return "", fmt.Errorf("profile_required: select --profile or WOS_PROFILE")
		}
		return names[0], nil
	}
	if !validProfileName(selected) {
		return "", fmt.Errorf("invalid profile name")
	}
	for _, name := range names {
		if name == selected {
			return name, nil
		}
	}
	return "", fmt.Errorf("profile not found; case must match exactly")
}
func (w *Workspace) LoadProjectV2() (ProjectV2, error) {
	var project ProjectV2
	path, e := w.DocumentPath(".wos/project")
	if e != nil {
		return project, e
	}
	raw, e := w.ReadV2(path)
	if e != nil {
		return project, e
	}
	if e = DecodeV2Document(raw, &project); e != nil {
		return project, e
	}
	return project, project.Validate()
}
func (w *Workspace) LoadProfileV2(name string) (ProfileV2, error) {
	var profile ProfileV2
	if !validProfileName(name) {
		return profile, fmt.Errorf("invalid profile name")
	}
	path, e := w.DocumentPath(filepath.Join(".wos/profiles", name, "profile"))
	if e != nil {
		return profile, e
	}
	raw, e := w.ReadV2(path)
	if e != nil {
		return profile, e
	}
	if e = DecodeV2Document(raw, &profile); e != nil {
		return profile, e
	}
	if profile.Name != name {
		return profile, fmt.Errorf("profile directory/header binding differs")
	}
	if e = profile.Validate(); e != nil {
		return profile, e
	}
	project, e := w.LoadProjectV2()
	if e != nil {
		return profile, e
	}
	if project.Connection.ServerURL != profile.Binding.ServerOrigin || project.Connection.ExpectedServerID != profile.Binding.ServerID || project.Scope.NamespaceID != profile.Binding.NamespaceID {
		return profile, fmt.Errorf("project/profile destination binding differs")
	}
	return profile, nil
}
