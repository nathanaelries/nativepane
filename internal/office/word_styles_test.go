package office

import (
	"bytes"
	"nativepane/internal/testdoc"
	"reflect"
	"testing"
)

func TestWordStylesPageAndPreferredCellWidths(t *testing.T) {
	parts := testdoc.Parts("docx")
	parts["word/styles.xml"] = `<w:styles xmlns:w="` + wordNS + `"><w:docDefaults><w:rPrDefault><w:rPr><w:rFonts w:ascii="Calibri"/><w:sz w:val="22"/></w:rPr></w:rPrDefault><w:pPrDefault><w:pPr><w:spacing w:after="200" w:line="276"/></w:pPr></w:pPrDefault></w:docDefaults>
<w:style w:type="paragraph" w:default="1" w:styleId="Normal"><w:pPr><w:spacing w:after="120" w:line="264"/></w:pPr><w:rPr><w:sz w:val="19"/></w:rPr></w:style>
<w:style w:type="paragraph" w:styleId="Heading"><w:basedOn w:val="Normal"/><w:pPr><w:keepNext/><w:spacing w:before="240"/></w:pPr><w:rPr><w:b/><w:sz w:val="36"/><w:color w:val="123456"/></w:rPr></w:style>
<w:style w:type="paragraph" w:styleId="Title"><w:basedOn w:val="Heading"/><w:pPr><w:pBdr><w:bottom w:val="single" w:sz="8" w:color="4F81BD" w:space="4"/></w:pBdr></w:pPr><w:rPr><w:rFonts w:asciiTheme="majorHAnsi"/></w:rPr></w:style>
<w:style w:type="character" w:styleId="Link"><w:rPr><w:u w:val="single"/><w:color w:val="154B73"/></w:rPr></w:style>
<w:style w:type="table" w:default="1" w:styleId="TableNormal"><w:tblPr><w:tblCellMar><w:left w:w="108"/></w:tblCellMar></w:tblPr></w:style>
</w:styles>`
	parts["word/theme/theme1.xml"] = `<a:theme xmlns:a="` + drawNS + `"><a:themeElements><a:fontScheme><a:majorFont><a:latin typeface="Arial"/></a:majorFont></a:fontScheme></a:themeElements></a:theme>`
	parts["word/document.xml"] = `<w:document xmlns:w="` + wordNS + `"><w:body><w:p><w:pPr><w:pStyle w:val="Title"/></w:pPr><w:r><w:t>Styled heading</w:t></w:r><w:r><w:rPr><w:b w:val="0"/><w:sz w:val="20"/></w:rPr><w:t>Override</w:t></w:r></w:p><w:p><w:r><w:rPr><w:rStyle w:val="Link"/></w:rPr><w:t>Body link</w:t></w:r></w:p>
<w:tbl><w:tblGrid><w:gridCol w:w="4000"/><w:gridCol w:w="4000"/></w:tblGrid><w:tr><w:trPr><w:tblHeader/></w:trPr><w:tc><w:tcPr><w:tcW w:type="dxa" w:w="2000"/><w:tcMar><w:top w:w="95"/></w:tcMar><w:tcBorders><w:bottom w:val="single" w:sz="4" w:color="D9D9D9"/></w:tcBorders></w:tcPr><w:p><w:pPr><w:spacing w:after="40" w:line="259"/></w:pPr><w:r><w:rPr><w:color w:val="FFFFFF"/></w:rPr><w:t>Cell</w:t></w:r></w:p></w:tc><w:tc><w:tcPr><w:tcW w:type="dxa" w:w="6000"/></w:tcPr><w:p/></w:tc></w:tr></w:tbl>
<w:sectPr><w:pgSz w:w="12240" w:h="15840"/><w:pgMar w:top="1037" w:right="1123" w:bottom="1037" w:left="1123"/></w:sectPr></w:body></w:document>`
	b := testdoc.Package(parts)
	m, err := (OOXML{}).Open(b, "docx")
	if err != nil {
		t.Fatal(err)
	}
	heading := m.Blocks[0]
	if !heading.KeepNext || heading.Style["fontSize"] != "24px" || heading.Style["lineHeight"] != "1.1" || heading.Style["marginTop"] != "16px" || heading.Style["marginBottom"] != "8px" || heading.Style["fontFamily"] != `"Arial", Calibri, Arial, sans-serif` {
		t.Fatal("inherited heading style lost", heading.Style)
	}
	if heading.Style["borderBottom"] != "1.3333333333333333px solid #4F81BD" || heading.Fields[0].Style["color"] != "#123456" || !heading.Fields[0].Bold || heading.Fields[1].Bold || heading.Fields[1].Style["fontSize"] != "13.333333333333334px" {
		t.Fatal("direct override or heading border lost", heading)
	}
	if m.Blocks[1].Style["fontSize"] != "12.666666666666666px" || m.Blocks[1].Fields[0].Style["color"] != "#154B73" || m.Blocks[1].Fields[0].Style["textDecorationLine"] != "underline" {
		t.Fatal("Normal or character style lost")
	}
	if *m.Page != (PageLayout{12240, 15840, 1037, 1123, 1037, 1123}) {
		t.Fatal("page dimensions lost", m.Page)
	}
	table := m.Blocks[2]
	cell := table.Rows[0].Cells[0]
	if !table.Rows[0].RepeatHeader || !reflect.DeepEqual(table.ColumnWidths, []int{2000, 6000}) || cell.Style["paddingLeft"] != "7.2px" || cell.Style["paddingTop"] != "6.333333333333333px" || cell.Style["borderBottom"] != "0.6666666666666666px solid #D9D9D9" {
		t.Fatal("table metrics lost", cell.Style, table.ColumnWidths)
	}
	if cell.Blocks[0].Style["lineHeight"] != "1.0791666666666666" || cell.Blocks[0].Fields[0].Style["color"] != "#FFFFFF" {
		t.Fatal("compact cell text style lost")
	}
	out, err := (OOXML{}).Apply(b, "docx", []Edit{{cell.Blocks[0].Fields[0].ID, "Revised cell"}})
	if err != nil {
		t.Fatal(err)
	}
	before, after := unzip(t, b), unzip(t, out)
	for _, part := range []string{"word/styles.xml", "word/theme/theme1.xml"} {
		if !bytes.Equal(before[part], after[part]) {
			t.Fatal("save changed formatting part", part)
		}
	}
}

func TestWordStyleCycleAndUnsafeValues(t *testing.T) {
	parts := testdoc.Parts("docx")
	parts["word/styles.xml"] = `<w:styles xmlns:w="` + wordNS + `"><w:style w:type="paragraph" w:default="1" w:styleId="A"><w:basedOn w:val="B"/><w:rPr><w:rFonts w:ascii="bad;url(https://example.com)"/><w:sz w:val="99999999"/></w:rPr></w:style><w:style w:styleId="B"><w:basedOn w:val="A"/></w:style></w:styles>`
	m, err := (OOXML{}).Open(testdoc.Package(parts), "docx")
	if err != nil {
		t.Fatal(err)
	}
	if m.Blocks[0].Style["fontSize"] != "14.666667px" || m.Blocks[0].Style["fontFamily"] != `"Times New Roman", Times, serif` {
		t.Fatal("unsafe style values escaped bounds")
	}
}
