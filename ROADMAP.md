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
- Configurable large-document capacity, paged spreadsheet grids, opening progress
  and single-pass bulk XML patches; regression tests beyond 20,000 fields.
- OpenAPI, local host example, MIT strategy, generated third-party notices and CI.
- Office-saved fixture manifest, semantic/visual regression gates, explicit omitted
  Word-content markers and engine-enforced view-only restrictions.
- Preserved edited-cell extensions/date formats/merges and slide geometry/media/
  themes/rich properties, with independent reader and package-preservation checks.

This release is an engineering starting point for a document pane. It is not an
office-suite feature-completeness or evidentiary-rendering claim.

## Next: rendering and hardening

- Expand the Office-authored interoperability corpus and supported-layout coverage.
- More Word style rules, mixed sections, lists, full conditional table styling,
  floating tables, within-row/paragraph splitting, images and improved font/layout fidelity.
- Dedicated tracked-change/field/control/protection displays before enabling edits.
- Display date/number formats and merged cells; extend preserved-cell semantics.
- Slide shapes, coordinates, images, themes and richer text.
- Better performance: virtualized document/slide pages, sparse undo, cached projections,
  streaming storage and per-session locks.
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
