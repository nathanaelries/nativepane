# Verification record

Local verification on 2026-09-30, Windows, Go 1.26.2, headless Microsoft Edge:

- `go test ./...`: passed. DOCX/XLSX/PPTX package-preserving round trips;
  numeric/shared/empty-cell edits; rejected formulas, malicious paths, macros,
  signatures, invalid XML, invalid edit IDs/characters and an expanded ZIP bomb.
- API tests: passed. Persistence across server reconstruction, revision conflicts,
  host/session permissions, read-only mode, wrong/tampered/expired tokens,
  cross-session denial, upload/session limits, CORS/CSP and bounded events.
- `go vet ./...` and JavaScript syntax checks: passed.
- `go list -m all`: only the NativePane module. Go formatting check: clean.
- Static Linux/amd64 cross-build with CGO disabled: passed (9.7 MB unstripped).
  `docker compose config --quiet`: passed.
- `tests/container_smoke.py` against the **native executable**: passed for all
  three formats, including independent ZIP/XML parsing and lifecycle events.
- Independent authoring/reopening: python-docx, openpyxl, python-pptx passed for
  edited text and retained formatting, table/header, formula/second worksheet,
  and slide geometry/second slide, respectively.
- Headless browser: upload, measurable text edit, autosave, undo/redo, original
  format download and close passed for all three formats. Host iframe, saved-event
  polling, token-fragment removal, iframe reload and session deletion passed.
  No browser JavaScript exceptions.
- Browser screenshots of the home screen, document pages, grid, slide-text view
  and host embed were inspected. QA artifacts are local under ignored `tmp/qa/`.

Docker runtime validation is **blocked by the local environment**: Docker Desktop
reported that its Hyper-V VM could not allocate 2048 MB of RAM (0x800705AA).
No container build/run success is claimed. Dockerfile performs tests and vet in
its build stage; CI also builds and runs the image when a working Docker daemon
is available. The native Go executable is the same application source.

The local race-detector attempt was unavailable because the installed Go
environment has CGO disabled; the CI job enables its normal Linux toolchain and
runs `go test -race ./...`. No local race-detector success is claimed.

These checks establish a working constrained editor, not Microsoft Office layout
equivalence. Files were reopened with independent libraries; manual Word/Excel/
PowerPoint interoperability and broad adversarial/security testing remain future
work. See README.md for unsupported features.

## DOCX table regression verification, 2026-09-30

- Go tests cover body flow order, nested tables, empty cells, multiple paragraphs,
  content-control wrappers, grid widths, horizontal and vertical merges, merge
  termination, skipped grid columns, and direct shading/alignment. Editing two
  cells changes only their text XML spans; all other XML and package parts remain
  unchanged. The saved model reopens with the same table topology.
- `tests/browser_tables.py` independently authors a DOCX with python-docx. Browser
  assertions check actual cell positions/sizes, nested table ownership, merged
  spans, paragraph order, unique text rendering, undo/redo, autosave and download.
  python-docx reopens the result with both edits and original merges intact.
  Passed; screenshot inspected at ignored `tmp/qa/docx-tables.png`.
- Full Go tests/vet, both JavaScript syntax checks, standard HTTP smoke tests for
  all three formats, and the existing browser regression suite passed.
- Docker build was attempted again and is still blocked by Docker Desktop's
  startup error. HTTP checks ran against the updated native executable.

## DOCX formatting and pagination regression verification, 2026-09-30

- Added Go coverage for defaults, basedOn inheritance, direct run overrides,
  character styles, theme fonts, paragraph spacing/borders, cell padding/borders,
  preferred widths, repeating headers and final-section geometry. Cyclic styles,
  unsafe font names and out-of-range values remain bounded. Style/theme package
  parts remain byte-identical after native text edits.
- `tests/browser_layout.py` passed using an independently authored multi-page
  DOCX. It measures font sizes/weight/color, page dimensions, cell proportions,
  printable content height, table continuation and repeated read-only headers.
  Editable fields appear once; edit reflow keeps field identity/caret and the
  independently reopened download preserves source styles.
