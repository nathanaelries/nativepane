# NativePane development

All original code is MIT. No external runtime dependency may be added without
reviewing its full transitive license set and updating the licensing documents.
Never add OnlyOffice/Collabora/LibreOffice or a copyleft converter to the image.

Go standard-library backend, plain browser JavaScript, no npm pipeline. The final
image is scratch with a CGO-disabled static executable and upstream notices.

Verify with `go test ./...`, `go vet ./...`, `node --check web/app.js`, and
`node --check web/example.js`. Go formatting is required. For runtime changes,
build Docker and run `python tests/container_smoke.py` against the local server.
Keep README fidelity claims in sync with code. Do not claim WYSIWYG layout.

The OOXML engine patches bounded XML byte spans and raw-copies unmodified ZIP
members. Never reserialize a whole package just to edit one text run. Add a
round-trip/preservation test for new editing behavior. Tokens and API keys must
never appear in logs, source, commits or query strings.

Use `go run ./cmd/notices` to regenerate the exact toolchain inventory, and
`go run ./cmd/samples` for original demo fixtures. Test outputs belong in tmp/.
