"""Pure XML comparison helpers shared by independent preservation tests."""
S = "http://schemas.openxmlformats.org/spreadsheetml/2006/main"
W = "http://schemas.openxmlformats.org/wordprocessingml/2006/main"
A = "http://schemas.openxmlformats.org/drawingml/2006/main"


def signature(node):
    return (node.tag, tuple(sorted(node.attrib.items())), node.text, node.tail,
            tuple(signature(child) for child in node))


def verify_edit(before, after, field, text, format):
    if format == "xlsx":
        a = next(c for c in before.iter("{" + S + "}c") if c.get("r") == field["address"])
        b = next(c for c in after.iter("{" + S + "}c") if c.get("r") == field["address"])
        for cell in (a, b):
            cell.attrib.pop("t", None)
            for child in list(cell):
                if child.tag in ("{" + S + "}v", "{" + S + "}is"):
                    cell.remove(child)
    else:
        tag = "{" + (W if format == "docx" else A) + "}t"
        a = next(n for n in before.iter(tag) if (n.text or "") == field["text"])
        a.text = text
        a.set("{http://www.w3.org/XML/1998/namespace}space", "preserve")
    assert signature(before) == signature(after), "unrelated XML structure/formatting changed"
