"""Fail on visual or browser-policy regressions; never auto-update baselines.
CI uses the digest-pinned Playwright image in ci.yml. Local runs on other OSes
can produce comparison artifacts, but must not replace the Linux baselines.
"""
import json
import os
from pathlib import Path

from PIL import Image, ImageChops
from playwright.sync_api import sync_playwright, expect

root = Path(__file__).resolve().parent.parent
fixtures = root / "tests" / "fixtures"
out = root / "tmp" / "visual"
out.mkdir(parents=True, exist_ok=True)
manifest = json.loads((fixtures / "manifest.json").read_text())
failures = []


def compare(actual, expected, name):
    if not expected.is_file():
        failures.append(name + ": missing reviewed visual baseline")
        return
    a, b = Image.open(actual).convert("RGB"), Image.open(expected).convert("RGB")
    if a.size != b.size:
        failures.append(f"{name}: image dimensions changed {b.size} -> {a.size}")
        return
    diff = ImageChops.difference(a, b)
    if diff.getbbox() is not None:
        diff.save(out / (name + "-diff.png"))
        failures.append(name + ": rendered pixels changed; inspect actual/expected/diff")


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
        await_fonts = "async () => { await document.fonts.ready; document.activeElement.blur(); }"
        page.evaluate(await_fonts)
        actual = out / (fixture["id"] + ".png")
        page.locator("#workspace").screenshot(path=str(actual), animations="disabled", caret="hide")
        compare(actual, fixtures / fixture["visual"], fixture["id"])
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
