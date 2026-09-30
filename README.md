```sh
docker run --rm -p 127.0.0.1:8080:8080 -v nativepane-data:/data nativepane:local
```

# NativePane

> Native Office documents in the browser. One container. MIT.  
> Native review without the native pain.

**First build the local image:** `docker build -t nativepane:local .`, then run
the command above. No prebuilt registry image is published by this repository.
Or run `docker compose up --build` for the same localhost-only demo.

## 60-second demo

1. Open **http://localhost:8080**.
2. Upload your `.docx`, `.xlsx`, or `.pptx` (small demo files are in `samples/`).
3. Edit existing text or an existing spreadsheet cell. Wait for **Saved**.
4. Click **Download original format**, then reopen it in your Office application.
5. Open **http://localhost:8080/example** to see a host upload a file, iframe the
   pane, poll saved/closed/downloaded events, and fetch the modified bytes.

This is a working, constrained first release—not a complete replacement for
Word, Excel, or PowerPoint. NativePane opens real OOXML ZIP packages and saves
changes inside them. It does not convert your files into an unrelated HTML format.
Original code is MIT; the Go runtime retains its permissive upstream notices.
There are no external application dependencies, paid engines, cloud accounts,
telemetry, CDNs, or license checks. See [LICENSING.md](LICENSING.md).

## What works and what does not

| Format | View/edit/save today | Fidelity limits |
|---|---|---|
| DOCX | Main-document text; document defaults and paragraph/character style inheritance; fonts, sizes, bold/italic, colors, underline, spacing, alignment and paragraph borders; page dimensions/margins; nested tables, merged cells, grid/preferred cell widths, cell fill/padding/borders; row pagination and repeated headers; edit existing run text | Browser font metrics can differ. Complex/conditional table styling, floating tables, legacy horizontal merges, mixed sections, within-row/paragraph splitting, images, headers/footers, lists, tabs, inline breaks, fields and footnotes are not rendered accurately. |
| XLSX | Worksheet order/names, existing cells, numbers, inline/shared strings, booleans; grid with row/column paging | No formula calculation; formulas and errors are read-only. Dependent cached results can become stale. Dates are serial numbers. No inserted rows/cells, formatting UI, merges, charts or pivot rendering. Shared-string edits become cell-local inline strings. |
| PPTX | Slides in presentation order; edit/save existing slide text | Text-only slide projection, not WYSIWYG. No master/theme/image/shape geometry, charts, animations, notes or rich-text layout. |
| PDF, DOC, XLS, PPT | Not supported in this release | Explicitly rejected; no hidden converter. |

Unsupported package parts are retained. Untouched ZIP members are raw-copied to
the output without recompression; edited XML is patched only at selected
text/cell spans. Changing a cell replaces that cell's content and preserves its
attributes (including style), but not unsupported children such as cell metadata
extensions. Text editing preserves run properties. No-op saves return original
bytes. Modified ZIP packages as a whole are not byte-identical or forensic originals.

**Track Changes is not supported:** tracked text can appear as ordinary text, and
editing it does not author a tracked revision. Content controls, field results and
document protection are not enforced. Do not use this version as an authoritative
review of hidden/deleted content or as a redaction tool. Keep evidentiary originals
in the host corpus and label this view as approximate. Signed, encrypted,
macro-enabled and strict-OOXML packages are unsupported. Macro/signature parts are
rejected when detected; embedded objects and external links are never executed or
fetched by NativePane. Preserved downloads can still contain active content in
unsupported parts; NativePane is not a file sanitizer.

Editing is limited to existing text runs/cells. Undo/redo keeps up to 100 browser
snapshots; autosave occurs after 900 ms of inactivity. Undo can undo a saved edit
and autosave its inverse. DOCX pagination uses the document's page size and margins,
and reflows after typing pauses or the browser resizes. Tables continue at row
boundaries and repeat designated header rows; vertically merged row groups stay
together. A single oversized paragraph or merged row group expands its page instead
of clipping text. Paragraph keep-with-next and page-break-before settings are
projected. Font names refer to fonts installed on the browser's OS, with local
fallbacks; no fonts are downloaded or bundled. Missing fonts, mixed sections and
unimplemented Word layout rules can still change page boundaries. Nested tables are supported through eight levels
and table grids through 256 columns. `samples/tables.docx` demonstrates nested and
merged cells. There is no simultaneous co-editing: stale revisions return 409 and
retain local unsaved text; reopen a fresh session to resolve the conflict manually.

