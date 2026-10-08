package woscli

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Workspace struct {
	root          *os.Root
	canonicalPath string
}

func OpenWorkspace(path string) (*Workspace, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("workspace must be a real directory")
	}
	root, err := os.OpenRoot(path)
	if err != nil {
		return nil, err
	}
	canonical, err := filepath.Abs(path)
	if err != nil {
		root.Close()
		return nil, err
	}
	canonical, err = filepath.EvalSymlinks(canonical)
	if err != nil {
		root.Close()
		return nil, err
	}
	return &Workspace{root: root, canonicalPath: canonical}, nil
}
func (w *Workspace) Close() error { return w.root.Close() }
func (w *Workspace) check(path string) error {
	if filepath.IsAbs(path) || !filepath.IsLocal(path) || strings.Contains(path, "\\") || strings.Contains(path, ":") {
		return fmt.Errorf("path must stay inside workspace")
	}
	current := ""
	for _, part := range strings.Split(filepath.Clean(path), string(filepath.Separator)) {
		current = filepath.Join(current, part)
		info, err := w.root.Lstat(current)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlink/reparse path is forbidden: %s", current)
		}
	}
	return nil
}
func (w *Workspace) Mkdir(path string) error {
	if err := w.check(path); err != nil {
		return err
	}
	return w.root.MkdirAll(path, 0700)
}
func (w *Workspace) Read(path string) ([]byte, error) {
	if err := w.check(path); err != nil {
		return nil, err
	}
	f, err := w.root.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("workspace file must be regular")
	}
	raw, err := io.ReadAll(io.LimitReader(f, (512<<10)+1))
	if err != nil {
		return nil, err
	}
	if len(raw) > 512<<10 {
		return nil, fmt.Errorf("workspace file exceeds limit")
	}
	return raw, nil
}
func (w *Workspace) AtomicWrite(path string, raw []byte) error {
	if len(raw) > 512<<10 {
		return fmt.Errorf("workspace file exceeds limit")
	}
	if err := w.check(path); err != nil {
		return err
	}
	if err := w.Mkdir(filepath.Dir(path)); err != nil {
		return err
	}
	suffix := make([]byte, 12)
	if _, err := rand.Read(suffix); err != nil {
		return err
	}
	temp := path + ".tmp-" + hex.EncodeToString(suffix)
	f, err := w.root.OpenFile(temp, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	defer w.root.Remove(temp)
	_, writeErr := f.Write(raw)
	if writeErr == nil {
		writeErr = f.Sync()
	}
	closeErr := f.Close()
	if writeErr != nil {
		return writeErr
	}
	if closeErr != nil {
		return closeErr
	}
	if err = w.check(path); err != nil {
		return err
	}
	if err = w.root.Rename(temp, path); err != nil {
		return err
	}
	// Directory sync is effective on Linux. Unsupported Windows directory Sync
	// is handled by the platform test/release gate rather than claiming POSIX durability.
	dir, err := w.root.Open(filepath.Dir(path))
	if err != nil {
		return err
	}
	defer dir.Close()
	if err = syncDirectory(dir); err != nil {
		return err
	}
	return nil
}
func (w *Workspace) WriteDocument(path string, value any) error {
	raw, err := EncodeDocument(value)
	if err != nil {
		return err
	}
	return w.AtomicWrite(path, raw)
}
func (w *Workspace) ReadDocument(path string, target any) error {
	raw, err := w.Read(path)
	if err != nil {
		return err
	}
	return DecodeDocument(raw, target)
}
func (w *Workspace) DocumentPath(base string) (string, error) {
	_, a := w.root.Lstat(base + ".yaml")
	_, b := w.root.Lstat(base + ".yml")
	if a == nil && b == nil {
		return "", fmt.Errorf("ambiguous .yaml/.yml files")
	}
	if a == nil {
		return base + ".yaml", nil
	}
	if b == nil {
		return base + ".yml", nil
	}
	if !os.IsNotExist(a) {
		return "", a
	}
	if !os.IsNotExist(b) {
		return "", b
	}
	return base + ".yaml", nil
}
func (w *Workspace) Lock(dir string) (func(), error) {
	if err := w.Mkdir(dir); err != nil {
		return nil, err
	}
	path := filepath.Join(dir, "workspace.lock")
	if err := w.check(path); err != nil {
		return nil, err
	}
	f, err := w.root.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return nil, fmt.Errorf("workspace lock unavailable; inspect abandoned lock before explicit recovery: %w", err)
	}
	nonce := make([]byte, 16)
	if _, err = rand.Read(nonce); err != nil {
		f.Close()
		w.root.Remove(path)
		return nil, err
	}
	raw, _ := json.Marshal(map[string]any{"pid": os.Getpid(), "instance": hex.EncodeToString(nonce), "started_at": time.Now().UTC()})
	_, err = f.Write(raw)
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		w.root.Remove(path)
		return nil, err
	}
	return func() { w.root.Remove(path) }, nil
}
