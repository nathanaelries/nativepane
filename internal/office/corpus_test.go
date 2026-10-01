package office

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestOfficeFixtureCorpus(t *testing.T) {
	root := filepath.Join("..", "..", "tests", "fixtures")
	var manifest struct {
		Version  int `json:"version"`
		Fixtures []struct {
			ID, File, Format, SHA256, Semantic, Producer, ProducerVersion string
			ReadOnly                                                      bool
		} `json:"fixtures"`
	}
	data, err := os.ReadFile(filepath.Join(root, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(data, &manifest); err != nil || manifest.Version != 2 || len(manifest.Fixtures) < 3 {
		t.Fatal("invalid fixture manifest", err)
	}
	for _, fixture := range manifest.Fixtures {
		t.Run(fixture.ID, func(t *testing.T) {
			source, err := os.ReadFile(filepath.Join(root, fixture.File))
			if err != nil {
				t.Fatal(err)
			}
			digest := sha256.Sum256(source)
			if hex.EncodeToString(digest[:]) != fixture.SHA256 {
				t.Fatal("source fixture hash changed")
			}
			members := unzip(t, source)
			var props struct {
				Application string
				AppVersion  string
			}
			if xml.Unmarshal(members["docProps/app.xml"], &props) != nil || props.Application != fixture.Producer || props.AppVersion != fixture.ProducerVersion {
				t.Fatal("Office producer evidence changed")
			}
			engine := OOXML{}
			model, err := engine.Open(source, fixture.Format)
			if err != nil {
				t.Fatal(err)
			}
			if model.ReadOnly != fixture.ReadOnly {
				t.Fatal("unexpected edit permission")
			}
			actual, err := json.MarshalIndent(model, "", "  ")
			if err != nil {
				t.Fatal(err)
			}
			actual = append(actual, '\n')
			output := filepath.Join("..", "..", "tmp", "fixtures", fixture.ID+".json")
			if err = os.MkdirAll(filepath.Dir(output), 0755); err != nil {
				t.Fatal(err)
			}
			if err = os.WriteFile(output, actual, 0644); err != nil {
				t.Fatal(err)
			}
			expected, err := os.ReadFile(filepath.Join(root, fixture.Semantic))
			if err != nil || !bytes.Equal(actual, expected) {
				t.Errorf("semantic regression: compare %s with %s (%v)", output, fixture.Semantic, err)
			}
			var field Field
			for _, block := range model.Blocks {
				for _, f := range block.Fields {
					if fixture.ReadOnly || !f.ReadOnly {
						field = f
						break
					}
				}
				if field.ID != "" {
					break
				}
			}
			if field.ID == "" {
				outputBytes, err := engine.Apply(source, fixture.Format, nil)
				if err != nil || !bytes.Equal(source, outputBytes) {
					t.Fatal("projection without editable fields changed on no-op save", err)
				}
				return
			}
			if fixture.ReadOnly {
				if _, err = engine.Apply(source, fixture.Format, []Edit{{field.ID, "Refused edit"}}); !errors.Is(err, ErrReadOnly) {
					t.Fatal("restricted fixture permitted an edit", err)
				}
				return
			}
			outputBytes, err := engine.Apply(source, fixture.Format, []Edit{{field.ID, "Corpus verified edit"}})
			if err != nil {
				t.Fatal(err)
			}
			reopened, err := engine.Open(outputBytes, fixture.Format)
			if err != nil {
				t.Fatal(err)
			}
			found := false
			for _, block := range reopened.Blocks {
				for _, f := range block.Fields {
					if f.ID == field.ID && f.Text == "Corpus verified edit" {
						found = true
					}
				}
			}
			if !found {
				t.Fatal("edit lost on reopen")
			}
			after := unzip(t, outputBytes)
			changed := 0
			for name, before := range members {
				if !bytes.Equal(before, after[name]) {
					changed++
				}
			}
			if len(members) != len(after) || changed != 1 {
				t.Fatal("unexpected member changes")
			}
			originalZIP, _ := zip.NewReader(bytes.NewReader(source), int64(len(source)))
			editedZIP, _ := zip.NewReader(bytes.NewReader(outputBytes), int64(len(outputBytes)))
			for i, member := range originalZIP.File {
				if member.Name != editedZIP.File[i].Name {
					t.Fatal("member order changed")
				}
				if bytes.Equal(members[member.Name], after[member.Name]) {
					rawBefore, _ := member.OpenRaw()
					rawAfter, _ := editedZIP.File[i].OpenRaw()
					before, _ := io.ReadAll(rawBefore)
					after, _ := io.ReadAll(rawAfter)
					if !bytes.Equal(before, after) {
						t.Fatal("untouched ZIP member recompressed", member.Name)
					}
				}
			}
		})
	}
}
