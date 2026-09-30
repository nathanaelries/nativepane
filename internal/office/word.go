package office

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
)

// Direct-child properties avoid accidentally borrowing formatting from a nested
// table, a revision's old properties, or another cell.
func (n *node) child(ns, name string) *node {
	if n != nil {
		for _, c := range n.children {
			if c != nil && c.name.Space == ns && c.name.Local == name {
				return c
			}
		}
	}
	return nil
}

func wordProperty(n *node, name string) string {
	if c := n.child(wordNS, name); c != nil {
		return c.attr("val")
	}
	return ""
}

func boundedNumber(raw string, fallback, min, max int) int {
	n, err := strconv.Atoi(raw)
	if err != nil || n < min || n > max {
		return fallback
	}
	return n
}

func (p *projection) wordBlocks(part string) error {
	root, err := parse(p.parts[part])
	if err != nil {
		return err
	}
	body := root.child(wordNS, "document").child(wordNS, "body")
	if body == nil {
		return errors.New("unsupported Word document namespace or missing body")
	}
	if err := p.wordSafety(part, root); err != nil {
		return err
	}
	p.word, err = loadWordStyles(p.parts)
	if err != nil {
		return err
	}
	p.model.Page = wordPage(body)
	p.model.Blocks, err = p.wordFlow(part, body, 0)
	if err != nil {
		return err
	}
	for _, kind := range []struct{ prefix, label string }{{"word/header", "Header"}, {"word/footer", "Footer"}, {"word/footnotes.xml", "Footnotes"}, {"word/endnotes.xml", "Endnotes"}, {"word/comments", "Comments"}} {
		for name := range p.parts {
			if strings.HasPrefix(name, kind.prefix) && strings.HasSuffix(name, ".xml") {
				p.model.Blocks = append(p.model.Blocks, wordMarker(kind.label+" content not rendered"))
				break
			}
		}
	}
	return err
}

// Walk flow containers in source order, stopping at paragraphs/tables. Descending
// through a table here would both flatten cells and index their text twice.
func (p *projection) wordFlow(part string, container *node, depth int) ([]Block, error) {
	blocks := []Block{}
	for _, n := range container.children {
		var block Block
		var err error
		switch {
		case n.name.Space == wordNS && n.name.Local == "p":
			block, err = p.wordParagraph(part, n)
		case n.name.Space == wordNS && n.name.Local == "tbl":
			block, err = p.wordTable(part, n, depth+1)
		default:
			if n.name.Space == wordNS {
				switch n.name.Local {
				case "tcPr", "sectPr", "bookmarkStart", "bookmarkEnd", "proofErr":
					continue
				case "altChunk":
					blocks = append(blocks, wordMarker("Embedded alternate content not rendered"))
					continue
				case "sdt", "ins", "del", "moveFrom", "moveTo":
					blocks = append(blocks, wordMarker(wordRestriction(n)+" not fully displayed"))
				case "sdtPr", "sdtEndPr", "customXmlPr":
					continue
				case "sdtContent", "customXml": // Transparent flow container.
				default:
					blocks = append(blocks, wordMarker("Unsupported Word content: "+n.name.Local))
				}
			} else if n.name.Local == "AlternateContent" {
				blocks = append(blocks, wordMarker("Alternate content not rendered"))
				continue
			} else if reason := wordRestriction(n); reason != "" {
				blocks = append(blocks, wordMarker(reason+" not fully displayed"))
			}
			// Content controls, custom XML and tracked wrappers can contain flow.
			var nested []Block
			nested, err = p.wordFlow(part, n, depth)
			blocks = append(blocks, nested...)
			if err != nil {
				return nil, err
			}
			continue
		}
		if err != nil {
			return nil, err
		}
		blocks = append(blocks, block)
	}
	return blocks, nil
}

func (p *projection) wordParagraph(part string, para *node) (Block, error) {
	b := Block{Label: "Paragraph", Kind: "paragraph", Fields: []Field{}}
	b.Markers = paragraphMarkers(para)
	b.Style, b.KeepNext, b.BreakBefore = p.word.paragraphStyle(para)
	props := para.child(wordNS, "pPr")
	id := wordProperty(props, "pStyle")
	if id == "" {
		id = p.word.defaultParagraph
	}
	for _, style := range p.word.chain(id) {
		if style.child(wordNS, "pPr").child(wordNS, "numPr") != nil && !hasMarker(b.Markers, "List numbering not rendered") {
			b.Markers = append(b.Markers, "List numbering not rendered")
		}
	}
	switch wordProperty(para.child(wordNS, "pPr"), "jc") {
	case "center":
		b.Align = "center"
	case "right", "end":
		b.Align = "right"
	case "both", "distribute":
		b.Align = "justify"
	}
	for _, t := range paragraphTexts(para) {
		owner := t.parent
		for owner != nil && !(owner.name.Space == wordNS && owner.name.Local == "p") {
			owner = owner.parent
		}
		if owner != para {
			continue // Nested text-box paragraphs are not part of this paragraph.
		}
		f := Field{Text: t.text, Kind: "text", Style: textStyle(b.Style)}
		props := t.parent.child(wordNS, "rPr")
		for _, style := range p.word.chain(wordProperty(props, "rStyle")) {
			p.word.run(f.Style, style.child(wordNS, "rPr"))
		}
		p.word.run(f.Style, props)
		f.Bold, f.Italic = f.Style["fontWeight"] == "bold", f.Style["fontStyle"] == "italic"
		if err := p.add(&b, part, t, f); err != nil {
			return Block{}, err
		}
	}
	return b, nil
}

