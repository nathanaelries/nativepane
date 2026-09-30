package office

import (
	"archive/zip"
	"bytes"
	"io"
	"nativepane/internal/testdoc"
	"strings"
	"testing"
)

func unzip(t *testing.T, b []byte) map[string][]byte {
	t.Helper()
	z, e := zip.NewReader(bytes.NewReader(b), int64(len(b)))
	if e != nil {
		t.Fatal(e)
	}
	out := map[string][]byte{}
	for _, f := range z.File {
		r, e := f.Open()
		if e != nil {
			t.Fatal(e)
		}
		v, e := io.ReadAll(r)
		r.Close()
		if e != nil {
			t.Fatal(e)
		}
		out[f.Name] = v
		if strings.HasSuffix(f.Name, ".xml") || strings.HasSuffix(f.Name, ".rels") {
			if _, e = parse(v); e != nil {
				t.Fatalf("%s: %v", f.Name, e)
			}
		}
	}
	return out
}
func TestRoundTrip(t *testing.T) {
	for _, format := range []string{"docx", "xlsx", "pptx"} {
		t.Run(format, func(t *testing.T) {
			original := testdoc.File(format)
			engine := OOXML{}
			model, e := engine.Open(original, format)
			if e != nil {
				t.Fatal(e)
			}
			f := model.Blocks[0].Fields[0]
			text := " Revised <text> & café 世界 "
			edited, e := engine.Apply(original, format, []Edit{{f.ID, text}})
			if e != nil {
				t.Fatal(e)
			}
			reopened, e := engine.Open(edited, format)
			if e != nil {
				t.Fatal(e)
			}
			if reopened.Blocks[0].Fields[0].Text != text {
				t.Fatal("edit did not survive reopen")
			}
			before, after := unzip(t, original), unzip(t, edited)
			if len(before) != len(after) {
				t.Fatal("package parts changed")
			}
			changed := 0
			for name, b := range before {
				if !bytes.Equal(b, after[name]) {
					changed++
				}
			}
			if changed != 1 {
				t.Fatalf("changed %d parts, want 1", changed)
			}
			if !bytes.Equal(before["customXml/item1.xml"], after["customXml/item1.xml"]) {
				t.Fatal("unknown part changed")
			}
			noOp, e := engine.Apply(original, format, nil)
			if e != nil || !bytes.Equal(original, noOp) {
				t.Fatal("no-op must preserve exact source bytes")
			}
		})
	}
}
func TestCells(t *testing.T) {
	b := testdoc.File("xlsx")
	engine := OOXML{}
	m, _ := engine.Open(b, "xlsx")
	var number, formula Field
	for _, f := range m.Blocks[0].Fields {
		if f.Address == "B2" {
			number = f
		}
		if f.Address == "C2" {
			formula = f
		}
	}
	out, e := engine.Apply(b, "xlsx", []Edit{{number.ID, "77.5"}})
	if e != nil {
		t.Fatal(e)
	}
	m, e = engine.Open(out, "xlsx")
	if e != nil {
		t.Fatal(e)
	}
	for _, f := range m.Blocks[0].Fields {
		if f.Address == "B2" && (f.Text != "77.5" || f.Kind != "number") {
			t.Fatal(f)
		}
	}
	if _, e = engine.Apply(b, "xlsx", []Edit{{formula.ID, "123"}}); e == nil {
		t.Fatal("formula must be read-only")
	}
}
func TestRejectUnsupportedAndMalicious(t *testing.T) {
	engine := OOXML{}
	for _, mutation := range []func(map[string]string){func(p map[string]string) { p["word/vbaProject.bin"] = "bad" }, func(p map[string]string) { p["_xmlsignatures/sig1.xml"] = "bad" }, func(p map[string]string) { p["../bad.xml"] = "bad" }, func(p map[string]string) { p["word/document.xml"] = `<!DOCTYPE x><x/>` }, func(p map[string]string) { p["word/document.xml"] = `<x>` }} {
		p := testdoc.Parts("docx")
		mutation(p)
		if _, e := engine.Open(testdoc.Package(p), "docx"); e == nil {
			t.Fatal("unsafe document accepted")
		}
	}
	if _, e := engine.Open([]byte("not ZIP"), "docx"); e == nil {
		t.Fatal("invalid ZIP accepted")
	}
	b := testdoc.File("docx")
	m, _ := engine.Open(b, "docx")
	id := m.Blocks[0].Fields[0].ID
	for _, edits := range [][]Edit{{{"missing", "x"}}, {{id, "a"}, {id, "b"}}, {{id, "\x00"}}} {
		if _, e := engine.Apply(b, "docx", edits); e == nil {
			t.Fatal("invalid edit accepted")
		}
	}
}
func TestZipBomb(t *testing.T) {
	var b bytes.Buffer
	z := zip.NewWriter(&b)
	f, _ := z.Create("big.xml")
	block := make([]byte, 1<<20)
	for i := 0; i < 129; i++ {
		_, _ = f.Write(block)
	}
	_ = z.Close()
	if _, e := (OOXML{}).Open(b.Bytes(), "docx"); e == nil {
		t.Fatal("ZIP bomb accepted")
	}
}
func TestSharedStringsAndEmptyCells(t *testing.T) {
	p := testdoc.Parts("xlsx")
	p["xl/sharedStrings.xml"] = `<sst xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><si><t>Shared</t></si></sst>`
	p["xl/worksheets/sheet1.xml"] = `<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData><row r="1"><c r="A1" t="s"><v>0</v></c><c r="B1" t="s"><v>0</v></c><c r="C1" s="2"/></row></sheetData></worksheet>`
	b := testdoc.Package(p)
	engine := OOXML{}
	m, e := engine.Open(b, "xlsx")
	if e != nil {
		t.Fatal(e)
	}
	out, e := engine.Apply(b, "xlsx", []Edit{{m.Blocks[0].Fields[0].ID, "Unshared"}, {m.Blocks[0].Fields[2].ID, "new"}})
	if e != nil {
		t.Fatal(e)
	}
	m, e = engine.Open(out, "xlsx")
	if e != nil {
		t.Fatal(e)
	}
	if m.Blocks[0].Fields[1].Text != "Shared" || m.Blocks[0].Fields[2].Text != "new" {
		t.Fatal(m)
	}
	if !bytes.Contains(unzip(t, out)["xl/worksheets/sheet1.xml"], []byte(`s="2"`)) {
		t.Fatal("style lost")
	}
}
