package woscli

import (
	"bytes"
	"encoding/json"
	"fmt"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// The destination is created exclusively by linking a complete, fsynced stage.
// No partial YAML is ever published and an existing destination is preserved.
// fault is a private test seam; callers never configure it through agent input.
func (w *Workspace) publishRawV2(path, stage string, raw []byte, fault func(string) error) error {
	if len(raw) > MaxLocalDocumentV2 {
		return fmt.Errorf("publication exceeds local bound")
	}
	if e := w.Mkdir(filepath.Dir(path)); e != nil {
		return e
	}
	if e := w.check(stage); e != nil {
		return e
	}
	f, e := w.root.OpenFile(stage, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if e != nil {
		return e
	}
	_, e = f.Write(raw)
	if e == nil {
		e = f.Sync()
	}
	closed := f.Close()
	if e != nil {
		return e
	}
	if closed != nil {
		return closed
	}
	if e = w.syncV2Directory(filepath.Dir(path)); e != nil {
		return e
	}
	if fault != nil {
		if e = fault("staged"); e != nil {
			return e
		}
	}
	if e = w.check(path); e != nil {
		return e
	}
	if e = w.root.Link(stage, path); e != nil {
		return e
	}
	if fault != nil {
		if e = fault("published"); e != nil {
			return e
		}
	}
	if e = w.root.Remove(stage); e != nil && !os.IsNotExist(e) {
		return e
	}
	if fault != nil {
		if e = fault("unlinked"); e != nil {
			return e
		}
	}
	return w.syncV2Directory(filepath.Dir(path))
}
func (w *Workspace) readStageV2(path string, links uint64) ([]byte, os.FileInfo, error) {
	if e := w.check(path); e != nil {
		return nil, nil, e
	}
	f, e := w.root.Open(path)
	if e != nil {
		return nil, nil, e
	}
	defer f.Close()
	n, e := regularLinksV2(f)
	if e != nil || n != links {
		return nil, nil, fmt.Errorf("staging file identity/link count differs")
	}
	info, e := f.Stat()
	if e != nil {
		return nil, nil, e
	}
	raw, e := io.ReadAll(io.LimitReader(f, MaxLocalDocumentV2+1))
	if e != nil {
		return nil, nil, e
	}
	if len(raw) > MaxLocalDocumentV2 {
		return nil, nil, fmt.Errorf("staging file exceeds local bound")
	}
	return raw, info, nil
}
func (w *Workspace) recoverInitialCreatePairV2(path string, file *os.File) error {
	links, e := regularLinksV2(file)
	if e != nil || links != 2 {
		return fmt.Errorf("ordinary files require a single link")
	}
	info, e := file.Stat()
	if e != nil {
		return e
	}
	dir, e := w.root.Open(filepath.Dir(path))
	if e != nil {
		return e
	}
	defer dir.Close()
	entries, e := dir.ReadDir(112)
	if e != nil && e != io.EOF {
		return e
	}
	prefix := regexp.QuoteMeta(filepath.Base(path))
	extension := filepath.Ext(path)
	if extension == ".yaml" || extension == ".yml" {
		prefix = regexp.QuoteMeta(strings.TrimSuffix(filepath.Base(path), extension)) + `\.ya?ml`
	}
	pattern := regexp.MustCompile("^" + prefix + `\.(tmp-create-[0-9a-f]{24}|materialize-[0-9a-f-]{36})$`)
	for _, entry := range entries {
		if !pattern.MatchString(entry.Name()) {
			continue
		}
		stage := filepath.Join(filepath.Dir(path), entry.Name())
		_, other, e := w.readStageV2(stage, 2)
		if e != nil {
			return e
		}
		if !os.SameFile(info, other) {
			continue
		}
		// Exact two aliases inside the confined root: unlink only the technical name,
		// retaining all current contents in the destination, including editor changes.
		if e = w.root.Remove(stage); e != nil && !os.IsNotExist(e) {
			return e
		}
		return w.syncV2Directory(filepath.Dir(path))
	}
	return fmt.Errorf("unrecognized hardlink remains forbidden")
}
func contractStagePathV2(profile string, id, intent d.ID) string {
	return contractPathV2(profile, id) + ".materialize-" + intent.String()
}
func isContractStageV2(name string) bool {
	return regexp.MustCompile(`^[0-9a-f-]{36}\.yaml\.materialize-[0-9a-f-]{36}$`).MatchString(name) && d.ID(name[:36]).Validate() == nil && d.ID(name[len(name)-36:]).Validate() == nil
}
func verifySameIssuedSpecV2(profile ProfileV2, raw []byte, expected ContractFileV2) (ContractFileV2, error) {
	var file ContractFileV2
	if e := DecodeV2Document(raw, &file); e != nil {
		return file, e
	}
	view, e := file.VerifyIssued(profile)
	if e != nil {
		return file, e
	}
	original, e := expected.VerifyIssued(profile)
	if e != nil {
		return file, e
	}
	a, _ := json.Marshal(file.Issued.Specification)
	b, _ := json.Marshal(expected.Issued.Specification)
	if view.Authority.ContractID != original.Authority.ContractID || view.Authority.ContractKind != original.Authority.ContractKind || !bytes.Equal(a, b) {
		return file, fmt.Errorf("existing/staged issued target differs; preserve content")
	}
	return file, nil
}

// Called only under profile -> contract locks for an authenticated pending
// accepted acquisition. Stage identity comes from that original pending ID.
func (w *Workspace) recoverContractStageV2(profile ProfileV2, id, intent d.ID, expected ContractFileV2) error {
	stage := contractStagePathV2(profile.Name, id, intent)
	if e := w.check(stage); e != nil {
		return e
	}
	f, e := w.root.Open(stage)
	if os.IsNotExist(e) {
		return nil
	}
	if e != nil {
		return e
	}
	links, e := regularLinksV2(f)
	f.Close()
	if e != nil {
		return e
	}
	path, e := w.DocumentPath(filepath.Join(".wos/profiles", profile.Name, "contract", id.String()))
	if e != nil {
		return e
	}
	if links == 2 {
		raw, staged, e := w.readStageV2(stage, 2)
		if e != nil {
			return e
		}
		_, final, e := w.readStageV2(path, 2)
		if e != nil {
			return e
		}
		if !os.SameFile(staged, final) {
			return fmt.Errorf("staging hardlink does not match destination")
		}
		if _, e = verifySameIssuedSpecV2(profile, raw, expected); e != nil {
			return e
		}
		if e = w.root.Remove(stage); e != nil && !os.IsNotExist(e) {
			return e
		}
		return w.syncV2Directory(filepath.Dir(path))
	}
	if links != 1 {
		return fmt.Errorf("unexpected staging aliases; preserve all files")
	}
	raw, info, e := w.readStageV2(stage, 1)
	if e != nil {
		return e
	}
	final, e := w.ReadV2(path)
	encoded, encodeErr := EncodeV2Document(expected)
	if encodeErr != nil {
		return encodeErr
	}
	if e == nil {
		if _, e = verifySameIssuedSpecV2(profile, final, expected); e != nil {
			return e
		}
		if !bytes.Equal(raw, encoded) {
			return fmt.Errorf("independent staged content differs; preserve both drafts")
		}
		if e = w.root.Remove(stage); e != nil && !os.IsNotExist(e) {
			return e
		}
		return w.syncV2Directory(filepath.Dir(path))
	}
	if !os.IsNotExist(e) {
		return e
	}
	if _, e = verifySameIssuedSpecV2(profile, raw, expected); e != nil {
		// Only the known constructor prefix of this accepted intention is repaired.
		// Other edits, invalid documents and unrelated stages are never overwritten.
		if !bytes.HasPrefix(encoded, raw) {
			return fmt.Errorf("staged content differs; preserve for explicit reconciliation")
		}
		writer, e := w.root.OpenFile(stage, os.O_WRONLY, 0600)
		if e != nil {
			return e
		}
		current, e := writer.Stat()
		if e != nil {
			writer.Close()
			return e
		}
		if !os.SameFile(current, info) {
			writer.Close()
			return fmt.Errorf("staging identity changed")
		}
		if e = validateRegularV2(writer); e != nil {
			writer.Close()
			return e
		}
		if e = writer.Truncate(0); e == nil {
			_, e = writer.Write(encoded)
		}
		if e == nil {
			e = writer.Sync()
		}
		closed := writer.Close()
		if e != nil {
			return e
		}
		if closed != nil {
			return closed
		}
	}
	if e = w.root.Link(stage, path); e != nil {
		return e
	}
	if e = w.root.Remove(stage); e != nil && !os.IsNotExist(e) {
		return e
	}
	return w.syncV2Directory(filepath.Dir(path))
}
func (w *Workspace) createAcceptedContractV2(profile ProfileV2, id, intent d.ID, file ContractFileV2) error {
	raw, e := EncodeV2Document(file)
	if e != nil {
		return e
	}
	return w.publishRawV2(contractPathV2(profile.Name, id), contractStagePathV2(profile.Name, id, intent), raw, nil)
}
