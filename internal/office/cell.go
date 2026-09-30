package office

import (
	"errors"
	"math"
	"strconv"
	"strings"
)

// patchCell replaces only value-bearing direct children. Every other child,
// including extLst and unknown vendor data, stays byte-identical and in place.
func patchCell(raw []byte, t target, text string) (string, error) {
	n := t.n
	start := string(raw[n.start:n.inner])
	qname := strings.Fields(strings.TrimPrefix(start, "<"))[0]
	qname = strings.TrimRight(qname, "/>")
	prefix := ""
	if i := strings.Index(qname, ":"); i >= 0 {
		prefix = qname[:i+1]
	}
	numeric, err := strconv.ParseFloat(text, 64)
	kind, payload := "inlineStr", `<`+prefix+`is><`+prefix+`t xml:space="preserve">`+escaped(text)+`</`+prefix+`t></`+prefix+`is>`
	if t.field.Kind == "number" && text != "" && err == nil && !math.IsInf(numeric, 0) && !math.IsNaN(numeric) {
		kind, payload = "n", `<`+prefix+`v>`+escaped(text)+`</`+prefix+`v>`
	} else if t.field.Kind == "boolean" && (text == "0" || text == "1") {
		kind, payload = "b", `<`+prefix+`v>`+text+`</`+prefix+`v>`
	}
	start = typeAttr.ReplaceAllString(start, "")
	start = strings.TrimSuffix(strings.TrimSuffix(start, ">"), "/") + ` t="` + kind + `">`
	var content strings.Builder
	content.WriteString(start)
	last, inserted := n.inner, false
	for _, child := range n.children {
		if child.name.Space == sheetNS && (child.name.Local == "v" || child.name.Local == "is") {
			content.Write(raw[last:child.start])
			if !inserted {
				content.WriteString(payload)
				inserted = true
			}
			last = child.end
		} else if !inserted {
			// New values precede extLst (and any unknown child) in empty cells.
			content.Write(raw[last:child.start])
			content.WriteString(payload)
			last, inserted = child.start, true
		}
	}
	if n.endStart < n.inner {
		// encoding/xml emits an EndElement for a self-closing tag at its end.
		if len(n.children) != 0 {
			return "", errors.New("invalid empty cell")
		}
	} else {
		content.Write(raw[last:n.endStart])
	}
	if !inserted {
		content.WriteString(payload)
	}
	content.WriteString(`</` + qname + `>`)
	return content.String(), nil
}
