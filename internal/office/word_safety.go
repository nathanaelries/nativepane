package office

import (
	"fmt"
	"sort"
	"strings"
)

func wordRestriction(n *node) string {
	if n.name.Space == "http://schemas.microsoft.com/office/word/2010/wordml" && (n.name.Local == "conflictIns" || n.name.Local == "conflictDel") {
		return "Tracked changes"
	}
	if n.name.Space != wordNS {
		return ""
	}
	switch n.name.Local {
	case "ins", "del", "delText", "delInstrText", "moveFrom", "moveTo",
		"moveFromRangeStart", "moveFromRangeEnd", "moveToRangeStart", "moveToRangeEnd",
		"customXmlInsRangeStart", "customXmlInsRangeEnd", "customXmlDelRangeStart", "customXmlDelRangeEnd",
		"customXmlMoveFromRangeStart", "customXmlMoveFromRangeEnd", "customXmlMoveToRangeStart", "customXmlMoveToRangeEnd",
		"rPrChange", "pPrChange", "tblPrChange", "tblPrExChange", "tblGridChange", "trPrChange", "tcPrChange", "sectPrChange",
		"numberingChange", "cellIns", "cellDel", "cellMerge":
		return "Tracked changes"
	case "trackRevisions":
		if wordEnabled(n) {
			return "Tracked changes"
		}
	case "fldSimple", "fldChar", "instrText":
		return "Fields"
	case "sdt", "ffData":
		return "Content controls"
	case "documentProtection", "writeProtection", "permStart", "permEnd":
		return "Document protection"
	}
	return ""
}

func walkWord(n *node, visit func(*node)) {
	visit(n)
	for _, child := range n.children {
		walkWord(child, visit)
	}
}

// Inspect all Word XML parts, including settings, headers, notes and glossary.
// A host edit token never overrides restrictions embedded in the document.
func (p *projection) wordSafety(mainPart string, root *node) error {
	reasons := map[string]bool{}
	inspect := func(n *node) {
		if reason := wordRestriction(n); reason != "" {
			reasons[reason] = true
		}
	}
	walkWord(root, inspect)
	names := make([]string, 0, len(p.parts))
	for name := range p.parts {
		if strings.HasPrefix(name, "word/") && strings.HasSuffix(name, ".xml") && name != mainPart {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	for _, name := range names {
		n, err := parse(p.parts[name])
		if err != nil {
			return fmt.Errorf("invalid Word part %s: %w", name, err)
		}
		walkWord(n, inspect)
	}
	for _, reason := range []string{"Tracked changes", "Fields", "Content controls", "Document protection"} {
		if reasons[reason] {
			p.model.ReadOnly = true
			p.model.Warnings = append(p.model.Warnings, reason+": view only. Dedicated display is not implemented; editing is disabled for the entire document.")
		}
	}
	return nil
}

func wordMarker(label string) Block {
	return Block{Kind: "unsupported", Label: label, Fields: []Field{}}
}

func hasMarker(markers []string, label string) bool {
	for _, marker := range markers {
		if marker == label {
			return true
		}
	}
	return false
}

func paragraphMarkers(para *node) []string {
	seen := map[string]bool{}
	var markers []string
	add := func(label string) {
		if label != "" && !seen[label] {
			seen[label] = true
			markers = append(markers, label)
		}
	}
	var walk func(*node)
	walk = func(n *node) {
		if n != para && n.name.Space == wordNS && n.name.Local == "p" {
			return
		}
		if reason := wordRestriction(n); reason != "" {
			add(reason + " not fully displayed")
		}
		if n.name.Space == wordNS {
			switch n.name.Local {
			case "drawing", "pict", "object":
				add("Image, drawing or embedded object not rendered")
				return
			case "txbxContent":
				add("Text box not rendered")
				return
			case "footnoteReference", "endnoteReference":
				add("Footnote/endnote reference not rendered")
			case "commentReference", "commentRangeStart":
				add("Comment not rendered")
			case "numPr":
				add("List numbering not rendered")
			case "tab", "ptab":
				add("Tab positioning not rendered")
			case "br", "cr", "lastRenderedPageBreak":
				add("Inline line/page break not rendered")
			case "sym":
				add("Symbol not rendered")
			case "noBreakHyphen", "softHyphen":
				add("Special hyphen not rendered")
			case "ruby":
				add("Ruby annotation not rendered")
				return
			case "subDoc":
				add("Linked subdocument not rendered")
				return
			case "sectPr":
				add("Section layout change not rendered")
			case "altChunk":
				add("Embedded alternate content not rendered")
				return
			case "vanish", "webHidden":
				if wordEnabled(n) {
					add("Hidden text is present; visibility is approximate")
				}
			}
		}
		if n.name.Space == "http://schemas.openxmlformats.org/officeDocument/2006/math" && (n.name.Local == "oMath" || n.name.Local == "oMathPara") {
			add("Equation not rendered")
			return
		}
		if n.name.Local == "AlternateContent" {
			add("Alternate content not rendered")
			return
		}
		for _, child := range n.children {
			walk(child)
		}
	}
	walk(para)
	return markers
}

func paragraphTexts(para *node) (texts []*node) {
	var visit func(*node)
	visit = func(n *node) {
		if n != para && n.name.Space == wordNS {
			switch n.name.Local {
			case "p", "drawing", "pict", "object", "txbxContent", "altChunk":
				return
			case "t":
				texts = append(texts, n)
				return
			}
		}
		if n.name.Local == "AlternateContent" {
			return
		}
		for _, child := range n.children {
			visit(child)
		}
	}
	visit(para)
	return
}