// Rows/cells can be wrapped in content controls. Stop at another structural
// boundary so nested tables never contribute rows/cells to their parent.
func tableChildren(n *node, name string) (out []*node) {
	for _, c := range n.children {
		if c.name.Space == wordNS {
			if c.name.Local == name {
				out = append(out, c)
				continue
			}
			switch c.name.Local {
			case "tbl", "tr", "tc", "p":
				continue
			}
		}
		out = append(out, tableChildren(c, name)...)
	}
	return
}

var wordFill = regexp.MustCompile(`^[0-9a-fA-F]{6}$`)

func (p *projection) wordTable(part string, tbl *node, depth int) (Block, error) {
	if depth > 8 {
		return Block{}, errors.New("nested table depth exceeds 8")
	}
	b := Block{Label: "Table", Kind: "table", Fields: []Field{}, Rows: []TableRow{}}
	if tbl.child(wordNS, "tblPr").child(wordNS, "tblpPr") != nil {
		b.Markers = append(b.Markers, "Floating table positioning not rendered")
	}
	if grid := tbl.child(wordNS, "tblGrid"); grid != nil {
		for _, col := range grid.children {
			if col.name.Space == wordNS && col.name.Local == "gridCol" {
				b.ColumnWidths = append(b.ColumnWidths, boundedNumber(col.attr("w"), 1, 1, 100000))
			}
		}
	}
	if len(b.ColumnWidths) > 256 {
		return Block{}, errors.New("table exceeds 256 grid columns")
	}
	// A vertical merge may continue only in the next row at the same grid column
	// and with the same horizontal span. A normal cell closes that merge.
	active := map[int]*TableCell{}
	preferredWidths := map[int]int{}
	for rowIndex, tr := range tableChildren(tbl, "tr") {
		row := TableRow{Cells: []*TableCell{}, RepeatHeader: wordEnabled(tr.child(wordNS, "trPr").child(wordNS, "tblHeader"))}
		column := boundedNumber(wordProperty(tr.child(wordNS, "trPr"), "gridBefore"), 0, 0, 256)
		next := map[int]*TableCell{}
		for _, tc := range tableChildren(tr, "tc") {
			props := tc.child(wordNS, "tcPr")
			span := boundedNumber(wordProperty(props, "gridSpan"), 1, 1, 256)
			// Preferred cell widths can refine a uniform tblGrid, a common pattern
			// in generated DOCX files. Unmerged cells define each column once.
			if width := props.child(wordNS, "tcW"); width != nil && width.attr("type") == "dxa" && span == 1 {
				if value, ok := numberAttr(width, "w", 1, 100000); ok && preferredWidths[column] == 0 {
					preferredWidths[column] = value
				}
			}
			if column+span > 256 {
				return Block{}, errors.New("table exceeds 256 grid columns")
			}
			blocks, err := p.wordFlow(part, tc, depth)
			if err != nil {
				return Block{}, err
			}
			for _, block := range blocks {
				// Retain the existing flat fields contract for API consumers and the
				// editor's snapshot/undo logic. Render using rows, not this inventory.
				b.Fields = append(b.Fields, block.Fields...)
			}
			cell := &TableCell{Column: column, ColSpan: span, RowSpan: 1, Blocks: blocks, Style: p.word.cellStyle(tbl, tc, rowIndex, column)}
			if shading := props.child(wordNS, "shd"); shading != nil && wordFill.MatchString(shading.attr("fill")) {
				cell.Fill = shading.attr("fill")
			}
			switch wordProperty(props, "vAlign") {
			case "center", "bottom":
				cell.VAlign = wordProperty(props, "vAlign")
			}
			merge := props.child(wordNS, "vMerge")
			if merge != nil && merge.attr("val") != "restart" {
				if anchor := active[column]; anchor != nil && anchor.ColSpan == span {
					anchor.RowSpan++
					// Conforming continuations are empty; keep any nonempty continuation
					// content visible/editable in the merged anchor rather than losing it.
					for _, block := range blocks {
						if len(block.Fields) > 0 || block.Kind == "table" || block.Kind == "unsupported" || len(block.Markers) > 0 {
							anchor.Blocks = append(anchor.Blocks, block)
						}
					}
					next[column] = anchor
					column += span
					continue
				}
			}
			row.Cells = append(row.Cells, cell)
			if merge != nil {
				next[column] = cell
			}
			column += span
		}
		column += boundedNumber(wordProperty(tr.child(wordNS, "trPr"), "gridAfter"), 0, 0, 256)
		if column > 256 {
			return Block{}, errors.New("table exceeds 256 grid columns")
		}
		for len(b.ColumnWidths) < column {
			b.ColumnWidths = append(b.ColumnWidths, 1)
		}
		b.Rows = append(b.Rows, row)
		active = next
	}
	for column, width := range preferredWidths {
		if column < len(b.ColumnWidths) {
			b.ColumnWidths[column] = width
		}
	}
	return b, nil
}
