// Package office projects a bounded OOXML package without rebuilding its XML.
package office

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const MaxExpanded = 128 << 20
const DefaultMaxFields = 250000
const MaxDocumentFields = 1000000
const wordNS = "http://schemas.openxmlformats.org/wordprocessingml/2006/main"
const sheetNS = "http://schemas.openxmlformats.org/spreadsheetml/2006/main"
const drawNS = "http://schemas.openxmlformats.org/drawingml/2006/main"
const relNS = "http://schemas.openxmlformats.org/officeDocument/2006/relationships"

var ErrReadOnly = errors.New("editing is disabled for this document until its restricted Word constructs have a dedicated display")

type Field struct {
	ID       string            `json:"id"`
	Text     string            `json:"text"`
	Address  string            `json:"address,omitempty"`
	Kind     string            `json:"kind,omitempty"`
	ReadOnly bool              `json:"readOnly,omitempty"`
	Bold     bool              `json:"bold,omitempty"`
	Italic   bool              `json:"italic,omitempty"`
	Style    map[string]string `json:"style,omitempty"`
}
type Block struct {
	Label        string            `json:"label"`
	Fields       []Field           `json:"fields"`
	Kind         string            `json:"kind,omitempty"`
	Align        string            `json:"align,omitempty"`
	ColumnWidths []int             `json:"columnWidths,omitempty"`
	Rows         []TableRow        `json:"rows,omitempty"`
	Style        map[string]string `json:"style,omitempty"`
	KeepNext     bool              `json:"keepNext,omitempty"`
	BreakBefore  bool              `json:"breakBefore,omitempty"`
	Markers      []string          `json:"markers,omitempty"`
}
type TableRow struct {
	Cells        []*TableCell `json:"cells"`
	RepeatHeader bool         `json:"repeatHeader,omitempty"`
}
type TableCell struct {
	Column  int               `json:"column"`
	ColSpan int               `json:"colSpan"`
	RowSpan int               `json:"rowSpan"`
	Blocks  []Block           `json:"blocks"`
	Fill    string            `json:"fill,omitempty"`
	VAlign  string            `json:"vAlign,omitempty"`
	Style   map[string]string `json:"style,omitempty"`
}
type PageLayout struct {
	Width  int `json:"width"`
	Height int `json:"height"`
	Top    int `json:"top"`
	Right  int `json:"right"`
	Bottom int `json:"bottom"`
	Left   int `json:"left"`
}
type Model struct {
	Format   string      `json:"format"`
	Blocks   []Block     `json:"blocks"`
	Warnings []string    `json:"warnings"`
	Page     *PageLayout `json:"page,omitempty"`
	ReadOnly bool        `json:"readOnly,omitempty"`
}
type Edit struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}
type Engine interface {
	Open([]byte, string) (Model, error)
	Apply([]byte, string, []Edit) ([]byte, error)
}
type OOXML struct{ MaxFields int }

func (o OOXML) fieldLimit() int {
	if o.MaxFields > 0 && o.MaxFields <= MaxDocumentFields {
		return o.MaxFields
	}
	return DefaultMaxFields
}

type node struct {
	name                        xml.Name
	attrs                       []xml.Attr
	start, inner, endStart, end int
	text                        string
	children                    []*node
	parent                      *node
}

func (n *node) attr(name string) string {
	for _, a := range n.attrs {
		if a.Name.Local == name {
			return a.Value
		}
	}
	return ""
}
func (n *node) all(ns, name string) (out []*node) {
	for _, c := range n.children {
		if c.name.Local == name && (ns == "" || c.name.Space == ns) {
			out = append(out, c)
		}
		out = append(out, c.all(ns, name)...)
	}
	return
}
func parse(data []byte) (*node, error) {
	d := xml.NewDecoder(bytes.NewReader(data))
	root := &node{}
	stack := []*node{root}
	count := 0
	for {
		before := int(d.InputOffset())
		tok, err := d.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("invalid XML: %w", err)
		}
		switch v := tok.(type) {
		case xml.StartElement:
			count++
			if len(stack) > 128 || count > 1000000 {
				return nil, errors.New("XML complexity limit exceeded")
			}
			p := stack[len(stack)-1]
			n := &node{name: v.Name, attrs: v.Attr, start: before, inner: int(d.InputOffset()), parent: p}
			p.children = append(p.children, n)
			stack = append(stack, n)
		case xml.EndElement:
			n := stack[len(stack)-1]
			n.endStart = before
			n.end = int(d.InputOffset())
			stack = stack[:len(stack)-1]
		case xml.CharData:
			stack[len(stack)-1].text += string(v)
		case xml.Directive:
			return nil, errors.New("XML directives are unsupported")
		}
	}
	if len(root.children) != 1 || len(stack) != 1 {
		return nil, errors.New("XML must have one root")
	}
	return root, nil
}

