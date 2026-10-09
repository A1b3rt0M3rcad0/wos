package woscli

import (
	"bytes"
	"context"
	"fmt"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/signing"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestV2PublicationChild(t *testing.T) {
	root := os.Getenv("WOS_V2_PUBLICATION_FIXTURE")
	if root == "" {
		return
	}
	_, file := contractFixtureV2(t)
	profile, view := file.Local.Profile, func() ContractViewV2 {
		p, _ := contractFixtureV2(t)
		v, e := file.VerifyIssued(p)
		if e != nil {
			t.Fatal(e)
		}
		return v
	}()
	id := d.ID(view.Authority.ContractID)
	intent := d.ID("0199ac10-0000-7000-8000-000000000010")
	w, e := OpenWorkspace(root)
	if e != nil {
		t.Fatal(e)
	}
	defer w.Close()
	raw, e := EncodeV2Document(file)
	if e != nil {
		t.Fatal(e)
	}
	point := os.Getenv("WOS_V2_PUBLICATION_POINT")
	e = w.publishRawV2(contractPathV2(profile, id), contractStagePathV2(profile, id, intent), raw, func(stage string) error {
		if point == stage {
			os.Exit(23)
		}
		return nil
	})
	if e != nil {
		t.Fatal(e)
	}
}
func TestAtomicContractPublicationRecoversProcessDeathAndDraftEdits(t *testing.T) {
	for _, point := range []string{"staged", "published", "unlinked"} {
		t.Run(point, func(t *testing.T) {
			t.Setenv("XDG_CACHE_HOME", t.TempDir())
			t.Setenv("LOCALAPPDATA", t.TempDir())
			project, profile, token := profileFixtureV2(t, "executor_a")
			_, file := contractFixtureV2(t)
			view, e := file.VerifyIssued(profile)
			if e != nil {
				t.Fatal(e)
			}
			id := d.ID(view.Authority.ContractID)
			intent := d.ID("0199ac10-0000-7000-8000-000000000010")
			root := t.TempDir()
			w, e := OpenWorkspace(root)
			if e != nil {
				t.Fatal(e)
			}
			defer w.Close()
			must := func(e error) {
				t.Helper()
				if e != nil {
					t.Fatal(e)
				}
			}
			must(w.CreateV2(".wos/project.yaml", project))
			must(w.CreateV2(profilePathV2(profile.Name), profile))
			process := exec.Command(os.Args[0], "-test.run=^TestV2PublicationChild$")
			process.Env = append(os.Environ(), "WOS_V2_PUBLICATION_FIXTURE="+root, "WOS_V2_PUBLICATION_POINT="+point)
			output, e := process.CombinedOutput()
			exit, ok := e.(*exec.ExitError)
			if !ok || exit.ExitCode() != 23 {
				t.Fatalf("child did not crash at %s: %v %s", point, e, output)
			}
			path := contractPathV2(profile.Name, id)
			stage := contractStagePathV2(profile.Name, id, intent)
			if point == "staged" {
				if _, e = w.root.Stat(path); !os.IsNotExist(e) {
					t.Fatal("partial destination published before exclusive link")
				}
			}
			if point == "published" {
				// An ordinary editor writes through either alias after publication. Recovery
				// unlinks only the staging name and retains the edited destination contents.
				file.Execution.Progress.Summary = "editor change after publication and process death"
				edited, e := EncodeV2Document(file)
				must(e)
				must(os.WriteFile(filepath.Join(root, path), edited, 0600))
			}
			must(w.recoverContractStageV2(profile, id, intent, file))
			observed, e := w.ReadV2(path)
			must(e)
			var recovered ContractFileV2
			must(DecodeV2Document(observed, &recovered))
			if recovered.Execution.Progress.Summary != file.Execution.Progress.Summary {
				t.Fatal("recovery discarded editor change")
			}
			if _, e = w.root.Stat(stage); !os.IsNotExist(e) {
				t.Fatal("known staging alias retained")
			}
			must(mutateProfileV2(context.Background(), w, profile, token, func(current *ProfileV2) error { return nil }))
			if e = w.createAcceptedContractV2(profile, id, intent, file); e == nil {
				t.Fatal("second publication overwrote existing destination")
			}
			preserved, e := w.ReadV2(path)
			must(e)
			if !bytes.Equal(observed, preserved) {
				t.Fatal("exclusive publication changed existing draft")
			}
		})
	}
}

