// Package testdoc creates small, original OOXML fixtures for tests and the demo.
package testdoc

import (
	"archive/zip"
	"bytes"
	"fmt"
)

func Package(parts map[string]string) []byte {
	var b bytes.Buffer
	w := zip.NewWriter(&b)
	for name, text := range parts {
		f, _ := w.Create(name)
		_, _ = f.Write([]byte(text))
	}
	_ = w.Close()
	return b.Bytes()
}
func Parts(format string) map[string]string {
	main := map[string]string{"docx": "word/document.xml", "xlsx": "xl/workbook.xml", "pptx": "ppt/presentation.xml"}[format]
	kind := map[string]string{"docx": "wordprocessingml.document", "xlsx": "spreadsheetml.sheet", "pptx": "presentationml.presentation"}[format]
	p := map[string]string{
		"[Content_Types].xml": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/><Override PartName="/` + main + `" ContentType="application/vnd.openxmlformats-officedocument.` + kind + `.main+xml"/>`,
		"_rels/.rels":         `<?xml version="1.0"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="` + main + `"/></Relationships>`,
		"customXml/item1.xml": `<preserved>This untouched part must survive byte for byte.</preserved>`,
	}
	switch format {
	case "docx":
		p[main] = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body><w:p><w:r><w:rPr><w:b/></w:rPr><w:t>NativePane</w:t></w:r></w:p><w:p><w:r><w:t>Native Office documents in the browser. One container. MIT.</w:t></w:r></w:p><w:p><w:r><w:t>Edit this sentence, wait for autosave, and download the original format.</w:t></w:r></w:p><w:p><w:r><w:t>Native review without the native pain.</w:t></w:r></w:p><w:sectPr><w:pgSz w:w="12240" w:h="15840"/><w:pgMar w:top="1440" w:right="1440" w:bottom="1440" w:left="1440"/></w:sectPr></w:body></w:document>`
	case "xlsx":
		p[main] = `<?xml version="1.0"?><workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><sheets><sheet name="Budget" sheetId="1" r:id="rId1"/></sheets></workbook>`
		p["xl/_rels/workbook.xml.rels"] = `<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/></Relationships>`
		p["xl/worksheets/sheet1.xml"] = `<?xml version="1.0"?><worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><dimension ref="A1:C3"/><sheetData><row r="1"><c r="A1" t="inlineStr"><is><t>NativePane budget</t></is></c><c r="B1" t="inlineStr"><is><t>Amount</t></is></c></row><row r="2"><c r="A2" t="inlineStr"><is><t>Hosting</t></is></c><c r="B2"><v>42</v></c><c r="C2"><f>B2*2</f><v>84</v></c></row><row r="3"><c r="A3" t="inlineStr"><is><t>Licenses</t></is></c><c r="B3"><v>0</v></c></row></sheetData></worksheet>`
		p["[Content_Types].xml"] += `<Override PartName="/xl/worksheets/sheet1.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/>`
	case "pptx":
		p[main] = `<?xml version="1.0"?><p:presentation xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships" xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main"><p:sldIdLst><p:sldId id="256" r:id="rId1"/></p:sldIdLst><p:sldSz cx="9144000" cy="5143500"/><p:notesSz cx="6858000" cy="9144000"/></p:presentation>`
		p["ppt/_rels/presentation.xml.rels"] = `<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slide" Target="slides/slide1.xml"/></Relationships>`
		p["ppt/slides/slide1.xml"] = `<?xml version="1.0"?><p:sld xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships" xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main"><p:cSld><p:spTree><p:nvGrpSpPr><p:cNvPr id="1" name=""/><p:cNvGrpSpPr/><p:nvPr/></p:nvGrpSpPr><p:grpSpPr/><p:sp><p:nvSpPr><p:cNvPr id="2" name="Title"/><p:cNvSpPr txBox="1"/><p:nvPr/></p:nvSpPr><p:spPr><a:xfrm><a:off x="914400" y="914400"/><a:ext cx="7315200" cy="2743200"/></a:xfrm><a:prstGeom prst="rect"><a:avLst/></a:prstGeom></p:spPr><p:txBody><a:bodyPr/><a:lstStyle/><a:p><a:r><a:t>NativePane</a:t></a:r></a:p><a:p><a:r><a:t>Native documents. One container. MIT.</a:t></a:r></a:p></p:txBody></p:sp></p:spTree></p:cSld></p:sld>`
		p["[Content_Types].xml"] += `<Override PartName="/ppt/slides/slide1.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.slide+xml"/>`
	default:
		panic(fmt.Sprintf("unknown fixture %s", format))
	}
	p["[Content_Types].xml"] += `</Types>`
	return p
}
func File(format string) []byte { return Package(Parts(format)) }
