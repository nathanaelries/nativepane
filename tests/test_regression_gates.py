"""Mutation checks prove visual and semantic comparators detect changes."""
import tempfile
import unittest
import xml.etree.ElementTree as E
from pathlib import Path
from PIL import Image
from visual_helpers import visual_regression
from corpus_helpers import verify_edit, W


class RegressionGates(unittest.TestCase):
    def test_visual_gate_rejects_one_pixel_dimensions_and_missing_baseline(self):
        root = Path(__file__).resolve().parent.parent / "tmp"
        root.mkdir(exist_ok=True)
        with tempfile.TemporaryDirectory(dir=root) as directory:
            output = Path(directory)
            actual, expected, diff = output / 'actual.png', output / 'expected.png', output / 'diff.png'
            Image.new('RGB', (8, 8), (255, 255, 255)).save(actual)
            self.assertIn('missing', visual_regression(actual, expected, diff))
            Image.open(actual).save(expected)
            self.assertIsNone(visual_regression(actual, expected, diff))
            changed = Image.open(actual)
            changed.putpixel((3, 3), (254, 255, 255))
            changed.save(actual)
            self.assertIn('pixels', visual_regression(actual, expected, diff))
            self.assertTrue(diff.is_file())
            Image.new('RGB', (9, 8)).save(actual)
            self.assertIn('dimensions', visual_regression(actual, expected, diff))

    def test_semantic_gate_allows_only_the_requested_run_edit(self):
        source = f'<w:p xmlns:w="{W}"><w:r><w:rPr><w:b/></w:rPr><w:t>Before</w:t></w:r></w:p>'
        valid = source.replace('<w:t>Before</w:t>', '<w:t xml:space="preserve">After</w:t>')
        verify_edit(E.fromstring(source), E.fromstring(valid), {'text':'Before'}, 'After', 'docx')
        invalid = valid.replace('<w:b/>', '<w:i/>')
        with self.assertRaises(AssertionError):
            verify_edit(E.fromstring(source), E.fromstring(invalid), {'text':'Before'}, 'After', 'docx')


if __name__ == '__main__':
    unittest.main()