func TestContractStagingRepairsOnlyOriginalPrefixAndRejectsOutsideLinks(t *testing.T) {
	for _, kind := range []string{"partial", "edited", "outside-link"} {
		t.Run(kind, func(t *testing.T) {
			p, file := contractFixtureV2(t)
			view, e := file.VerifyIssued(p)
			if e != nil {
				t.Fatal(e)
			}
			id := d.ID(view.Authority.ContractID)
			intent := d.ID("0199ac10-0000-7000-8000-000000000010")
			root := t.TempDir()
			w, e := OpenWorkspace(root)
			if e != nil {
				t.Fatal(e)
			}
			defer w.Close()
			raw, e := EncodeV2Document(file)
			if e != nil {
				t.Fatal(e)
			}
			stage := contractStagePathV2(p.Name, id, intent)
			if e = w.Mkdir(filepath.Dir(stage)); e != nil {
				t.Fatal(e)
			}
			content := raw[:len(raw)/2]
			if kind == "edited" {
				content = []byte("unrelated editor content; preserve")
			}
			if kind == "outside-link" {
				content = raw
			}
			if e = os.WriteFile(filepath.Join(root, stage), content, 0600); e != nil {
				t.Fatal(e)
			}
			if kind == "outside-link" {
				// Exact published pair plus an outside alias must remain forbidden.
				if e = w.root.Link(stage, contractPathV2(p.Name, id)); e != nil {
					t.Fatal(e)
				}
				if e = os.Link(filepath.Join(root, stage), filepath.Join(t.TempDir(), "outside")); e != nil {
					t.Skipf("filesystem hardlinks unavailable: %v", e)
				}
			}
			e = w.recoverContractStageV2(p, id, intent, file)
			if kind == "partial" {
				if e != nil {
					t.Fatal(e)
				}
				got, e := w.ReadV2(contractPathV2(p.Name, id))
				if e != nil || !bytes.Equal(got, raw) {
					t.Fatal("original partial stage not recovered")
				}
			} else {
				if e == nil {
					t.Fatal("unrelated staged data/links accepted")
				}
				got, e := os.ReadFile(filepath.Join(root, stage))
				if e != nil || !bytes.Equal(got, content) {
					t.Fatal("failed reconciliation changed staged data")
				}
			}
		})
	}
}
func TestInitialPublicationNormalizesOnlyItsOwnConfinedTwoAliasPair(t *testing.T) {
	root := t.TempDir()
	w, e := OpenWorkspace(root)
	if e != nil {
		t.Fatal(e)
	}
	defer w.Close()
	project, _, _ := profileFixtureV2(t, "executor_a")
	raw, e := EncodeV2Document(project)
	if e != nil {
		t.Fatal(e)
	}
	path := ".wos/project.yaml"
	stage := path + ".tmp-create-000000000000000000000001"
	e = w.publishRawV2(path, stage, raw, func(point string) error {
		if point == "published" {
			return fmt.Errorf("simulated post-publication error")
		}
		return nil
	})
	if e == nil {
		t.Fatal("fault not injected")
	}
	got, e := w.ReadV2(path)
	if e != nil || signing.Digest(got) != signing.Digest(raw) {
		t.Fatal("published initial document not recovered")
	}
	if _, e = w.root.Stat(stage); !os.IsNotExist(e) {
		t.Fatal("technical alias remains")
	}
}

func TestPublicationRecoveryPreservesRenamedYMLDraft(t *testing.T) {
	p, file := contractFixtureV2(t)
	view, e := file.VerifyIssued(p)
	if e != nil {
		t.Fatal(e)
	}
	id := d.ID(view.Authority.ContractID)
	intent := d.ID("0199ac10-0000-7000-8000-000000000010")
	root := t.TempDir()
	w, e := OpenWorkspace(root)
	if e != nil {
		t.Fatal(e)
	}
	defer w.Close()
	path := contractPathV2(p.Name, id)
	stage := contractStagePathV2(p.Name, id, intent)
	raw, e := EncodeV2Document(file)
	if e != nil {
		t.Fatal(e)
	}
	e = w.publishRawV2(path, stage, raw, func(point string) error {
		if point == "published" {
			return fmt.Errorf("process stopped after publication")
		}
		return nil
	})
	if e == nil {
		t.Fatal("fault not injected")
	}
	renamed := path[:len(path)-4] + "yml"
	if e = w.root.Rename(path, renamed); e != nil {
		t.Fatal(e)
	}
	file.Execution.Progress.Summary = "renamed YAML draft is retained"
	edited, e := EncodeV2Document(file)
	if e != nil {
		t.Fatal(e)
	}
	if e = os.WriteFile(filepath.Join(root, renamed), edited, 0600); e != nil {
		t.Fatal(e)
	}
	if e = w.recoverContractStageV2(p, id, intent, file); e != nil {
		t.Fatal(e)
	}
	if _, e = w.root.Stat(path); !os.IsNotExist(e) {
		t.Fatal("recovery created a duplicate .yaml alias")
	}
	got, e := w.ReadV2(renamed)
	if e != nil || !bytes.Equal(got, edited) {
		t.Fatal("renamed draft changed")
	}
}
