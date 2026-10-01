# CI regression proof

These are deliberate, isolated fixture-baseline mutations. The source Office
files remain unchanged. Neither proof branch is a release branch or a candidate
for merging. Both use the ordinary workflow with no skipped or weakened gates.

## Positive controls

The semantic proof's parent `73d1d699b7b80b1895d06f39b151210d8dcb1d93`
[passed full CI](https://github.com/nathanaelries/nativepane/actions/runs/36797702660).
The final isolated-capture candidate `499a00ed73be74be0183e4ca337f79fa12a74ec9`
also [passed both jobs](https://github.com/nathanaelries/nativepane/actions/runs/36798898006).
This includes Go race/vet/format checks, the scratch Docker image and container
HTTP smoke, all 49 independent semantic/native-save cases, browser behavior,
large-file cases, 68 manifest visual views and three supplemental baselines.

## Semantic negative control

Branch `interop-proof-semantic`, commit
`288f7865dc9da54f86eec35b0319a1add1cadc7b`, changes only the first `bold: true`
expectation to `false` in `expected/word-fonts.docx.json`.
[CI failed](https://github.com/nathanaelries/nativepane/actions/runs/36797787563):

- `TestOfficeFixtureCorpus/word-fonts`: `semantic regression`.
- The independent HTTP gate: `word-fonts: model differs from reviewed semantic baseline`.

Source hashes, field counts and provenance still pass. This proves a real model
property regression cannot be hidden by those inventory checks.

## Visual negative control

Branch `interop-proof-visual-isolated`, commit
`afa0f10c2dcb1976ce516e8ccc38acd83e1e28ca`, changes one RGB component of only
pixel `(950, 950)` in `visual/word-fonts.docx.png`. Dimensions and all other
pixels are unchanged. Its parent is the green isolated-capture candidate above.
[CI failed](https://github.com/nathanaelries/nativepane/actions/runs/36798927717):

- The `verify` job passed, including Go race tests and Docker/container checks.
- All semantic, native-save, browser-policy and large-document assertions passed.
- The sole visual failure was `word-fonts: rendered pixels changed`.

No pixel tolerance, screenshot mask, automatic baseline update or workflow bypass
was introduced. Per-source browser contexts prevent earlier UI interactions from
contaminating the next capture. All screenshots still include the workspace.

## Release branch

Only the valid corpus, reviewed baselines, tests and documentation are merged.
The deliberately corrupt proof commits stay on their named branches. The
[main workflow](https://github.com/nathanaelries/nativepane/actions/workflows/ci.yml?query=branch%3Amain)
runs the same complete gates. This is regression coverage for NativePane's
approximate layout, not a certification of Microsoft Office visual fidelity.
