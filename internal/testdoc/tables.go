package testdoc

// TableParts exercises document order, nested tables, empty cells, horizontal and
// vertical merges and multiple paragraphs in a cell.
func TableParts() map[string]string {
	p := Parts("docx")
	p["word/document.xml"] = `<?xml version="1.0" encoding="UTF-8"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body>
<w:p><w:r><w:t>Before the table</w:t></w:r></w:p>
<w:tbl>
<w:tblPr><w:tblBorders><w:top w:val="single"/><w:bottom w:val="single"/><w:insideH w:val="single"/><w:insideV w:val="single"/></w:tblBorders></w:tblPr>
<w:tblGrid><w:gridCol w:w="2400"/><w:gridCol w:w="3600"/><w:gridCol w:w="1200"/></w:tblGrid>
<w:tr><w:tc><w:tcPr><w:gridSpan w:val="2"/><w:shd w:fill="DDEEFF"/></w:tcPr><w:p><w:pPr><w:jc w:val="center"/></w:pPr><w:r><w:rPr><w:b/></w:rPr><w:t>Merged heading</w:t></w:r></w:p></w:tc><w:tc><w:p><w:r><w:t>Third column</w:t></w:r></w:p></w:tc></w:tr>
<w:tr><w:tc><w:tcPr><w:vMerge w:val="restart"/><w:vAlign w:val="center"/></w:tcPr><w:p><w:r><w:t>Vertical cell</w:t></w:r></w:p></w:tc><w:tc><w:p><w:r><w:t>Editable cell</w:t></w:r></w:p><w:p><w:r><w:t>Second paragraph</w:t></w:r></w:p><w:tbl><w:tblGrid><w:gridCol w:w="3600"/></w:tblGrid><w:tr><w:tc><w:p><w:r><w:t>Nested cell</w:t></w:r></w:p></w:tc></w:tr></w:tbl><w:p/></w:tc><w:tc><w:p/></w:tc></w:tr>
<w:tr><w:tc><w:tcPr><w:vMerge/></w:tcPr><w:p/></w:tc><w:tc><w:p><w:r><w:t>Final row</w:t></w:r></w:p></w:tc><w:tc><w:p><w:r><w:t>Last cell</w:t></w:r></w:p></w:tc></w:tr>
</w:tbl>
<w:p><w:r><w:t>After the table</w:t></w:r></w:p>
<w:sectPr><w:pgSz w:w="12240" w:h="15840"/><w:pgMar w:top="1440" w:right="1440" w:bottom="1440" w:left="1440"/></w:sectPr>
</w:body></w:document>`
	return p
}

func TableFile() []byte { return Package(TableParts()) }