## HTTP API and embedding

The stable, versioned contract is [web/openapi.json](web/openapi.json), served at
`/openapi.json`. A host backend performs these steps (bearer-mode example):

```sh
# API_KEY is a host backend secret. Never put it in an iframe URL.
curl -H "Authorization: Bearer $API_KEY" -H 'Content-Type: application/json' \
  -d '{"filename":"example.docx","mode":"edit"}' \
  http://localhost:8080/api/v1/sessions
# Response: id, token, embedUrl, tokenExpiresAt, expiresAt, mode

curl -X PUT -H "Authorization: Bearer $API_KEY" \
  -H 'Content-Type: application/octet-stream' --data-binary @example.docx \
  http://localhost:8080/api/v1/sessions/SESSION_ID/file

# Prefix embedUrl with your trusted NativePane origin and put it in an iframe.
# Fetch the possibly modified bytes:
curl -H "Authorization: Bearer $API_KEY" \
  http://localhost:8080/api/v1/sessions/SESSION_ID/file -o saved.docx
```

The source stream is buffered within configured limits for ZIP validation.
NativePane owns only an expiring session copy, not the corpus. The host must fetch
modified bytes before expiration. Upload occurs once; create a new session for a
new source. Delete with `DELETE /api/v1/sessions/{id}` when retention is no longer
needed. There is no corpus listing or arbitrary server-side URL fetch endpoint.

| Endpoint | Purpose |
|---|---|
| `POST /api/v1/sessions` | Host creates a view/edit session and gets signed access |
| `PUT /api/v1/sessions/{id}/file` | Host streams original bytes once |
| `GET /api/v1/sessions/{id}/document` | Get revision and editable model |
| `PATCH /api/v1/sessions/{id}/document` | Atomic `{revision, edits:[{id,text}]}` patch |
| `GET /api/v1/sessions/{id}/file` | Retrieve current Office bytes |
| `GET /api/v1/sessions/{id}/events?after=0` | Poll ordered lifecycle events |
| `POST /api/v1/sessions/{id}/events` | Report `{"type":"closed"}` |
| `POST /api/v1/sessions/{id}/token` | Host renews short-lived access |
| `GET /api/v1/sessions/{id}` | Inspect metadata |
| `DELETE /api/v1/sessions/{id}` | Host removes session and bytes |

HMAC-SHA256 tokens bind session ID, permission, and expiration. A view session
cannot be patched even with the host key. Fragment tokens do not reach HTTP access
logs; the browser removes the fragment, keeps the token in sessionStorage, and
uses an Authorization header. Treat embed URLs as credentials. The iframe must
be reopened with a renewed token after expiration; this version has no silent
token-renewal bridge. Tokens do not extend the session's lifetime. Deleting the
session revokes all of its tokens; individual token revocation is not implemented.

The server retains the last 256 events and reports `truncated` for an old cursor.
`saved` means the snapshot was committed; `downloaded` means bytes were requested,
not proof of a successful disk write. `closed` is a best-effort browser signal;
crashes/network loss can omit it. Polling is the reliable integration mechanism.
The iframe additionally sends `{source:'nativepane',sessionId,type,revision}` via
postMessage to its referrer's exact origin. Set iframe `referrerpolicy="origin"`
for these optional hints; validate both event.origin and event.source as shown
in [web/example.js](web/example.js). NativePane does not ship webhooks yet.

Standalone URL opening uses the browser's fetch with omitted credentials and
requires the remote source to support CORS. The host integration should stream
its own authenticated source bytes from its backend instead.

## Configuration and production

**AUTH=none is unsafe outside a trusted localhost demo.** Anyone who can reach
that server is a host administrator. Session tokens do not protect an open-mode
server. Keep the shown Docker host binding at `127.0.0.1`.

For production, put NativePane behind HTTPS, set `AUTH=bearer`, and supply two
different randomly generated secrets of at least 32 bytes. Keep `API_KEY` on the
host backend. Keep `SESSION_SECRET` stable across restarts and rotate it to revoke
all issued tokens. Do not serve multiple tenants with one shared API key unless
your host backend performs tenant authorization. Use a reverse proxy for TLS,
rate limits, and request logging without Authorization values. This first release
has not undergone a penetration test or broad Office interoperability certification.

