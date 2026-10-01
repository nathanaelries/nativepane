"""Fail on visual or browser-policy regressions; never auto-update baselines.
CI uses the digest-pinned Playwright image in ci.yml. Local runs on other OSes
can produce comparison artifacts, but must not replace the Linux baselines.
"""
import json
import os
import xml.etree.ElementTree as E
from pathlib import Path
from zipfile import ZipFile

from playwright.sync_api import sync_playwright, expect
from visual_helpers import visual_regression
from corpus_helpers import verify_edit

root = Path(__file__).resolve().parent.parent
fixtures = root / "tests" / "fixtures"
out = root / "tmp" / "visual"
out.mkdir(parents=True, exist_ok=True)
manifest = json.loads((fixtures / "manifest.json").read_text())
failures = []


def compare(actual, expected, name):
    problem = visual_regression(actual, expected, out / (name + "-diff.png"))
    if problem:
        failures.append(name + ": " + problem)


with sync_playwright() as p:
    options = {"headless": True}
    if os.environ.get("BROWSER_EXECUTABLE"):
        options["executable_path"] = os.environ["BROWSER_EXECUTABLE"]
    browser = p.chromium.launch(**options)
    context = browser.new_context(viewport={"width": 1440, "height": 1100}, device_scale_factor=1, locale="en-US", timezone_id="UTC", reduced_motion="reduce")
    page = context.new_page()
    page.set_default_timeout(120000)
    errors = []
    page.on("pageerror", lambda error: errors.append(str(error)))
    page.goto(os.environ.get("NATIVEPANE_URL", "http://127.0.0.1:8080"))
    for fixture in manifest["fixtures"]:
        page.locator("#upload").set_input_files(str(fixtures / fixture["file"]))
        expect(page.locator("#filename")).to_have_text(Path(fixture["file"]).name)
        expect(page.locator("#save-status")).to_have_text("Saved · revision 1")
        if fixture["readOnly"]:
            expect(page.locator("#mode")).to_have_text("View only")
            expect(page.locator("#restriction-note")).to_be_visible()
            expect(page.locator("#save")).to_be_disabled()
            assert page.locator('[data-field][contenteditable="plaintext-only"]').count() == 0
            assert page.locator("input[data-field]:not([readonly])").count() == 0
        else:
            expect(page.locator("#restriction-note")).to_be_hidden()
            expect(page.locator("#save")).to_be_enabled()
        if fixture['format'] != 'xlsx':
            expect(page.locator('[data-field]')).to_have_count(fixture['projectedFields'])
            ids = page.locator('[data-field]').evaluate_all('(els) => els.map(e => e.dataset.field)')
            assert len(set(ids)) == len(ids), fixture['id'] + ': duplicate projected text'
        await_fonts = "async () => { await document.fonts.ready; document.activeElement.blur(); }"
        page.evaluate(await_fonts)
        page.mouse.move(0, 0)
        actual = out / (fixture["id"] + ".png")
        page.locator("#workspace").screenshot(path=str(actual), animations="disabled", caret="hide")
        compare(actual, fixtures / fixture["visual"], fixture["id"])
        for view in fixture.get('views', []):
            page.locator('#outline button').nth(view['sheet']).click()
            for _ in range(view['rowPage']):
                page.get_by_role('button', name='Rows →', exact=True).click()
            for _ in range(view['colPage']):
                page.get_by_role('button', name='Columns →', exact=True).click()
            page.evaluate(await_fonts)
            page.mouse.move(0, 0)
            name = fixture['id'] + '.' + view['id']
            actual = out / (name + '.png')
            page.locator('#workspace').screenshot(path=str(actual), animations='disabled', caret='hide')
            compare(actual, fixtures / view['visual'], name)
        if fixture.get('layoutProbe'):
            probe = fixture['layoutProbe']
            field = page.locator('[data-field="' + probe['field'] + '"]')
            field.fill(probe['text'])
            expect(page.locator('#save-status')).to_have_text('Saved · revision 2')
            # Reflow must keep every original edit target exactly once, and
            # expanded row groups plus repeated read-only header mirrors visible.
            expect(page.locator('[data-field]')).to_have_count(fixture['projectedFields'])
            expect(page.locator('[data-field="' + probe['field'] + '"]')).to_have_text(probe['text'])
            assert page.locator('[data-mirror-field]').count() > 0, 'Office header continuation was not repeated'
            assert page.locator('.page-shell').count() > 1
            with page.expect_download() as downloaded:
                page.locator('#download').click()
            saved = out / (fixture['id'] + '.edited.docx')
            downloaded.value.save_as(str(saved))
            model = json.loads((fixtures / fixture['semantic']).read_text())
            original_field = next(f for b in model['blocks'] for f in b['fields'] if f['id'] == probe['field'])
            metrics = field.evaluate('''(field) => ({
                pageHeight: field.closest('.page').offsetHeight,
                fieldBottom: field.getBoundingClientRect().bottom,
                contentBottom: field.closest('.page-content').getBoundingClientRect().bottom
            })''')
            assert metrics['pageHeight'] > model['page']['height'] / 15, 'oversized row was not expanded'
            assert metrics['fieldBottom'] <= metrics['contentBottom'] + 1, 'edited text was clipped'
            assert page.locator('[data-mirror-field][contenteditable="plaintext-only"]').count() == 0
            with ZipFile(fixtures / fixture['file']) as before, ZipFile(saved) as after:
                assert before.namelist() == after.namelist()
                for member in before.namelist():
                    if member != 'word/document.xml':
                        assert before.read(member) == after.read(member), member
                verify_edit(E.fromstring(before.read('word/document.xml')), E.fromstring(after.read('word/document.xml')), original_field, probe['text'], 'docx')
            page.evaluate(await_fonts)
            page.mouse.move(0, 0)
            actual = out / (fixture['id'] + '.edited.png')
            page.locator('#workspace').screenshot(path=str(actual), animations='disabled', caret='hide')
            compare(actual, fixtures / probe['visual'], fixture['id'] + '.edited')
        page.locator("#close").click()
        expect(page.locator("#welcome")).to_be_visible()
        print(fixture["id"] + ": screenshot and browser edit policy checked")
    assert not errors, errors
    browser.close()

# Rich typography, nested/merged tables and continuation-page coverage comes
# from independently authored supplemental fixtures, not mislabeled Office files.
for name, actual in (("word-layout", root / "tmp/qa/layout-regression.png"), ("word-tables", root / "tmp/qa/docx-tables.png"), ("word-unsupported", root / "tmp/qa/unsupported-regression.png")):
    assert actual.is_file(), f"Run browser_layout.py/browser_tables.py before {name}"
    compare(actual, fixtures / "visual" / (name + ".png"), name)

if failures:
    raise AssertionError("\n".join(failures))
print("Office corpus and supplemental visual baselines PASS")
