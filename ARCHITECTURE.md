# Architecture decision: preserve packages, ship one process

Status: accepted for the first working release, 2026-09-30.

## Stack and boundaries

Go's standard library supplies HTTP, ZIP, XML, JSON, cryptographic randomness,
HMAC signing, and files. A plain JavaScript client renders an explicit model.
There is no npm build, database, conversion service, or Office engine. One static
binary embeds the UI and OpenAPI document. The final Docker image uses scratch,
runs as an unprivileged numeric user, and mounts /data for temporary sessions.

The Engine interface has Open and Apply operations. The first implementation
indexes editable OOXML text or worksheet cells using XML token byte offsets.
Edits replace only the selected spans; unmodified XML is not reserialized. ZIP
entries outside those edits are raw-copied. This preserves unsupported package
parts instead of rebuilding a document from HTML. It does not guarantee semantic
fidelity: changed text can invalidate layout, signatures, cached formulas, field
results, or application-specific assumptions. Signed packages are rejected.

The browser receives an editable projection, never arbitrary document HTML.
DOCX displays paragraphs and flow tables on approximate pages, XLSX displays existing cells,
and PPTX displays each slide's text. Unsupported objects remain in the download
but are not necessarily visible. Text is inserted with DOM textContent/value.
This is deliberately a constrained editor, not a complete office suite.

DOCX flow is projected in body order as paragraphs or tables. Table rows/cells
retain nested blocks, grid widths and horizontal/vertical spans. A table block's
flat fields inventory remains available for API compatibility and browser undo;
the renderer uses its row/cell topology, so text is displayed only once. Table
cells use the same byte-span edit targets as ordinary runs. Document defaults,
paragraph/character basedOn chains, direct properties and theme font/color hints
are projected as bounded CSS hints. Page geometry comes from the final section.
Fonts use the browser OS with local fallbacks; font binaries are not bundled.
Tables split between row groups, retaining vertical merges and repeating designated
headers as read-only mirrors of the original fields. Browser edits reflow with
caret restoration. Oversized paragraphs/merged groups expand a page; splitting
inside them, mixed sections and full conditional table-style semantics remain future work.

## Sessions and host integration

A host backend authenticates with an API bearer secret, creates a session with
filename and view/edit mode, streams bytes, and passes the returned fragment-token
embed URL to an iframe. Signed HMAC tokens bind session, permission, and expiry.
The client removes the fragment from browser history and uses an Authorization
header. Tokens are never query parameters. The host can poll ordered events and
fetch the latest bytes; browser postMessage notifications are hints only.

The API never downloads arbitrary URLs. The standalone URL opener uses browser
fetch with CORS and omitted credentials; production hosts stream their own corpus
bytes. This avoids a server-side SSRF proxy and does not require corpus ownership.
The sample host is a local AUTH=none demonstration. Production secrets stay on
the integrating backend, never in iframe HTML or frontend configuration.

Each session is one atomically replaced JSON snapshot containing metadata,
base64 original/current bytes, revision and a bounded event log. A process mutex
serializes reads and commits. This favors correctness and easy crash recovery
over large-file throughput. There is one server process per data directory;
horizontal replication and network-filesystem durability are not supported.
Optimistic revision checks reject stale edits. A Storage interface isolates
persistence for future object-store support. Session expiration triggers cleanup;
hosts can explicitly delete sessions sooner. Expiration is enforced on reads.

## Security and operations

Bound compressed/uncompressed sizes, ZIP entry counts, XML depth, model size,
session count, request sizes, token lifetime, and event history. Reject encrypted,
macro-enabled, signed and legacy packages. Never fetch external relationships or
execute embedded objects/formulas/macros. Authenticated download is attachment-only.
Allowlisted CORS and CSP frame-ancestors are configured separately. Production
uses AUTH=bearer behind HTTPS. AUTH=none is a localhost demonstration only.

There is no outbound runtime network dependency, telemetry, or phone-home.
An air-gapped installation loads a prebuilt image and needs only a browser.
The final image includes third-party notices from the exact build toolchain.

## Sacrifices and future work

We prioritize preserved source packages and a testable integration contract over
pixel-perfect layout. No text insertion/deletion of structural elements, tracked
changes authoring, formula calculation, chart editing, image positioning, or
live co-editing in this release. Formatting is preserved in the file but only a
small subset is projected. Better renderers can implement Engine and a matching
browser adapter. Revisioned edit operations provide a future collaboration seam;
they are not a CRDT. See ROADMAP.md.
