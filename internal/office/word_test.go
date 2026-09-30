package office

import (
	"bytes"
	"nativepane/internal/testdoc"
	"reflect"
	"strings"
	"testing"
)

func TestWordTableProjectionAndNativeSave(t *testing.T) {
	original := testdoc.TableFile()
	engine := OOXML{}
	m, err := engine.Open(original, "docx")
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Blocks) != 3 || m.Blocks[0].Fields[0].Text != "Before the table" || m.Blocks[2].Fields[0].Text != "After the table" {
		t.Fatal("body flow order lost", m.Blocks)
	}
	table := m.Blocks[1]
	if table.Kind != "table" || !reflect.DeepEqual(table.ColumnWidths, []int{2400, 3600, 1200}) || len(table.Rows) != 3 {
		t.Fatal("table grid lost", table)
	}
	heading := table.Rows[0].Cells[0]
	if heading.ColSpan != 2 || heading.Fill != "DDEEFF" || heading.Blocks[0].Align != "center" || !heading.Blocks[0].Fields[0].Bold {
		t.Fatal("horizontal merge or formatting lost", heading)
	}
	if table.Rows[1].Cells[0].RowSpan != 2 || table.Rows[1].Cells[0].VAlign != "center" || len(table.Rows[2].Cells) != 2 || table.Rows[2].Cells[0].Column != 1 {
		t.Fatal("vertical merge shifted the following cells")
	}
	cell := table.Rows[1].Cells[1]
	if len(cell.Blocks) != 4 || cell.Blocks[1].Fields[0].Text != "Second paragraph" || cell.Blocks[2].Kind != "table" || len(cell.Blocks[2].Rows) != 1 {
		t.Fatal("cell paragraphs or nested table flattened", cell)
	}
	if len(table.Rows[1].Cells[2].Blocks[0].Fields) != 0 {
		t.Fatal("empty cell disappeared")
	}
	ids := map[string]bool{}
	var edits []Edit
	for _, b := range m.Blocks {
		for _, f := range b.Fields {
			if ids[f.ID] {
				t.Fatal("duplicate text target", f.ID)
			}
			ids[f.ID] = true
			if f.Text == "Editable cell" || f.Text == "Nested cell" {
				edits = append(edits, Edit{f.ID, "Revised " + f.Text})
			}
		}
	}
	if len(ids) != 10 || len(edits) != 2 {
		t.Fatal("table field inventory lost or duplicated text", len(ids), len(edits))
	}
	output, err := engine.Apply(original, "docx", edits)
	if err != nil {
		t.Fatal(err)
	}
	reopened, err := engine.Open(output, "docx")
	if err != nil {
		t.Fatal(err)
	}
	if reopened.Blocks[1].Rows[1].Cells[1].Blocks[0].Fields[0].Text != "Revised Editable cell" || reopened.Blocks[1].Rows[1].Cells[1].Blocks[2].Fields[0].Text != "Revised Nested cell" {
		t.Fatal("nested cell edits did not survive reopen")
	}
	before, after := unzip(t, original), unzip(t, output)
	expected := string(before["word/document.xml"])
	for _, text := range []string{"Editable cell", "Nested cell"} {
		expected = strings.Replace(expected, "<w:t>"+text+"</w:t>", `<w:t xml:space="preserve">Revised `+text+"</w:t>", 1)
	}
	if string(after["word/document.xml"]) != expected {
		t.Fatal("save modified table structure or unrelated document XML")
	}
	for name, b := range before {
		if name != "word/document.xml" && !bytes.Equal(b, after[name]) {
			t.Fatal("unrelated package part changed", name)
		}
	}
}

func TestWordVerticalMergeEndsAndSkippedGridColumns(t *testing.T) {
	parts := testdoc.Parts("docx")
	parts["word/document.xml"] = `<w:document xmlns:w="` + wordNS + `"><w:body><w:tbl><w:tblGrid><w:gridCol w:w="1000"/><w:gridCol w:w="1000"/><w:gridCol w:w="1000"/></w:tblGrid>
<w:tr><w:trPr><w:gridBefore w:val="1"/></w:trPr><w:tc><w:tcPr><w:vMerge w:val="restart"/></w:tcPr><w:p><w:r><w:t>Start</w:t></w:r></w:p></w:tc><w:tc><w:p/></w:tc></w:tr>
<w:tr><w:trPr><w:gridBefore w:val="1"/></w:trPr><w:tc><w:tcPr><w:vMerge/></w:tcPr><w:p><w:r><w:t>Continuation</w:t></w:r></w:p></w:tc><w:tc><w:p/></w:tc></w:tr>
<w:tr><w:trPr><w:gridBefore w:val="1"/><w:gridAfter w:val="1"/></w:trPr><w:tc><w:p><w:r><w:t>New cell</w:t></w:r></w:p></w:tc></w:tr>
<w:tr><w:trPr><w:gridBefore w:val="1"/></w:trPr><w:tc><w:tcPr><w:vMerge/></w:tcPr><w:p><w:r><w:t>Orphan continuation</w:t></w:r></w:p></w:tc></w:tr>
</w:tbl></w:body></w:document>`
	m, err := (OOXML{}).Open(testdoc.Package(parts), "docx")
	if err != nil {
		t.Fatal(err)
	}
	table := m.Blocks[0]
	anchor := table.Rows[0].Cells[0]
	if anchor.Column != 1 || anchor.RowSpan != 2 || anchor.Blocks[1].Fields[0].Text != "Continuation" {
		t.Fatal("merge position/content lost", anchor)
	}
	if len(table.Rows[1].Cells) != 1 || table.Rows[1].Cells[0].Column != 2 || table.Rows[2].Cells[0].RowSpan != 1 || len(table.Rows[3].Cells) != 1 {
		t.Fatal("merge crossed a normal cell or hid an orphan continuation")
	}
}
