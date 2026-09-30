package server

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"nativepane/internal/office"
)

type Config struct {
	Auth, APIKey, SigningKey, DataDir string
	MaxUpload                         int64
	MaxSessions                       int
	MaxDocumentFields                 int
	SessionTTL, TokenTTL              time.Duration
	FrameOrigins, CORSOrigins         []string
}

func EnvConfig() (Config, error) {
	c := Config{Auth: env("AUTH", "none"), APIKey: os.Getenv("API_KEY"), SigningKey: os.Getenv("SESSION_SECRET"), DataDir: env("DATA_DIR", "data"), MaxUpload: 32 << 20, MaxSessions: 100, SessionTTL: time.Hour, TokenTTL: 10 * time.Minute}
	for k, p := range map[string]*int64{"MAX_UPLOAD_BYTES": &c.MaxUpload} {
		if v := os.Getenv(k); v != "" {
			n, e := strconv.ParseInt(v, 10, 64)
			if e != nil {
				return c, e
			}
			*p = n
		}
	}
	if v := os.Getenv("MAX_SESSIONS"); v != "" {
		n, e := strconv.Atoi(v)
		if e != nil {
			return c, e
		}
		c.MaxSessions = n
	}
	c.MaxDocumentFields = office.DefaultMaxFields
	if v := os.Getenv("MAX_DOCUMENT_FIELDS"); v != "" {
		n, e := strconv.Atoi(v)
		if e != nil || n < 1 || n > office.MaxDocumentFields {
			return c, fmt.Errorf("MAX_DOCUMENT_FIELDS must be between 1 and %d", office.MaxDocumentFields)
		}
		c.MaxDocumentFields = n
	}
	for k, p := range map[string]*time.Duration{"SESSION_TTL_SECONDS": &c.SessionTTL, "TOKEN_TTL_SECONDS": &c.TokenTTL} {
		if v := os.Getenv(k); v != "" {
			n, e := strconv.Atoi(v)
			if e != nil || n < 1 || n > 86400 {
				return c, fmt.Errorf("invalid %s", k)
			}
			*p = time.Duration(n) * time.Second
		}
	}
	for k, p := range map[string]*[]string{"FRAME_ANCESTORS": &c.FrameOrigins, "CORS_ORIGINS": &c.CORSOrigins} {
		if v := os.Getenv(k); v != "" {
			for _, origin := range strings.Split(v, ",") {
				origin = strings.TrimSpace(origin)
				u, e := url.Parse(origin)
				if e != nil || u.Host == "" || (u.Scheme != "https" && u.Scheme != "http") || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Path != "" {
					return c, fmt.Errorf("%s must contain exact http(s) origins without trailing slashes", k)
				}
				*p = append(*p, origin)
			}
		}
	}
	return c, nil
}
func env(k, fallback string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return fallback
}

type Server struct {
	cfg    Config
	store  Storage
	engine office.Engine
	assets fs.FS
	mu     sync.Mutex
}

