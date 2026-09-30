# Rendering and native-save regression corpus

The twelve files under `office/` are unmodified, pinned documents from the
.NET Foundation's MIT Open XML SDK test assets. `manifest.json` records source
revision/path, SHA-256, license, Office producer/version, edit policy and baseline
paths. Upstream LICENSE and NOTICE are retained under `licenses/`; no SDK code is
linked. Office-saved provenance is supported by their upstream conformance-test
location and unchanged `docProps/app.xml` metadata. We do not claim a fresh manual
save/validation in installed Office on this machine. Never relabel generated or
mutated files as unmodified Office-authored fixtures.

## Gates and coverage

| Behavior | Authoritative checks |
|---|---|
| Word font/style/page projection | Whole-model Office snapshots; Go defaults/theme/inheritance tests; measured `browser_layout.py` and reviewed pixel baseline |
| Nested/merged tables, order, widths, pagination, headers and reflow | Go topology/preservation tests; independent `browser_tables.py`/`browser_layout.py`; their pixel baselines |
| Omitted Word constructs are visible | Inline/block markers; `hardening_test.go` covers drawings/text boxes, lists, notes, tabs, breaks, alternate content and flow omissions; browser snapshots |
| Revisions, fields, controls or protection cannot edit | Four actual restricted Office files; Go variants/settings/header/note checks; host/session bypass tests; browser disabled controls; API 403 and unchanged bytes |
| XLSX order, existing cells, formulas | Whole-model Office snapshots; Go cell tests; independent readers and browser checks |
| XLSX extensions, formats, epochs and merges survive | Go byte-preservation tests; Office styles/date-workbook XML and reader checks; supplemental 1900/1904 dates, currency/percentage/merge/formula fixtures in `preservation_cases.py` |
| PPTX order, text edits, geometry/media/themes/rich runs | Office shapes/text/image snapshots; exact slide XML skeleton and raw ZIP-copy checks; independent reader; rich-run/image supplemental fixture |
| Autosave, undo/redo, downloads, embed and sessions | `browser_smoke.py`, Go server tests, container HTTP smoke |
| Large files | Go 25,001-field cases for all formats plus bulk edit; browser 50,000-cell XLSX and 25,001-run DOCX; independent reopen and preserved members |
| MIT default runtime | stdlib-only module gate, notices generator, scratch Docker build and container smoke |

Supplemental fixtures are independently authored by our QA scripts and stay in
`tmp/`. They cover typography, long/nested tables, unknown extensions, date epochs
and rich slide text more precisely than the small historical Office examples.

## Running and reviewing

`go test ./...` checks semantic baselines and native preservation offline, with no
external tools. Mismatches write actual models to `tmp/fixtures/`. Tests never
overwrite expected baselines.

CI runs independent readers and browser gates inside the digest-pinned Playwright
Python 1.60.0 Ubuntu Noble image using `tests/qa-requirements.txt`. The viewport,
device scale, locale and timezone are fixed. Strict pixel comparisons have zero
tolerance; missing baselines fail too. Office layout equivalence is not asserted:
screenshots protect NativePane's reviewed approximate rendering. CI uploads actual
renders and pixel diffs on failure. All test outputs stay under `tmp/`.

Against a running server, run `corpus_semantic.py`, `preservation_cases.py`,
`browser_smoke.py`, `browser_tables.py`, `browser_layout.py`, `browser_unsupported.py`, `browser_large.py`,
then `browser_corpus.py`, using Python with the pinned QA requirements installed.
`test_regression_gates.py` deliberately mutates one pixel and a run property to
prove that the visual and semantic comparators reject regressions.
Use the CI image for authoritative pixels; another OS/browser can have different
font metrics and still generate useful local comparison artifacts.

For intentional changes, inspect the source, semantic diff and CI actual/diff
screenshots, run behavior/preservation gates, then copy only reviewed artifacts
to manifest baseline paths and commit them with the implementation. Require the
next normal CI run to be green. Tests never automatically accept new baselines.
Source updates must retain provenance, licenses and hashes. New layout/edit work
requires explicit semantic/edit/visual coverage. No proprietary Office app or
converter is required; no engine, WOPI, OIDC, webhook, co-edit, formula calculator
or PDF renderer is added to the product or the corpus.
