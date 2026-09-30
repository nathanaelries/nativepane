"""Optional browser regression test. Requires Playwright and Chromium or Edge.
Set BROWSER_EXECUTABLE to use a system browser instead of Playwright's Chromium.
"""
import os
from pathlib import Path
from playwright.sync_api import sync_playwright, expect

root = Path(__file__).resolve().parent.parent
out = root / "tmp" / "qa"
out.mkdir(parents=True, exist_ok=True)
base = os.environ.get("NATIVEPANE_URL", "http://127.0.0.1:8080")
with sync_playwright() as p:
    options = {"headless": True}
    if os.environ.get("BROWSER_EXECUTABLE"):
        options["executable_path"] = os.environ["BROWSER_EXECUTABLE"]
    browser = p.chromium.launch(**options)
    page = browser.new_page(viewport={"width": 1440, "height": 1050})
    errors = []
    page.on("pageerror", lambda e: errors.append(str(e)))
    page.goto(base)
    expect(page.get_by_role("heading", name="A native place for Office documents.")).to_be_visible()
    page.screenshot(path=str(out / "welcome.png"), full_page=True)
    for extension in ("docx", "xlsx", "pptx"):
        page.locator("#upload").set_input_files(str(root / "samples" / ("welcome." + extension)))
        expect(page.locator("#filename")).to_have_text("welcome." + extension)
        field = page.locator("[data-field]").first
        before = field.input_value() if extension == "xlsx" else field.inner_text()
        field.fill("Browser verified edit")
        expect(page.locator("#save-status")).to_have_text("Saved · revision 2", timeout=15000)
        page.get_by_role("button", name="↶ Undo", exact=True).click()
        if extension == "xlsx":
            expect(field).to_have_value(before)
        else:
            expect(field).to_have_text(before)
        page.get_by_role("button", name="↷ Redo", exact=True).click()
        if extension == "xlsx":
            expect(field).to_have_value("Browser verified edit")
        else:
            expect(field).to_have_text("Browser verified edit")
        with page.expect_download() as download:
            page.get_by_role("button", name="↓ Download original format", exact=True).click()
        download.value.save_as(str(out / ("browser-saved." + extension)))
        page.screenshot(path=str(out / ("editor-" + extension + ".png")), full_page=True)
        page.get_by_role("button", name="Close pane", exact=True).click()
        expect(page.locator("#welcome")).to_be_visible()
        print(extension.upper() + ": browser upload/edit/autosave/undo/redo/download/close PASS")
    page.goto(base + "/example")
    page.locator("#host-file").set_input_files(str(root / "samples" / "welcome.docx"))
    page.get_by_role("button", name="Create pane", exact=True).click()
    frame = page.frame_locator("#pane")
    expect(frame.locator("#filename")).to_have_text("welcome.docx")
    # The credential fragment is removed, but reload retains session access in this tab.
    child = page.frames[1]
    assert "token=" not in child.url
    child.goto(child.url)
    expect(frame.locator("#filename")).to_have_text("welcome.docx")
    frame.locator("[data-field]").first.fill("Host embedded edit")
    expect(frame.locator("#save-status")).to_have_text("Saved · revision 2", timeout=15000)
    expect(page.locator("#events")).to_contain_text('"type":"saved"', timeout=10000)
    page.screenshot(path=str(out / "embed.png"), full_page=True)
    page.get_by_role("button", name="Delete session", exact=True).click()
    expect(page.locator("#host-status")).to_have_text("Session and stored bytes deleted.")
    assert not errors, errors
    print("Iframe, host event polling, deletion, no browser exceptions PASS")
    browser.close()