func New(c Config, assets fs.FS) (*Server, error) {
	if c.MaxDocumentFields == 0 {
		c.MaxDocumentFields = office.DefaultMaxFields
	}
	if c.MaxDocumentFields < 1 || c.MaxDocumentFields > office.MaxDocumentFields {
		return nil, errors.New("invalid document capacity")
	}
	if c.Auth != "none" && c.Auth != "bearer" {
		return nil, errors.New("AUTH must be none or bearer")
	}
	if c.Auth == "bearer" && (len(c.APIKey) < 32 || len(c.SigningKey) < 32 || c.APIKey == c.SigningKey) {
		return nil, errors.New("bearer mode requires different API_KEY and SESSION_SECRET values, each at least 32 bytes")
	}
	if c.MaxUpload < 1 || c.MaxUpload > 128<<20 || c.MaxSessions < 1 || c.MaxSessions > 10000 || c.SessionTTL < time.Second || c.SessionTTL > 24*time.Hour || c.TokenTTL < time.Second || c.TokenTTL > time.Hour {
		return nil, errors.New("invalid upload, session, or TTL limits")
	}
	if c.SigningKey == "" {
		b := make([]byte, 32)
		if _, e := rand.Read(b); e != nil {
			return nil, e
		}
		c.SigningKey = hex.EncodeToString(b)
	}
	if e := os.MkdirAll(c.DataDir, 0700); e != nil {
		return nil, e
	}
	return &Server{cfg: c, store: DiskStore{c.DataDir}, engine: office.OOXML{MaxFields: c.MaxDocumentFields}, assets: assets}, nil
}
func (s *Server) Cleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, e := s.store.Sweep(time.Now()); e != nil {
		log.Printf("session cleanup failed: %v", e)
	}
}
func (s *Server) host(r *http.Request) bool {
	if s.cfg.Auth == "none" {
		return true
	}
	return subtle.ConstantTimeCompare([]byte(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")), []byte(s.cfg.APIKey)) == 1 && strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ")
}

type claims struct {
	ID   string `json:"sid"`
	Mode string `json:"mode"`
	Exp  int64  `json:"exp"`
}

func (s *Server) token(v *Session) (string, time.Time) {
	exp := time.Now().Add(s.cfg.TokenTTL)
	if exp.After(v.ExpiresAt) {
		exp = v.ExpiresAt
	}
	b, _ := json.Marshal(claims{v.ID, v.Mode, exp.Unix()})
	payload := base64.RawURLEncoding.EncodeToString(b)
	m := hmac.New(sha256.New, []byte(s.cfg.SigningKey))
	m.Write([]byte(payload))
	return payload + "." + base64.RawURLEncoding.EncodeToString(m.Sum(nil)), exp
}
func (s *Server) authorized(r *http.Request, v *Session, write bool) bool {
	if s.host(r) {
		return true
	}
	raw := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	parts := strings.Split(raw, ".")
	if len(parts) != 2 {
		return false
	}
	sig, e := base64.RawURLEncoding.DecodeString(parts[1])
	if e != nil {
		return false
	}
	m := hmac.New(sha256.New, []byte(s.cfg.SigningKey))
	m.Write([]byte(parts[0]))
	if !hmac.Equal(sig, m.Sum(nil)) {
		return false
	}
	b, e := base64.RawURLEncoding.DecodeString(parts[0])
	if e != nil {
		return false
	}
	var c claims
	if json.Unmarshal(b, &c) != nil {
		return false
	}
	return c.ID == v.ID && c.Mode == v.Mode && c.Exp > time.Now().Unix() && (!write || c.Mode == "edit")
}
func jsonResponse(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func fail(w http.ResponseWriter, status int, msg string) {
	jsonResponse(w, status, map[string]string{"error": msg})
}
func decode(w http.ResponseWriter, r *http.Request, max int64, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, max)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if e := d.Decode(v); e != nil {
		fail(w, 400, "invalid JSON body or body too large")
		return false
	}
	var extra any
	if d.Decode(&extra) != io.EOF {
		fail(w, 400, "expected one JSON object")
		return false
	}
	return true
}
func contains(items []string, s string) bool {
	for _, v := range items {
		if v == s {
			return true
		}
	}
	return false
}
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
	ancestors := "'self'"
	if len(s.cfg.FrameOrigins) > 0 {
		ancestors += " " + strings.Join(s.cfg.FrameOrigins, " ")
	}
	w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' blob:; connect-src 'self' https: http:; object-src 'none'; base-uri 'none'; form-action 'self'; frame-src 'self'; frame-ancestors "+ancestors)
	origin := r.Header.Get("Origin")
	if contains(s.cfg.CORSOrigins, origin) {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Vary", "Origin")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, If-Match")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Expose-Headers", "ETag, Content-Disposition")
	}
	if r.Method == "OPTIONS" {
		if !contains(s.cfg.CORSOrigins, origin) {
			fail(w, 403, "origin is not allowlisted")
			return
		}
		w.WriteHeader(204)
		return
	}
	if r.URL.Path == "/healthz" {
		jsonResponse(w, 200, map[string]string{"status": "ok"})
		return
	}
	if r.URL.Path == "/api/v1/config" && r.Method == "GET" {
		jsonResponse(w, 200, map[string]any{"auth": s.cfg.Auth, "maxUploadBytes": s.cfg.MaxUpload, "maxDocumentFields": s.cfg.MaxDocumentFields})
		return
	}
	if strings.HasPrefix(r.URL.Path, "/api/") {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.api(w, r)
		return
	}
	if r.Method != "GET" && r.Method != "HEAD" {
		fail(w, 405, "method not allowed")
		return
	}
	name := strings.TrimPrefix(r.URL.Path, "/")
	switch name {
	case "", "embed":
		name = "index.html"
	case "example":
		name = "example.html"
	}
	// Explicit allowlist prevents directory listings and accidental source serving.
	if !contains([]string{"index.html", "app.js", "style.css", "example.html", "example.js", "openapi.json"}, name) {
		fail(w, 404, "not found")
		return
	}
	b, e := fs.ReadFile(s.assets, name)
	if e != nil {
		fail(w, 404, "not found")
		return
	}
	w.Header().Set("Content-Type", mime.TypeByExtension(filepath.Ext(name)))
	if r.Method != "HEAD" {
		_, _ = w.Write(b)
	}
}
func (s *Server) api(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/api/v1/sessions" && r.Method == "POST" {
		if !s.host(r) {
			fail(w, 401, "host API authentication required")
			return
		}
		var req struct {
			Filename string `json:"filename"`
			Mode     string `json:"mode"`
		}
		if !decode(w, r, 4096, &req) {
			return
		}
		if req.Mode == "" {
			req.Mode = "edit"
		}
		if req.Mode != "view" && req.Mode != "edit" {
			fail(w, 400, "mode must be view or edit")
			return
		}
		if len(req.Filename) == 0 || len(req.Filename) > 240 || strings.ContainsAny(req.Filename, "/\\\r\n\x00") {
			fail(w, 400, "invalid filename")
			return
		}
		format := strings.TrimPrefix(strings.ToLower(filepath.Ext(req.Filename)), ".")
		if !contains([]string{"docx", "xlsx", "pptx"}, format) {
			fail(w, 415, "supported formats: DOCX, XLSX, PPTX")
			return
		}
		count, e := s.store.Sweep(time.Now())
		if e != nil {
			fail(w, 500, "storage unavailable")
			return
		}
		if count >= s.cfg.MaxSessions {
			fail(w, 503, "session limit reached")
			return
		}
		b := make([]byte, 24)
		if _, e = rand.Read(b); e != nil {
			fail(w, 500, "random source unavailable")
			return
		}
		v := &Session{ID: hex.EncodeToString(b), Filename: req.Filename, Format: format, Mode: req.Mode, ExpiresAt: time.Now().UTC().Add(s.cfg.SessionTTL), Events: []Event{}}
		v.event("created")
		if e = s.store.Put(v); e != nil {
			fail(w, 500, "could not persist session")
			return
		}
		s.sessionResponse(w, 201, v)
		return
	}
	tail := strings.TrimPrefix(r.URL.Path, "/api/v1/sessions/")
	parts := strings.Split(tail, "/")
	if len(parts) > 2 || !validID.MatchString(parts[0]) {
		fail(w, 404, "not found")
		return
	}
	v, e := s.store.Get(parts[0])
	if e != nil {
		fail(w, 404, "session not found")
		return
	}
	if !time.Now().Before(v.ExpiresAt) {
		_ = s.store.Delete(v.ID)
		fail(w, 410, "session expired")
		return
	}
	if !s.authorized(r, v, false) {
		fail(w, 401, "valid session token or host key required")
		return
	}
	action := ""
	if len(parts) == 2 {
		action = parts[1]
	}
	switch {
	case action == "" && r.Method == "GET":
		jsonResponse(w, 200, map[string]any{"id": v.ID, "filename": v.Filename, "format": v.Format, "mode": v.Mode, "revision": v.Revision, "expiresAt": v.ExpiresAt, "uploaded": len(v.Data) > 0})
	case action == "" && r.Method == "DELETE":
		if !s.host(r) {
			fail(w, 403, "host authentication required")
			return
		}
		if e = s.store.Delete(v.ID); e != nil {
			fail(w, 500, "could not delete session")
			return
		}
		w.WriteHeader(204)
	case action == "token" && r.Method == "POST":
		if !s.host(r) {
			fail(w, 403, "host authentication required")
			return
		}
		s.sessionResponse(w, 200, v)
	case action == "file" && r.Method == "PUT":
		if !s.host(r) {
			fail(w, 403, "only host may upload source bytes")
			return
		}
		if len(v.Data) > 0 {
			fail(w, 409, "source already uploaded; create another session")
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, s.cfg.MaxUpload)
		b, e := io.ReadAll(r.Body)
		if e != nil {
			fail(w, 413, "upload exceeds configured limit")
			return
		}
		if _, e = s.engine.Open(b, v.Format); e != nil {
			fail(w, 422, e.Error())
			return
		}
		v.Data = b
		v.Revision = 1
		v.event("uploaded")
		if !s.commit(w, v) {
			return
		}
		jsonResponse(w, 200, map[string]int{"revision": v.Revision})
	case action == "document" && r.Method == "GET":
		if len(v.Data) == 0 {
			fail(w, 409, "upload a source file first")
			return
		}
		model, e := s.engine.Open(v.Data, v.Format)
		if e != nil {
			fail(w, 422, e.Error())
			return
		}
		w.Header().Set("ETag", fmt.Sprintf(`"%d"`, v.Revision))
		mode := v.Mode
		if model.ReadOnly {
			mode = "view"
		}
		jsonResponse(w, 200, map[string]any{"revision": v.Revision, "filename": v.Filename, "mode": mode, "model": model})
	case action == "document" && r.Method == "PATCH":
		if v.Mode != "edit" || !s.authorized(r, v, true) {
			fail(w, 403, "view sessions cannot edit")
			return
		}
		if len(v.Data) == 0 {
			fail(w, 409, "upload a source file first")
			return
		}
		var req struct {
			Revision int           `json:"revision"`
			Edits    []office.Edit `json:"edits"`
		}
		if !decode(w, r, s.cfg.MaxUpload, &req) {
			return
		}
		if req.Revision != v.Revision {
			fail(w, 409, "revision conflict: reopen before editing")
			return
		}
		if len(req.Edits) == 0 {
			fail(w, 400, "edits must not be empty")
			return
		}
		b, e := s.engine.Apply(v.Data, v.Format, req.Edits)
		if e != nil {
			if errors.Is(e, office.ErrReadOnly) {
				fail(w, 403, e.Error())
				return
			}
			fail(w, 422, e.Error())
			return
		}
		if int64(len(b)) > s.cfg.MaxUpload {
			fail(w, 413, "saved file exceeds upload limit")
			return
		}
		v.Data = b
		v.Revision++
		v.event("saved")
		if !s.commit(w, v) {
			return
		}
		jsonResponse(w, 200, map[string]int{"revision": v.Revision})
	case action == "file" && r.Method == "GET":
		if len(v.Data) == 0 {
			fail(w, 409, "upload a source file first")
			return
		}
		v.event("downloaded")
		if !s.commit(w, v) {
			return
		}
		w.Header().Set("Content-Type", map[string]string{"docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document", "xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", "pptx": "application/vnd.openxmlformats-officedocument.presentationml.presentation"}[v.Format])
		w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": v.Filename}))
		w.Header().Set("ETag", fmt.Sprintf(`"%d"`, v.Revision))
		_, _ = w.Write(v.Data)
	case action == "events" && r.Method == "GET":
		after := 0
		if raw := r.URL.Query().Get("after"); raw != "" {
			after, e = strconv.Atoi(raw)
			if e != nil || after < 0 {
				fail(w, 400, "invalid event cursor")
				return
			}
		}
		events := []Event{}
		for _, ev := range v.Events {
			if ev.Sequence > after {
				events = append(events, ev)
			}
		}
		oldest := v.Sequence + 1
		if len(v.Events) > 0 {
			oldest = v.Events[0].Sequence
		}
		jsonResponse(w, 200, map[string]any{"events": events, "cursor": v.Sequence, "truncated": after < oldest-1})
	case action == "events" && r.Method == "POST":
		var req struct {
			Type string `json:"type"`
		}
		if !decode(w, r, 1024, &req) {
			return
		}
		if req.Type != "closed" {
			fail(w, 400, "only closed may be posted by clients")
			return
		}
		v.event("closed")
		if !s.commit(w, v) {
			return
		}
		w.WriteHeader(204)
	default:
		fail(w, 404, "unknown endpoint or method")
	}
}
func (s *Server) commit(w http.ResponseWriter, v *Session) bool {
	if e := s.store.Put(v); e != nil {
		fail(w, 500, "could not persist session")
		return false
	}
	return true
}
func (s *Server) sessionResponse(w http.ResponseWriter, status int, v *Session) {
	token, exp := s.token(v)
	jsonResponse(w, status, map[string]any{"id": v.ID, "token": token, "tokenExpiresAt": exp.UTC(), "expiresAt": v.ExpiresAt, "embedUrl": "/embed#session=" + v.ID + "&token=" + token, "mode": v.Mode})
}
