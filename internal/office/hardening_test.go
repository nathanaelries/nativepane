package office

import (
	"bytes"
	"errors"
	"nativepane/internal/testdoc"
	"strings"
	"testing"
)

func TestWordRestrictedConstructsRefuseAllEdits(t *testing.T) {
	constructs := []struct{ label, xml, part string }{
		{"insert", `<w:ins><w:r><w:t>Inserted</w:t></w:r></w:ins>`, ""},
		{"delete", `<w:del><w:r><w:delText>Deleted</w:delText></w:r></w:del>`, ""},
		{"property revision", `<w:pPr><w:pPrChange w:id="1"><w:pPr/></w:pPrChange></w:pPr>`, ""},
		{"move", `<w:moveFromRangeStart w:id="1"/>`, ""},
		{"conflict revision", `<w14:conflictIns xmlns:w14="http://schemas.microsoft.com/office/word/2010/wordml"/>`, ""},
		{"table revision", `<w:tbl><w:tblPr><w:tblPrChange/></w:tblPr></w:tbl>`, ""},
		{"simple field", `<w:fldSimple w:instr="DATE"><w:r><w:t>Cached date</w:t></w:r></w:fldSimple>`, ""},
		{"field begin", `<w:r><w:fldChar w:fldCharType="begin"/></w:r>`, ""},
		{"field instruction", `<w:r><w:instrText>DATE</w:instrText></w:r>`, ""},
		{"content control", `<w:sdt><w:sdtContent><w:r><w:t>Control</w:t></w:r></w:sdtContent></w:sdt>`, ""},
		{"legacy form", `<w:r><w:ffData/></w:r>`, ""},
		{"protection", `<w:documentProtection w:enforcement="1" w:edit="readOnly"/>`, "word/settings.xml"},
		{"write protection", `<w:writeProtection/>`, "word/settings.xml"},
		{"tracked mode", `<w:trackRevisions/>`, "word/settings.xml"},
		{"header field", `<w:p><w:fldSimple w:instr="PAGE"/></w:p>`, "word/header1.xml"},
		{"footer control", `<w:sdt/>`, "word/footer1.xml"},
		{"footnote revision", `<w:p><w:r><w:rPr><w:rPrChange/></w:rPr></w:r></w:p>`, "word/footnotes.xml"},
	}
	for _, item := range constructs {
		t.Run(item.label, func(t *testing.T) {
			parts := testdoc.Parts("docx")
			if item.part == "" {
				parts["word/document.xml"] = `<w:document xmlns:w="` + wordNS + `"><w:body><w:p><w:r><w:t>Ordinary run</w:t></w:r>` + item.xml + `</w:p></w:body></w:document>`
			} else {
				parts[item.part] = `<w:root xmlns:w="` + wordNS + `">` + item.xml + `</w:root>`
			}
			source := testdoc.Package(parts)
			engine := OOXML{}
			model, err := engine.Open(source, "docx")
			if err != nil {
				t.Fatal(err)
			}
			if !model.ReadOnly || len(model.Warnings) < 2 {
				t.Fatal("restriction not reported")
			}
			field := model.Blocks[0].Fields[0]
			if !field.ReadOnly {
				t.Fatal("ordinary text remained editable")
			}
			if _, err := engine.Apply(source, "docx", []Edit{{field.ID, "Bypass attempt"}}); !errors.Is(err, ErrReadOnly) {
				t.Fatal("edit was not refused", err)
			}
			output, err := engine.Apply(source, "docx", nil)
			if err != nil || !bytes.Equal(source, output) {
				t.Fatal("view/download changed original")
			}
		})
	}
}

func TestWordUnsupportedMarkersAndNoDuplicateFallback(t *testing.T) {
	parts := testdoc.Parts("docx")
	parts["word/document.xml"] = `<w:document xmlns:w="` + wordNS + `" xmlns:mc="http://schemas.openxmlformats.org/markup-compatibility/2006"><w:body><w:p><w:pPr><w:numPr/></w:pPr><w:r><w:t>Visible</w:t><w:tab/><w:br/><w:drawing><w:txbxContent><w:p><w:r><w:t>Textbox</w:t></w:r></w:p></w:txbxContent></w:drawing><w:footnoteReference w:id="1"/></w:r><mc:AlternateContent><mc:Choice><w:r><w:t>Choice</w:t></w:r></mc:Choice><mc:Fallback><w:r><w:t>Fallback</w:t></w:r></mc:Fallback></mc:AlternateContent></w:p><w:altChunk/></w:body></w:document>`
	parts["word/header1.xml"] = `<w:hdr xmlns:w="` + wordNS + `"><w:p><w:r><w:t>Header</w:t></w:r></w:p></w:hdr>`
	model, err := (OOXML{}).Open(testdoc.Package(parts), "docx")
	if err != nil {
		t.Fatal(err)
	}
	if len(model.Blocks[0].Fields) != 1 || model.Blocks[0].Fields[0].Text != "Visible" {
		t.Fatal("unsupported text leaked/duplicated")
	}
	markers := strings.Join(model.Blocks[0].Markers, "|")
	for _, label := range []string{"List numbering", "Tab positioning", "Inline line/page break", "Image, drawing", "Footnote/endnote", "Alternate content"} {
		if !strings.Contains(markers, label) {
			t.Fatal("missing marker", label, markers)
		}
	}
	if model.Blocks[1].Kind != "unsupported" || model.Blocks[2].Label != "Header content not rendered" {
		t.Fatal("flow omission not marked")
	}
}

