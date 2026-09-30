package testdoc

import (
	"fmt"
	"strings"
)

// LargeFile generates original fixtures with an exact number of editable fields.
func LargeFile(format string, count int) []byte {
	parts := Parts(format)
	var body strings.Builder
	switch format {
	case "docx":
		body.WriteString(`<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body>`)
		for i := 0; i < count; i++ {
			if i%50 == 0 {
				body.WriteString(`<w:p>`)
			}
			fmt.Fprintf(&body, `<w:r><w:t>Run %d. </w:t></w:r>`, i)
			if i%50 == 49 || i == count-1 {
				body.WriteString(`</w:p>`)
			}
		}
		body.WriteString(`</w:body></w:document>`)
		parts["word/document.xml"] = body.String()
	case "xlsx":
		body.WriteString(`<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData>`)
		for i := 0; i < count; i++ {
			if i%10 == 0 {
				fmt.Fprintf(&body, `<row r="%d">`, i/10+1)
			}
			fmt.Fprintf(&body, `<c r="%c%d"><v>%d</v></c>`, 'A'+i%10, i/10+1, i)
			if i%10 == 9 || i == count-1 {
				body.WriteString(`</row>`)
			}
		}
		body.WriteString(`</sheetData></worksheet>`)
		parts["xl/worksheets/sheet1.xml"] = body.String()
	case "pptx":
		body.WriteString(`<p:sld xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main" xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main"><p:cSld><p:spTree><p:sp><p:txBody><a:p>`)
		for i := 0; i < count; i++ {
			fmt.Fprintf(&body, `<a:r><a:t>Run %d</a:t></a:r>`, i)
		}
		body.WriteString(`</a:p></p:txBody></p:sp></p:spTree></p:cSld></p:sld>`)
		parts["ppt/slides/slide1.xml"] = body.String()
	default:
		panic("unsupported large fixture format")
	}
	return Package(parts)
}