type target struct {
	part  string
	n     *node
	field Field
}
type projection struct {
	model     Model
	targets   map[string]target
	parts     map[string][]byte
	archive   *zip.Reader
	word      *wordStyles
	maxFields int
}

func readPackage(data []byte, format string) (*projection, error) {
	if format != "docx" && format != "xlsx" && format != "pptx" {
		return nil, errors.New("only DOCX, XLSX and PPTX are supported")
	}
	z, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, errors.New("not an unencrypted Office ZIP package")
	}
	if len(z.File) > 4096 {
		return nil, errors.New("ZIP entry limit exceeded")
	}
	p := &projection{archive: z, parts: map[string][]byte{}, targets: map[string]target{}, model: Model{Format: format, Blocks: []Block{}, Warnings: []string{"Approximate view. Unsupported objects are preserved in the file but may not be displayed."}}}
	var total uint64
	for _, f := range z.File {
		if _, ok := p.parts[f.Name]; ok {
			return nil, errors.New("duplicate ZIP entry")
		}
		if path.Clean(f.Name) != strings.TrimSuffix(f.Name, "/") || strings.HasPrefix(f.Name, "/") || strings.Contains(f.Name, "\\") || strings.HasPrefix(f.Name, "../") {
			return nil, errors.New("unsafe ZIP path")
		}
		low := strings.ToLower(f.Name)
		if strings.Contains(low, "vbaproject") || strings.HasPrefix(low, "_xmlsignatures/") {
			return nil, errors.New("macros and signed packages are unsupported")
		}
		if f.UncompressedSize64 > MaxExpanded || total+f.UncompressedSize64 > MaxExpanded {
			return nil, errors.New("expanded package exceeds 128 MiB")
		}
		total += f.UncompressedSize64
		r, e := f.Open()
		if e != nil {
			return nil, e
		}
		b, e := io.ReadAll(io.LimitReader(r, MaxExpanded+1))
		r.Close()
		if e != nil {
			return nil, e
		}
		if uint64(len(b)) != f.UncompressedSize64 {
			return nil, errors.New("invalid ZIP size")
		}
		p.parts[f.Name] = b
	}
	ct := p.parts["[Content_Types].xml"]
	if len(ct) == 0 || bytes.Contains(bytes.ToLower(ct), []byte("macroenabled")) {
		return nil, errors.New("missing content types or macro-enabled package")
	}
	if _, err := parse(ct); err != nil {
		return nil, err
	}
	required := map[string]string{"docx": "word/document.xml", "xlsx": "xl/workbook.xml", "pptx": "ppt/presentation.xml"}[format]
	if len(p.parts[required]) == 0 {
		return nil, errors.New("filename does not match package format")
	}
	return p, nil
}
func (p *projection) add(block *Block, part string, n *node, f Field) error {
	if len(p.targets) >= p.maxFields {
		return errors.New("This document exceeds this server's document capacity. Ask the administrator to increase MAX_DOCUMENT_FIELDS, or open a smaller document.")
	}
	f.ID = fmt.Sprintf("%s:%d", part, len(p.targets))
	f.ReadOnly = f.ReadOnly || p.model.ReadOnly
	block.Fields = append(block.Fields, f)
	p.targets[f.ID] = target{part, n, f}
	return nil
}
func textOf(n *node, ns string) string {
	var b strings.Builder
	for _, t := range n.all(ns, "t") {
		b.WriteString(t.text)
	}
	return b.String()
}
func (p *projection) textBlocks(part, ns string, slide bool, label string) error {
	root, err := parse(p.parts[part])
	if err != nil {
		return err
	}
	combined := Block{Label: label, Fields: []Field{}}
	for i, para := range root.all(ns, "p") {
		b := Block{Label: fmt.Sprintf("Paragraph %d", i+1), Fields: []Field{}}
		for _, t := range para.all(ns, "t") {
			f := Field{Text: t.text, Kind: "text"}
			r := t.parent
			if ns == wordNS {
				for _, x := range r.all(ns, "b") {
					f.Bold = x.attr("val") != "0" && x.attr("val") != "false"
				}
				for _, x := range r.all(ns, "i") {
					f.Italic = x.attr("val") != "0" && x.attr("val") != "false"
				}
			}
			if err := p.add(&b, part, t, f); err != nil {
				return err
			}
		}
		if slide {
			combined.Fields = append(combined.Fields, b.Fields...)
		} else {
			p.model.Blocks = append(p.model.Blocks, b)
		}
	}
	if slide {
		p.model.Blocks = append(p.model.Blocks, combined)
	}
	return nil
}
func (p *projection) orderedParts(main, tag, relType string) ([][2]string, error) {
	root, e := parse(p.parts[main])
	if e != nil {
		return nil, e
	}
	if main == "xl/workbook.xml" && root.child(sheetNS, "workbook") == nil {
		return nil, errors.New("unsupported spreadsheet namespace (strict OOXML is not supported)")
	}
	relFile := path.Join(path.Dir(main), "_rels", path.Base(main)+".rels")
	rels, e := parse(p.parts[relFile])
	if e != nil {
		return nil, e
	}
	mapping := map[string]string{}
	for _, r := range rels.all("", "Relationship") {
		if r.attr("TargetMode") == "External" {
			continue
		}
		if !strings.HasSuffix(r.attr("Type"), "/"+relType) {
			continue
		}
		target := r.attr("Target")
		if strings.HasPrefix(target, "/") {
			target = strings.TrimPrefix(target, "/")
		} else {
			target = path.Join(path.Dir(main), target)
		}
		mapping[r.attr("Id")] = target
	}
	var out [][2]string
	for _, n := range root.all("", tag) {
		id := ""
		for _, a := range n.attrs {
			if a.Name.Space == relNS && a.Name.Local == "id" {
				id = a.Value
			}
		}
		target := mapping[id]
		if target == "" || p.parts[target] == nil {
			return nil, errors.New("missing or unsupported Office relationship")
		}
		out = append(out, [2]string{target, n.attr("name")})
	}
	return out, nil
}

