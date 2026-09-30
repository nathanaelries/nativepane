package server

import (
	"encoding/json"
	"nativepane/internal/office"
	"nativepane/internal/testdoc"
	"strings"
	"testing"
	"testing/fstest"
)

func TestDocumentCapacityConfiguration(t *testing.T) {
	for _, value := range []string{"0", "-1", "1000001", "invalid"} {
		t.Run(value, func(t *testing.T) {
			t.Setenv("MAX_DOCUMENT_FIELDS", value)
			if _, err := EnvConfig(); err == nil {
				t.Fatal("invalid capacity accepted")
			}
		})
	}
	t.Setenv("MAX_DOCUMENT_FIELDS", "300000")
	c, err := EnvConfig()
	if err != nil || c.MaxDocumentFields != 300000 {
		t.Fatalf("config: %v, %v", c.MaxDocumentFields, err)
	}
	t.Setenv("MAX_DOCUMENT_FIELDS", "")
	c, err = EnvConfig()
	if err != nil || c.MaxDocumentFields != office.DefaultMaxFields {
		t.Fatal("default capacity not applied")
	}
}

func TestCapacityAPI(t *testing.T) {
	c := config(t)
	c.MaxDocumentFields = 25001
	s, err := New(c, fstest.MapFS{})
	if err != nil {
		t.Fatal(err)
	}
	id, _ := newSession(t, s, c.APIKey, "xlsx", "edit")
	base := "/api/v1/sessions/" + id
	w := call(s, "PUT", base+"/file", c.APIKey, testdoc.LargeFile("xlsx", 25002))
	want(t, w, 422)
	if !strings.Contains(w.Body.String(), "MAX_DOCUMENT_FIELDS") {
		t.Fatal("capacity response lacks a remedy")
	}
	// A rejected upload leaves the session empty and can be retried.
	want(t, call(s, "PUT", base+"/file", c.APIKey, testdoc.LargeFile("xlsx", 25001)), 200)
	want(t, call(s, "GET", base+"/document", c.APIKey, nil), 200)
	w = call(s, "GET", "/api/v1/config", "", nil)
	var limits struct {
		MaxDocumentFields int `json:"maxDocumentFields"`
	}
	if json.Unmarshal(w.Body.Bytes(), &limits) != nil || limits.MaxDocumentFields != 25001 {
		t.Fatal("configured capacity not exposed")
	}
}
