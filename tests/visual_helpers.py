"""Strict, independently testable screenshot comparison; no baseline updates."""
from PIL import Image, ImageChops


def visual_regression(actual, expected, diff_path):
    if not expected.is_file():
        return "missing reviewed visual baseline"
    a, b = Image.open(actual).convert("RGB"), Image.open(expected).convert("RGB")
    if a.size != b.size:
        return f"image dimensions changed {b.size} -> {a.size}"
    diff = ImageChops.difference(a, b)
    if diff.getbbox() is not None:
        diff.save(diff_path)
        return "rendered pixels changed; inspect actual/expected/diff"
    return None