var cellAddress = regexp.MustCompile(`^[A-Z]{1,3}[1-9][0-9]{0,6}$`)

func project(data []byte, format string, maxFields int) (*projection, error) {
	p, e := readPackage(data, format)
	if e != nil {
		return nil, e
	}
	p.maxFields = maxFields
	switch format {
	case "docx":
		e = p.wordBlocks("word/document.xml")
	case "pptx":
		var parts [][2]string
		parts, e = p.orderedParts("ppt/presentation.xml", "sldId", "slide")
		if e != nil {
			break
		}
		for i, part := range parts {
			e = p.textBlocks(part[0], drawNS, true, fmt.Sprintf("Slide %d", i+1))
			if e != nil {
				break
			}
		}
		p.model.Warnings = append(p.model.Warnings, "Text-only slide projection: images, themes, geometry, and animations are not rendered.")
	case "xlsx":
		var shared []string
		if raw := p.parts["xl/sharedStrings.xml"]; raw != nil {
			var r *node
			r, e = parse(raw)
			if e != nil {
				break
			}
			for _, si := range r.all(sheetNS, "si") {
				shared = append(shared, textOf(si, sheetNS))
			}
		}
		var parts [][2]string
		parts, e = p.orderedParts("xl/workbook.xml", "sheet", "worksheet")
		if e != nil {
			break
		}
		for _, part := range parts {
			var r *node
			r, e = parse(p.parts[part[0]])
			if e != nil {
				break
			}
			if r.child(sheetNS, "worksheet") == nil {
				e = errors.New("unsupported worksheet namespace (strict OOXML is not supported)")
				break
			}
			b := Block{Label: part[1], Fields: []Field{}}
			for _, c := range worksheetCells(r) {
				address := c.attr("r")
				if !cellAddress.MatchString(address) {
					e = errors.New("unsupported cell address")
					break
				}
				f := Field{Address: address, Kind: "text"}
				if value := c.child(sheetNS, "v"); value != nil {
					f.Text = value.text
				}
				if formula := c.child(sheetNS, "f"); formula != nil {
					f.Text = "=" + formula.text
					f.ReadOnly = true
					f.Kind = "formula"
				} else {
					switch c.attr("t") {
					case "s":
						var idx int
						idx, e = strconv.Atoi(f.Text)
						if e != nil || idx < 0 || idx >= len(shared) {
							e = errors.New("invalid shared string reference")
							break
						}
						f.Text = shared[idx]
					case "inlineStr":
						if inline := c.child(sheetNS, "is"); inline != nil {
							f.Text = textOf(inline, sheetNS)
						}
					case "", "n":
						f.Kind = "number"
					case "b":
						f.Kind = "boolean"
					case "e":
						f.ReadOnly = true
					}
				}
				if e != nil {
					break
				}
				e = p.add(&b, part[0], c, f)
				if e != nil {
					break
				}
			}
			if e != nil {
				break
			}
			p.model.Blocks = append(p.model.Blocks, b)
		}
		p.model.Warnings = append(p.model.Warnings, "Existing cells only. Formulas are read-only; cached results are not recalculated. Dates appear as serial values. Pivot tables and charts are not rendered.")
	}
	if e != nil {
		return nil, e
	}
	if len(p.model.Blocks) == 0 {
		return nil, errors.New("no supported content found (strict OOXML is not supported)")
	}
	return p, nil
}
func (o OOXML) Open(data []byte, format string) (Model, error) {
	p, e := project(data, format, o.fieldLimit())
	if e != nil {
		return Model{}, e
	}
	return p.model, nil
}
func escaped(s string) string {
	var b bytes.Buffer
	_ = xml.EscapeText(&b, []byte(s))
	return b.String()
}

