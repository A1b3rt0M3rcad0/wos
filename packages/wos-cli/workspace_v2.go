package woscli

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/signing"
	"io"
	"os"
	"path/filepath"
)

func (w *Workspace) ReadV2(path string) ([]byte, error) {
	if e := w.check(path); e != nil {
		return nil, e
	}
	file, e := w.root.Open(path)
	if e != nil {
		return nil, e
	}
	defer file.Close()
	if e = validateRegularV2(file); e != nil {
		if recovered := w.recoverInitialCreatePairV2(path, file); recovered != nil {
			// A second confined reader may already have removed the same
			// technical alias; the still-open destination must now be regular.
			if validateRegularV2(file) != nil {
				return nil, e
			}
		}
		if e = validateRegularV2(file); e != nil {
			return nil, e
		}
	}
	raw, e := io.ReadAll(io.LimitReader(file, MaxLocalDocumentV2+1))
	if e != nil {
		return nil, e
	}
	if len(raw) > MaxLocalDocumentV2 {
		return nil, fmt.Errorf("local document exceeds 1 MiB")
	}
	return raw, nil
}

// WriteV2 uses cooperative caller locking plus content comparison before rename.
// A non-cooperating OS writer can race the final rename; profiles are not an OS
// isolation boundary. Changed content observed here is preserved and rejected.
func (w *Workspace) WriteV2(path string, value any, expectedDigest string) error {
	raw, e := EncodeV2Document(value)
	if e != nil {
		return e
	}
	return w.writeRawV2(path, raw, expectedDigest, false)
}
func (w *Workspace) CreateV2(path string, value any) error {
	raw, e := EncodeV2Document(value)
	if e != nil {
		return e
	}
	return w.writeRawV2(path, raw, "", true)
}
func (w *Workspace) writeRawV2(path string, raw []byte, expected string, exclusive bool) error {
	if e := w.check(path); e != nil {
		return e
	}
	if len(raw) > MaxLocalDocumentV2 {
		return fmt.Errorf("local document exceeds 1 MiB")
	}
	if e := w.Mkdir(filepath.Dir(path)); e != nil {
		return e
	}
	if exclusive {
		suffix := make([]byte, 12)
		if _, e := rand.Read(suffix); e != nil {
			return e
		}
		return w.publishRawV2(path, path+".tmp-create-"+hex.EncodeToString(suffix), raw, nil)
	}

	suffix := make([]byte, 12)
	if _, e := rand.Read(suffix); e != nil {
		return e
	}
	temporary := path + ".tmp-" + hex.EncodeToString(suffix)
	file, e := w.root.OpenFile(temporary, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if e != nil {
		return e
	}
	defer w.root.Remove(temporary)
	_, e = file.Write(raw)
	if e == nil {
		e = file.Sync()
	}
	closed := file.Close()
	if e != nil {
		return e
	}
	if closed != nil {
		return closed
	}
	current, e := w.ReadV2(path)
	if e != nil {
		return e
	}
	if expected == "" || signing.Digest(current) != expected {
		return fmt.Errorf("local_version_conflict: edited document preserved")
	}
	if e = w.check(path); e != nil {
		return e
	}
	if e = w.root.Rename(temporary, path); e != nil {
		return e
	}
	return w.syncV2Directory(filepath.Dir(path))
}
func (w *Workspace) syncV2Directory(path string) error {
	directory, e := w.root.Open(path)
	if e != nil {
		return e
	}
	defer directory.Close()
	return syncDirectory(directory)
}