func TestCellExtensionPreservation(t *testing.T) {
	for _, originalType := range []string{` t="s"><v>0</v>`, ` t="n"><v>42</v>`, ` t="b"><v>1</v>`, ` t="inlineStr"><is><t>Before</t></is>`, `>`} {
		parts := testdoc.Parts("xlsx")
		parts["xl/sharedStrings.xml"] = `<sst xmlns="` + sheetNS + `"><si><t>Shared</t></si></sst>`
		extension := `<extLst><ext uri="urn:nativepane:test"><v:payload xmlns:v="urn:vendor" v:flag="keep"> untouched &amp; data </v:payload></ext></extLst>`
		parts["xl/worksheets/sheet1.xml"] = `<worksheet xmlns="` + sheetNS + `"><sheetData><row r="1"><c r="A1" s="5" cm="2" vm="3"` + originalType + extension + `</c></row></sheetData><mergeCells count="1"><mergeCell ref="A1:B1"/></mergeCells></worksheet>`
		source := testdoc.Package(parts)
		model, err := (OOXML{}).Open(source, "xlsx")
		if err != nil {
			t.Fatal(err)
		}
		output, err := (OOXML{}).Apply(source, "xlsx", []Edit{{model.Blocks[0].Fields[0].ID, "After"}})
		if err != nil {
			t.Fatal(err)
		}
		before, after := unzip(t, source), unzip(t, output)
		if !bytes.Contains(after["xl/worksheets/sheet1.xml"], []byte(extension)) || !bytes.Contains(after["xl/worksheets/sheet1.xml"], []byte(`s="5" cm="2" vm="3"`)) || !bytes.Contains(after["xl/worksheets/sheet1.xml"], []byte(`<mergeCells count="1"><mergeCell ref="A1:B1"/></mergeCells>`)) {
			t.Fatal("cell metadata or merges changed")
		}
		for name, raw := range before {
			if name != "xl/worksheets/sheet1.xml" && !bytes.Equal(raw, after[name]) {
				t.Fatal("unrelated part changed", name)
			}
		}
	}
}

func TestMergedContinuationRetainsUnsupportedMarkers(t *testing.T) {
	parts := testdoc.Parts("docx")
	parts["word/document.xml"] = `<w:document xmlns:w="` + wordNS + `"><w:body><w:tbl><w:tblGrid><w:gridCol w:w="1000"/></w:tblGrid><w:tr><w:tc><w:tcPr><w:vMerge w:val="restart"/></w:tcPr><w:p><w:r><w:t>Anchor</w:t></w:r></w:p></w:tc></w:tr><w:tr><w:tc><w:tcPr><w:vMerge/></w:tcPr><w:p><w:r><w:drawing/></w:r></w:p><w:altChunk/></w:tc></w:tr></w:tbl></w:body></w:document>`
	source := testdoc.Package(parts)
	engine := OOXML{}
	model, err := engine.Open(source, "docx")
	if err != nil {
		t.Fatal(err)
	}
	anchor := model.Blocks[0].Rows[0].Cells[0]
	if anchor.RowSpan != 2 || len(anchor.Blocks) != 3 || len(anchor.Blocks[1].Markers) != 1 || anchor.Blocks[2].Kind != "unsupported" {
		t.Fatal("merged continuation dropped unsupported content markers")
	}
	output, err := engine.Apply(source, "docx", []Edit{{anchor.Blocks[0].Fields[0].ID, "Changed"}})
	if err != nil {
		t.Fatal(err)
	}
	reopened, err := engine.Open(output, "docx")
	if err != nil || len(reopened.Blocks[0].Rows[0].Cells[0].Blocks) != 3 {
		t.Fatal("markers changed after native save", err)
	}
	before, after := unzip(t, source), unzip(t, output)
	want := bytes.Replace(before["word/document.xml"], []byte(`<w:t>Anchor</w:t>`), []byte(`<w:t xml:space="preserve">Changed</w:t>`), 1)
	if !bytes.Equal(want, after["word/document.xml"]) {
		t.Fatal("merged source content changed outside edited run")
	}
}

func TestCellExtensionContentIsNotEditableCellData(t *testing.T) {
	parts := testdoc.Parts("xlsx")
	parts["xl/worksheets/sheet1.xml"] = `<worksheet xmlns="` + sheetNS + `"><sheetData><row r="1"><c r="A1" t="inlineStr"><is><t>Actual value</t></is><extLst><ext uri="urn:vendor"><t>Metadata text</t><v>999</v><f>Metadata formula</f><c r="Z99"><v>777</v></c></ext></extLst></c></row></sheetData></worksheet>`
	source := testdoc.Package(parts)
	model, err := (OOXML{}).Open(source, "xlsx")
	if err != nil {
		t.Fatal(err)
	}
	if len(model.Blocks[0].Fields) != 1 || model.Blocks[0].Fields[0].Text != "Actual value" || model.Blocks[0].Fields[0].ReadOnly {
		t.Fatal("extension data was projected as a cell/value/formula")
	}
	output, err := (OOXML{}).Apply(source, "xlsx", []Edit{{model.Blocks[0].Fields[0].ID, "Changed"}})
	if err != nil {
		t.Fatal(err)
	}
	after := unzip(t, output)["xl/worksheets/sheet1.xml"]
	if !bytes.Contains(after, []byte(`<t>Metadata text</t><v>999</v><f>Metadata formula</f><c r="Z99"><v>777</v></c>`)) {
		t.Fatal("extension children changed")
	}
}
