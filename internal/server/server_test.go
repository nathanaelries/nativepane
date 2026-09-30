package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"nativepane/internal/office"
	"nativepane/internal/testdoc"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
	"time"
)

func config(t *testing.T) Config {
	return Config{Auth: "bearer", APIKey: strings.Repeat("a", 32), SigningKey: strings.Repeat("s", 32), DataDir: t.TempDir(), MaxUpload: 2 << 20, MaxSessions: 10, SessionTTL: time.Hour, TokenTTL: time.Minute, FrameOrigins: []string{"https://host.example"}, CORSOrigins: []string{"https://host.example"}}
}
func call(s *Server, method, path, key string, body []byte) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, path, bytes.NewReader(body))
	if key != "" {
		r.Header.Set("Authorization", "Bearer "+key)
	}
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	return w
}
func want(t *testing.T, w *httptest.ResponseRecorder, code int) {
	t.Helper()
	if w.Code != code {
		t.Fatalf("status=%d want=%d body=%s", w.Code, code, w.Body.String())
	}
}
func newSession(t *testing.T, s *Server, key, format, mode string) (string, string) {
	t.Helper()
	w := call(s, "POST", "/api/v1/sessions", key, []byte(fmt.Sprintf(`{"filename":"example.%s","mode":"%s"}`, format, mode)))
	want(t, w, 201)
	var data map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &data)
	return data["id"].(string), data["token"].(string)
}
func TestAPIEndToEnd(t *testing.T) {
	for _, format := range []string{"docx", "xlsx", "pptx"} {
		t.Run(format, func(t *testing.T) {
			c := config(t)
			s, e := New(c, fstest.MapFS{})
			if e != nil {
				t.Fatal(e)
			}
			id, token := newSession(t, s, c.APIKey, format, "edit")
			base := "/api/v1/sessions/" + id
			original := testdoc.File(format)
			want(t, call(s, "PUT", base+"/file", c.APIKey, original), 200)
			want(t, call(s, "PUT", base+"/file", c.APIKey, original), 409)
			w := call(s, "GET", base+"/document", token, nil)
			want(t, w, 200)
			var response struct {
				Revision int          `json:"revision"`
				Model    office.Model `json:"model"`
			}
			if e = json.Unmarshal(w.Body.Bytes(), &response); e != nil {
				t.Fatal(e)
			}
			edit := map[string]any{"revision": response.Revision, "edits": []office.Edit{{ID: response.Model.Blocks[0].Fields[0].ID, Text: "Edited through API"}}}
			body, _ := json.Marshal(edit)
			want(t, call(s, "PATCH", base+"/document", token, body), 200)
			want(t, call(s, "PATCH", base+"/document", token, body), 409)
			// Recreate the server to prove a persisted edit survives a process restart.
			s, e = New(c, fstest.MapFS{})
			if e != nil {
				t.Fatal(e)
			}
			w = call(s, "GET", base+"/file", token, nil)
			want(t, w, 200)
			m, e := (office.OOXML{}).Open(w.Body.Bytes(), format)
			if e != nil {
				t.Fatal(e)
			}
			if m.Blocks[0].Fields[0].Text != "Edited through API" {
				t.Fatal("download lost edits")
			}
			want(t, call(s, "POST", base+"/events", token, []byte(`{"type":"closed"}`)), 204)
			w = call(s, "GET", base+"/events?after=0", c.APIKey, nil)
			want(t, w, 200)
			for _, kind := range []string{"created", "uploaded", "saved", "downloaded", "closed"} {
				if !strings.Contains(w.Body.String(), kind) {
					t.Fatalf("missing %s", kind)
				}
			}
			want(t, call(s, "DELETE", base, c.APIKey, nil), 204)
			want(t, call(s, "GET", base, token, nil), 404)
		})
	}
}
func TestAuthorization(t *testing.T) {
	c := config(t)
	s, _ := New(c, fstest.MapFS{})
	want(t, call(s, "POST", "/api/v1/sessions", "", []byte(`{"filename":"x.docx"}`)), 401)
	id, token := newSession(t, s, c.APIKey, "docx", "view")
	base := "/api/v1/sessions/" + id
	want(t, call(s, "PUT", base+"/file", token, testdoc.File("docx")), 403)
	want(t, call(s, "PUT", base+"/file", c.APIKey, testdoc.File("docx")), 200)
	want(t, call(s, "GET", base+"/document", "", nil), 401)
	want(t, call(s, "GET", base+"/document", token+"bad", nil), 401)
	want(t, call(s, "PATCH", base+"/document", token, []byte(`{"revision":1,"edits":[]}`)), 403)
	want(t, call(s, "PATCH", base+"/document", c.APIKey, []byte(`{"revision":1,"edits":[]}`)), 403)
	want(t, call(s, "DELETE", base, token, nil), 403)
	want(t, call(s, "POST", base+"/token", token, nil), 403)
	want(t, call(s, "POST", base+"/token", c.APIKey, nil), 200)
	other, _ := newSession(t, s, c.APIKey, "docx", "edit")
	want(t, call(s, "GET", "/api/v1/sessions/"+other, token, nil), 401)
	want(t, call(s, "GET", base+"?token="+token, "", nil), 401)
	v, _ := s.store.Get(id)
	v.ExpiresAt = time.Now().Add(-time.Second)
	expired, _ := s.token(v)
	want(t, call(s, "GET", base, expired, nil), 401)
	_ = s.store.Put(v)
	want(t, call(s, "GET", base, c.APIKey, nil), 410)
}
func TestLimitsAndHeaders(t *testing.T) {
	c := config(t)
	c.MaxUpload = 16
	c.MaxSessions = 1
	s, _ := New(c, fstest.MapFS{"index.html": {Data: []byte("hello")}})
	id, _ := newSession(t, s, c.APIKey, "docx", "edit")
	want(t, call(s, "PUT", "/api/v1/sessions/"+id+"/file", c.APIKey, make([]byte, 17)), 413)
	want(t, call(s, "POST", "/api/v1/sessions", c.APIKey, []byte(`{"filename":"second.docx"}`)), 503)
	want(t, call(s, "POST", "/api/v1/sessions", c.APIKey, []byte(`{"filename":"x.docm"}`)), 415)
	want(t, call(s, "POST", "/api/v1/sessions", c.APIKey, []byte(`{"filename":"../x.docx"}`)), 400)
	want(t, call(s, "GET", "/../../main.go", "", nil), 404)
	r := httptest.NewRequest("OPTIONS", "/api/v1/sessions", nil)
	r.Header.Set("Origin", "https://host.example")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	want(t, w, 204)
	if w.Header().Get("Access-Control-Allow-Origin") != "https://host.example" || !strings.Contains(w.Header().Get("Content-Security-Policy"), "frame-ancestors 'self' https://host.example") {
		t.Fatal(w.Header())
	}
	r.Header.Set("Origin", "https://evil.example")
	w = httptest.NewRecorder()
	s.ServeHTTP(w, r)
	want(t, w, 403)
	w = call(s, "GET", "/", "", nil)
	want(t, w, 200)
	b, _ := io.ReadAll(w.Result().Body)
	if string(b) != "hello" {
		t.Fatal(string(b))
	}
}
func TestEventHistoryBound(t *testing.T) {
	s := Session{}
	for i := 0; i < 300; i++ {
		s.event("saved")
	}
	if len(s.Events) != 256 || s.Events[0].Sequence != 45 || s.Sequence != 300 {
		t.Fatal("event history is not bounded")
	}
}
