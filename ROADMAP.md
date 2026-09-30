# NativePane roadmap

## First release: implemented

- One static Go server, embedded browser app, scratch Docker runtime.
- Real DOCX text, XLSX existing-cell and PPTX slide-text editing with native saves.
- DOCX flow tables, nested cells/tables, grid proportions and horizontal/vertical merges.
- DOCX document/paragraph/character styling, page geometry, preferred cell widths,
  row pagination, repeated headers and browser reflow after edits.
- Package-preserving OOXML adapter and independent renderer/storage interfaces.
- Browser autosave, undo/redo, original-format download and browser URL upload.
- Signed view/edit sessions, shared-secret host auth, optimistic save revisions.
- Host upload/stream, embed URL, event polling, byte retrieval and deletion.
- Configurable embed/CORS origins, package/request limits and session expiration.
- OpenAPI, local host example, MIT strategy, generated third-party notices and CI.

This release is an engineering starting point for a document pane. It is not an
office-suite feature-completeness or evidentiary-rendering claim.

## Next: rendering and hardening

- Office-authored interoperability fixture corpus; visual and semantic regression tests.
- More Word style rules, mixed sections, lists, full conditional table styling,
  floating tables, within-row/paragraph splitting, images and improved font/layout fidelity.
- Visible unsupported-content markers, field/control/protection awareness and a
  dedicated tracked-change display before permitting those documents to be edited.
- Preserve all cell extensions during edits; date/number formats and merged cells.
- Slide shapes, coordinates, images, themes and richer text.
- Better performance: cached projections, streaming storage and per-session locks.
- Storage quotas, rate limiting, cancellation, stronger crash-recovery tests.
- Token renewal bridge with explicit host origin validation; individual revocation.
- PDF viewing with a separately reviewed permissive dependency, if justified.

## Later: host integration and editing breadth

- Optional OIDC, per-host keys/scopes, audit retention and object-store adapter.
- Optional signed webhooks with retries and SSRF-safe destination registration.
- WOPI adapter after protocol and licensing review; preserve the simple HTTP API.
- Structural/rich formatting edits; formula calculation with explicit compatibility.
- Optional co-editing only after a document-operation model and conflict semantics
  are proven. Select a permissive CRDT implementation if needed.
- More formats via independently reviewed adapters. Legacy binary Office support
  is not promised through a copyleft conversion engine.

No paid or AGPL/GPL/SSPL engine is planned for the default path. Any optional
integration with different terms must be isolated, separately documented, and
excluded from the default image and recommended integration path.
