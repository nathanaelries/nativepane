package server

import (
	"bytes"
	"encoding/json"
	"nativepane/internal/office"
	"nativepane/internal/testdoc"
	"strings"
	"testing"
	"testing/fstest"
)

func TestRestrictedDocumentsCannotBeEditedByHostOrSession(t *testing.T) {
	for _, construct := range []string{`<w:ins/>`, `<w:fldSimple w:instr="DATE"/>`, `<w:sdt/>`, `<w:documentProtection/>`} {
		t.Run(construct, func(t *testing.T) {
			c := config(t)
			s, err := New(c, fstest.MapFS{})
			if err != nil {
				t.Fatal(err)
			}
			parts := testdoc.Parts("docx")
			parts["word/settings.xml"] = `<w:settings xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">` + construct + `</w:settings>`
			source := testdoc.Package(parts)
			id, token := newSession(t, s, c.APIKey, "docx", "edit")
			base := "/api/v1/sessions/" + id
			want(t, call(s, "PUT", base+"/file", c.APIKey, source), 200)
			w := call(s, "GET", base+"/document", token, nil)
			want(t, w, 200)
			var response struct {
				Mode     string
				Revision int
				Model    office.Model
			}
			if json.Unmarshal(w.Body.Bytes(), &response) != nil || response.Mode != "view" || !response.Model.ReadOnly || response.Revision != 1 {
				t.Fatal("effective view-only mode missing")
			}
			field := response.Model.Blocks[0].Fields[0]
			body, _ := json.Marshal(map[string]any{"revision": 1, "edits": []office.Edit{{ID: field.ID, Text: "Attempted bypass"}}})
			for _, credential := range []string{c.APIKey, token} {
				want(t, call(s, "PATCH", base+"/document", credential, body), 403)
			}
			w = call(s, "GET", base+"/file", token, nil)
			want(t, w, 200)
			if !bytes.Equal(w.Body.Bytes(), source) {
				t.Fatal("rejected edits changed document")
			}
			w = call(s, "GET", base+"/events", c.APIKey, nil)
			want(t, w, 200)
			if strings.Contains(w.Body.String(), `"saved"`) {
				t.Fatal("rejected edit generated saved event")
			}
		})
	}
}