var typeAttr = regexp.MustCompile(`\s+t\s*=\s*(?:"[^"]*"|'[^']*')`)
var spaceAttr = regexp.MustCompile(`\s+xml:space\s*=\s*(?:"[^"]*"|'[^']*')`)

type replacement struct {
	start, end int
	value      string
}

func (o OOXML) Apply(data []byte, format string, edits []Edit) ([]byte, error) {
	p, e := project(data, format, o.fieldLimit())
	if e != nil {
		return nil, e
	}
	if len(edits) == 0 {
		return data, nil
	}
	if p.model.ReadOnly {
		return nil, ErrReadOnly
	}
	if len(edits) > p.maxFields {
		return nil, errors.New("too many edits")
	}
	changes := map[string][]replacement{}
	seen := map[string]bool{}
	for _, edit := range edits {
		t, ok := p.targets[edit.ID]
		if !ok || seen[edit.ID] {
			return nil, errors.New("unknown or duplicate field")
		}
		seen[edit.ID] = true
		if t.field.ReadOnly {
			return nil, errors.New("field is read-only")
		}
		if len(edit.Text) > 1<<20 {
			return nil, errors.New("field text too long")
		}
		for _, r := range edit.Text {
			if r < 32 && r != '\t' && r != '\n' && r != '\r' || r == 0xfffe || r == 0xffff {
				return nil, errors.New("invalid XML character")
			}
		}
		n := t.n
		raw := p.parts[t.part]
		start := string(raw[n.start:n.inner])
		qname := strings.Fields(strings.TrimPrefix(start, "<"))[0]
		qname = strings.TrimRight(qname, "/>")
		var value string
		if format == "xlsx" {
			var err error
			value, err = patchCell(raw, t, edit.Text)
			if err != nil {
				return nil, err
			}
		} else {
			start = spaceAttr.ReplaceAllString(start, "")
			start = strings.TrimSuffix(strings.TrimSuffix(start, ">"), "/")
			value = start + ` xml:space="preserve">` + escaped(edit.Text) + `</` + qname + `>`
		}
		changes[t.part] = append(changes[t.part], replacement{n.start, n.end, value})
	}
	for part, rs := range changes {
		sort.Slice(rs, func(i, j int) bool { return rs[i].start < rs[j].start })
		raw := p.parts[part]
		var patched bytes.Buffer
		patched.Grow(len(raw))
		last := 0
		for _, r := range rs {
			if r.start < last {
				return nil, errors.New("overlapping edits")
			}
			patched.Write(raw[last:r.start])
			patched.WriteString(r.value)
			last = r.end
		}
		patched.Write(raw[last:])
		raw = patched.Bytes()
		if _, e := parse(raw); e != nil {
			return nil, e
		}
		p.parts[part] = raw
	}
	var out bytes.Buffer
	w := zip.NewWriter(&out)
	_ = w.SetComment(p.archive.Comment)
	for _, f := range p.archive.File {
		if _, ok := changes[f.Name]; !ok {
			e = w.Copy(f)
		} else {
			h := f.FileHeader
			h.Method = zip.Deflate
			var dst io.Writer
			dst, e = w.CreateHeader(&h)
			if e == nil {
				_, e = dst.Write(p.parts[f.Name])
			}
		}
		if e != nil {
			return nil, e
		}
	}
	if e = w.Close(); e != nil {
		return nil, e
	}
	return out.Bytes(), nil
}