- The reported document was tested locally without adding its content to the
  public fixtures. It now renders in two pages, with the first table on page one,
  inherited title styling, body typography, source margins and preferred columns.
  This confirms the visible regression improved, not general Word fidelity.
- Full Go tests/vet, JavaScript syntax checks, table/general browser regressions,
  and HTTP smoke checks passed. Docker build remains blocked by Desktop startup.

The historical table note above described the earlier behavior. Current tables
continue at row boundaries; oversized paragraphs or vertically merged row groups
still expand their page. Browser-local font fallbacks, mixed sections and other
unsupported layout features can still produce different Office page boundaries.

## Large-document capacity regression verification, 2026-09-30

- Removed the hardcoded 20,000-field threshold. The default is now 250,000,
  configurable through MAX_DOCUMENT_FIELDS up to 1,000,000. XML parts remain
  bounded to 1,000,000 elements, depth 128 and the expanded package byte limit.
- Go tests open 25,001-field DOCX/XLSX/PPTX fixtures, edit beyond the former limit,
  reopen the output and verify that only the expected package member changes.
  A 25,001-run bulk DOCX patch passes with all edits preserved using single-pass
  XML replacement. Configuration bounds, exact capacity boundaries and retry
  after rejected API uploads are covered.
- tests/browser_large.py passed for an independently authored 50,000-cell XLSX
  and 25,001-run DOCX, including browser upload, edit, autosave, download, independent
  reopening, preservation of untouched package members, and reopening the same
  file. The XLSX live grid contained only its 1,000 visible cells. Combined test
  durations were approximately 5 seconds and 8 seconds respectively on this machine;
  these are local observations, not throughput guarantees.
- No runtime dependencies were added. Larger DOCX/PPTX views and full undo
  snapshots still consume browser memory; capacity is bounded, not unlimited.
- The initial GitHub CI run (commit aa062e9) passed Go race tests/vet, JavaScript
  checks, Docker build and container HTTP smoke tests. This establishes container
  validation in CI; local Docker Desktop still cannot start.

## Rendering and native-save hardening, 2026-09-30

- Twelve unchanged Word/Excel/PowerPoint files from the pinned MIT Open XML SDK
  corpus have source hashes, Office producer metadata, licensing provenance and
  whole-model semantic expectations in `tests/fixtures/manifest.json`. Office
  itself is not installed locally; provenance is upstream and metadata evidence,
  not a new manual save or fidelity certification.
- Fifteen Linux browser baselines were reviewed: twelve Office fixtures and
  three original supplemental layout/table/omission cases. They use the pinned
  Playwright QA container; missing baselines, any pixel difference or changed
  dimensions fail. Mutation tests verify rejection of a one-pixel difference and
  an unrelated run-property change. Captures normalize fonts, caret and pointer.
- Word restriction tests cover revisions, fields, controls and protection in
  body/settings/header/footer/notes. Both host and signed-session edit attempts
  return 403 without changing the document or emitting a saved event. Browser
  controls are disabled and a view-only reason is visible. Unsupported-content
  markers survive pagination and marker-only merged-cell continuations.
- XLSX tests retain edited-cell extensions and unknown metadata, merge ranges,
  formats, 1900/1904 epochs and formulas. Independent reopening verifies date,
  currency, percentage and merged-anchor edits. Nested extension content is not
  mistaken for editable cells or formulas.
- PPTX tests retain the slide XML skeleton, shapes, coordinates, images, themes,
  rich run properties and unchanged ZIP members; python-pptx independently
  reopens the edited file.
- Local Go tests/vet, JS syntax checks, comparator mutation tests, omission-marker
  browser checks and three-format native HTTP smoke passed. GitHub CI supplies
  Linux race, scratch image/container, independent-reader, large-document and
  exact visual regression checks. Local Docker build was attempted and still
  fails because Docker Desktop cannot start.
- No new runtime dependency or host endpoint was added. README/OpenAPI changes
  describe actual read-only/marker/403 behavior and improved cell preservation.
  Renderer/storage interfaces, signed sessions and native saves remain covered
  by the existing core/API/browser tests. Layout remains approximate.
