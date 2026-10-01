# NativePane license strategy

NativePane's original server, browser application, integration example, tests,
and documentation are MIT licensed. Commercial use and embedding are free.
Keep LICENSE and the third-party notices with redistributed copies. Replace
the copyright-holder placeholder before publishing under your own ownership.

MIT is the product's license, not a claim that somebody else's code has been
relicensed. Our only runtime dependency is Go's standard library and runtime,
under BSD-style and other permissive notices. There are no external Go modules,
JavaScript packages, web fonts, Office engines, paid SDKs, or CDN resources.
THIRD_PARTY_LICENSES.md inventories these components; the container carries the
actual toolchain's license and source notices under /licenses.

The repository also includes unmodified Office-saved regression documents from
the .NET Foundation's MIT Open XML SDK test corpus. Their exact source revision,
paths and SHA-256 hashes are recorded in tests/fixtures/manifest.json; upstream
LICENSE and NOTICE files are retained in tests/fixtures/licenses. These are test
data, not an Office engine or SDK dependency, and do not ship in the runtime image.
The expanded corpus also uses unmodified Excel test documents from ClosedXML
(MIT) and two Word test/sample documents from docx4j (Apache-2.0). Their pinned
source revisions, exact license texts and notices are retained alongside the
fixtures. Apache-2.0 is permissive and applies to that test data; we do not
relicense it MIT. No ClosedXML/docx4j code, renderer, Java/.NET library or its
transitive dependencies is imported or shipped. The original product and the
default runtime dependency inventory remain unchanged.
No Microsoft Office installation, subscription or proprietary converter is
required to build or run the corpus checks.

The default image is FROM scratch with a static, CGO-disabled executable and
notices. It contains no Linux distribution, libc, shell, package manager, or
copyleft office engine. The intermediate Go build image is build tooling only,
not a layer in the shipped image. Docker and the user's browser are independently
obtained tools, not bundled product dependencies.

OnlyOffice Document Server, Collabora, LibreOffice, AGPL/GPL/SSPL engines and
commercial renderers are not included or recommended. No optional engine is
currently shipped. Any future adapter must undergo a separate dependency and
license review; its addition must not silently change the default image.

Contribution rule: adding any dependency requires its exact version, upstream
license text, transitive-license inventory, and an update to this file. CI rejects
external Go modules. The build collects source copyright/license notices from
the Go toolchain, including its vendored standard-library dependencies.

This is a small, honest implementation of OOXML, not a license workaround for a
full office suite. We accept reduced layout fidelity to preserve a permissive
default product. See README.md for specific limits.