| Variable | Default | Meaning |
|---|---|---|
| `PORT` | `8080` | HTTP port |
| `BIND_ADDR` | `127.0.0.1` locally; `0.0.0.0` in Docker | Listener address; Docker host port publication controls outside exposure |
| `DATA_DIR` | `data` locally; `/data` in Docker | Writable private session storage |
| `AUTH` | `none` | `none` or `bearer` |
| `API_KEY` | unset | Host bearer secret; required in bearer mode |
| `SESSION_SECRET` | random on startup in open mode | HMAC secret; required in bearer mode |
| `MAX_UPLOAD_BYTES` | `33554432` | Compressed file/patch limit; configurable through 128 MiB |
| `MAX_SESSIONS` | `100` | Maximum unexpired sessions |
| `SESSION_TTL_SECONDS` | `3600` | Session lifetime, maximum 86400 seconds |
| `TOKEN_TTL_SECONDS` | `600` | Token lifetime, maximum 3600 seconds, bounded by session expiry |
| `FRAME_ANCESTORS` | none beyond `'self'` | Comma-separated exact embedding origins, e.g. `https://review.example.com` |
| `CORS_ORIGINS` | none | Comma-separated exact browser API origins; no wildcards |

Origins must include the scheme and optional port, with no paths or trailing
slashes. CSP `frame-ancestors` permits only self plus FRAME_ANCESTORS. CORS does
not grant iframe permission and frame permission does not grant API access.
No credentials cookies are used. CSP permits browser HTTP(S) fetch for the URL
opener; this is not a server outbound-network requirement.

Session snapshots contain plaintext document bytes (base64 inside JSON). Use an
encrypted host volume if required; deletion is filesystem unlink, not secure
erasure. Expired sessions are inaccessible immediately and swept every minute or
on session creation. Budget disk for base64 overhead and a temporary second copy
during atomic saves. Snapshot writes are synced before rename; power-loss durability
of the directory rename depends on the host filesystem. One process owns one data
directory. Do not share it across replicas. Backups must obey your corpus policy.

Expanded ZIPs are capped at 128 MiB, 4,096 entries, 20,000 editable fields and XML
depth 128. The request/ZIP/model/snapshot pipeline is memory-buffered, and API
requests are serialized for correctness. Start with the Compose 1 GiB memory limit
and tune upload limits for actual workload; this is for modest session concurrency.

The default image is a static Go executable in `scratch`, running as UID 65532,
with no shell or Linux distribution. Ensure bind-mounted data is writable by that
UID. Named volumes work with the included directory. Build needs a cached or
downloadable Go build image; runtime is air-gapped. Export with `docker save` and
import with `docker load` to install without a registry connection.

## Development and verification

Go 1.26.2 is the default tested toolchain. No package install step is needed.

```sh
go run .
go test ./...
go vet ./...
go run ./cmd/notices       # exact standard-library inventory + upstream notices
go run ./cmd/samples       # regenerate original demo Office packages
```

CI also runs the race detector, builds the default Docker image and performs an
HTTP round trip against the container. `tests/container_smoke.py` uses only Python's
standard library. Optional `tests/independent_roundtrip.py` uses separately installed
python-docx, openpyxl and python-pptx to author real files, edit via HTTP, and reopen
the downloads in independent implementations. Optional `tests/browser_smoke.py`
uses Playwright and an installed Chromium browser to test upload, edit, autosave,
undo/redo, download and embedding. None of these Python packages ship in the image.
`tests/browser_tables.py` checks DOCX table layout, nested and merged cells, editing,
and independent reopening of the saved document using python-docx and Playwright.
`tests/browser_layout.py` checks inherited typography, page geometry, preferred
cell widths, multi-page tables, repeated headers, edit reflow and preserved styles.

See [ARCHITECTURE.md](ARCHITECTURE.md), [ROADMAP.md](ROADMAP.md), and
[THIRD_PARTY_LICENSES.md](THIRD_PARTY_LICENSES.md). The current test results and
environment limitations are recorded in [VERIFICATION.md](VERIFICATION.md).
