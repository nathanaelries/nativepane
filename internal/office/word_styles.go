package office

import (
	"fmt"
	"regexp"
	"strings"
)

// CSS hints are generated from bounded, typed OOXML values, never arbitrary CSS
// or HTML from a package. All source style/theme parts remain untouched on save.
type wordStyles struct {
	styles                         map[string]*node
	defaults                       map[string]string
	defaultParagraph, defaultTable string
	fonts, colors                  map[string]string
}

func copyStyle(src map[string]string) map[string]string {
	dst := map[string]string{}
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
func textStyle(style map[string]string) map[string]string {
	out := map[string]string{}
	for _, key := range []string{"fontFamily", "fontSize", "fontWeight", "fontStyle", "color", "textDecorationLine", "letterSpacing"} {
		if v := style[key]; v != "" {
			out[key] = v
		}
	}
	return out
}

func pixel(v float64) string { return fmt.Sprintf("%gpx", v) }
func numberAttr(n *node, name string, min, max int) (int, bool) {
	if n == nil || n.attr(name) == "" {
		return 0, false
	}
	v := boundedNumber(n.attr(name), min-1, min, max)
	return v, v >= min
}
func wordEnabled(n *node) bool {
	return n != nil && n.attr("val") != "0" && n.attr("val") != "false" && n.attr("val") != "off"
}

func loadWordStyles(parts map[string][]byte) (*wordStyles, error) {
	s := &wordStyles{styles: map[string]*node{}, defaults: map[string]string{"fontFamily": `"Times New Roman", Times, serif`, "fontSize": "14.666667px", "color": "#000000", "fontWeight": "normal", "fontStyle": "normal", "lineHeight": "normal", "marginTop": "0px", "marginBottom": "0px"}, fonts: map[string]string{}, colors: map[string]string{}}
	if b := parts["word/theme/theme1.xml"]; len(b) > 0 {
		r, err := parse(b)
		if err != nil {
			return nil, err
		}
		for _, kind := range []string{"majorFont", "minorFont"} {
			for _, font := range r.all(drawNS, kind) {
				if latin := font.child(drawNS, "latin"); latin != nil {
					prefix := strings.TrimSuffix(kind, "Font")
					s.fonts[prefix+"HAnsi"] = latin.attr("typeface")
					s.fonts[prefix+"Ascii"] = latin.attr("typeface")
				}
			}
		}
		for _, scheme := range r.all(drawNS, "clrScheme") {
			for _, color := range scheme.children {
				if len(color.children) > 0 {
					value := color.children[0].attr("val")
					if !wordFill.MatchString(value) {
						value = color.children[0].attr("lastClr")
					}
					if wordFill.MatchString(value) {
						s.colors[color.name.Local] = value
					}
				}
			}
		}
	}
	if b := parts["word/styles.xml"]; len(b) > 0 {
		r, err := parse(b)
		if err != nil {
			return nil, err
		}
		if defaults := r.child(wordNS, "styles").child(wordNS, "docDefaults"); defaults != nil {
			s.run(s.defaults, defaults.child(wordNS, "rPrDefault").child(wordNS, "rPr"))
			s.paragraph(s.defaults, defaults.child(wordNS, "pPrDefault").child(wordNS, "pPr"))
		}
		for _, style := range r.all(wordNS, "style") {
			s.styles[style.attr("styleId")] = style
			if style.attr("default") == "1" {
				switch style.attr("type") {
				case "paragraph":
					s.defaultParagraph = style.attr("styleId")
				case "table":
					s.defaultTable = style.attr("styleId")
				}
			}
		}
	}
	return s, nil
}

func (s *wordStyles) chain(id string) []*node {
	seen := map[string]bool{}
	var chain []*node
	for id != "" && !seen[id] && len(chain) < 32 {
		seen[id] = true
		n := s.styles[id]
		if n == nil {
			break
		}
		chain = append(chain, n)
		id = wordProperty(n, "basedOn")
	}
	for i, j := 0, len(chain)-1; i < j; i, j = i+1, j-1 {
		chain[i], chain[j] = chain[j], chain[i]
	}
	return chain
}

var fontName = regexp.MustCompile(`^[\p{L}\p{N} _.,-]{1,100}$`)

func fontFamily(name string) string {
	if !fontName.MatchString(name) {
		return ""
	}
	lower := strings.ToLower(name)
	if strings.Contains(lower, "times") || strings.Contains(lower, "cambria") || strings.Contains(lower, "georgia") {
		return fmt.Sprintf(`"%s", "Times New Roman", serif`, name)
	}
	return fmt.Sprintf(`"%s", Calibri, Arial, sans-serif`, name)
}
func (s *wordStyles) color(n *node) string {
	if n == nil {
		return ""
	}
	value := n.attr("val")
	if theme := s.colors[n.attr("themeColor")]; theme != "" {
		value = theme
	}
	if wordFill.MatchString(value) {
		return "#" + value
	}
	if value == "auto" {
		return "#000000"
	}
	return ""
}
func (s *wordStyles) run(out map[string]string, props *node) {
	if props == nil {
		return
	}
	if fonts := props.child(wordNS, "rFonts"); fonts != nil {
		name := fonts.attr("ascii")
		if name == "" {
			name = fonts.attr("hAnsi")
		}
		theme := fonts.attr("asciiTheme")
		if theme == "" {
			theme = fonts.attr("hAnsiTheme")
		}
		if face := s.fonts[theme]; face != "" {
			name = face
		}
		if family := fontFamily(name); family != "" {
			out["fontFamily"] = family
		}
	}
	if size, ok := numberAttr(props.child(wordNS, "sz"), "val", 2, 400); ok {
		out["fontSize"] = pixel(float64(size) * 2 / 3)
	}
	for _, item := range []struct{ name, key, on, off string }{{"b", "fontWeight", "bold", "normal"}, {"i", "fontStyle", "italic", "normal"}} {
		if n := props.child(wordNS, item.name); n != nil {
			if wordEnabled(n) {
				out[item.key] = item.on
			} else {
				out[item.key] = item.off
			}
		}
	}
	if color := s.color(props.child(wordNS, "color")); color != "" {
		out["color"] = color
	}
	if u := props.child(wordNS, "u"); u != nil {
		if u.attr("val") == "none" {
			out["textDecorationLine"] = "none"
		} else {
			out["textDecorationLine"] = "underline"
		}
	}
	if wordEnabled(props.child(wordNS, "strike")) {
		out["textDecorationLine"] = "line-through"
	}
	if value, ok := numberAttr(props.child(wordNS, "spacing"), "val", -1000, 1000); ok {
		out["letterSpacing"] = pixel(float64(value) / 15)
	}
}

func (s *wordStyles) border(n *node) string {
	if n == nil {
		return ""
	}
	switch n.attr("val") {
	case "nil", "none":
		return "none"
	case "":
		return ""
	}
	style := "solid"
	switch n.attr("val") {
	case "dotted":
		style = "dotted"
	case "dashed", "dashSmallGap":
		style = "dashed"
	case "double":
		style = "double"
	}
	size := boundedNumber(n.attr("sz"), 4, 0, 96)
	value := n.attr("color")
	if theme := s.colors[n.attr("themeColor")]; theme != "" {
		value = theme
	}
	color := "#000000"
	if wordFill.MatchString(value) {
		color = "#" + value
	}
	return pixel(float64(size)/6) + " " + style + " " + color
}
func (s *wordStyles) paragraph(out map[string]string, props *node) {
	if props == nil {
		return
	}
	spacing := props.child(wordNS, "spacing")
	for _, item := range []struct{ attr, key string }{{"before", "marginTop"}, {"after", "marginBottom"}} {
		if v, ok := numberAttr(spacing, item.attr, 0, 14400); ok {
			out[item.key] = pixel(float64(v) / 15)
		}
	}
	if v, ok := numberAttr(spacing, "line", 1, 14400); ok {
		if spacing.attr("lineRule") == "exact" || spacing.attr("lineRule") == "atLeast" {
			out["lineHeight"] = pixel(float64(v) / 15)
		} else {
			out["lineHeight"] = fmt.Sprintf("%g", float64(v)/240)
		}
	}
	switch wordProperty(props, "jc") {
	case "center":
		out["textAlign"] = "center"
	case "right", "end":
		out["textAlign"] = "right"
	case "both", "distribute":
		out["textAlign"] = "justify"
	case "left", "start":
		out["textAlign"] = "left"
	}
	indent := props.child(wordNS, "ind")
	for _, item := range []struct{ attr, key string }{{"left", "marginLeft"}, {"right", "marginRight"}, {"firstLine", "textIndent"}} {
		if v, ok := numberAttr(indent, item.attr, -14400, 14400); ok {
			out[item.key] = pixel(float64(v) / 15)
		}
	}
	if v, ok := numberAttr(indent, "hanging", 0, 14400); ok {
		out["textIndent"] = pixel(-float64(v) / 15)
	}
	if borders := props.child(wordNS, "pBdr"); borders != nil {
		for _, side := range []string{"top", "right", "bottom", "left"} {
			n := borders.child(wordNS, side)
			if border := s.border(n); border != "" {
				key := strings.ToUpper(side[:1]) + side[1:]
				out["border"+key] = border
				if space, ok := numberAttr(n, "space", 0, 31); ok {
					out["padding"+key] = pixel(float64(space) * 4 / 3)
				}
			}
		}
	}
}

func (s *wordStyles) paragraphStyle(para *node) (map[string]string, bool, bool) {
	out := copyStyle(s.defaults)
	props := para.child(wordNS, "pPr")
	id := wordProperty(props, "pStyle")
	if id == "" {
		id = s.defaultParagraph
	}
	keep, br := false, false
	chain := s.chain(id)
	chain = append(chain, &node{children: []*node{props}})
	for _, style := range chain {
		s.run(out, style.child(wordNS, "rPr"))
		p := style.child(wordNS, "pPr")
		s.paragraph(out, p)
		if n := p.child(wordNS, "keepNext"); n != nil {
			keep = wordEnabled(n)
		}
		if n := p.child(wordNS, "pageBreakBefore"); n != nil {
			br = wordEnabled(n)
		}
	}
	return out, keep, br
}

func (s *wordStyles) cellStyle(tbl, cell *node, row, col int) map[string]string {
	out := map[string]string{}
	id := wordProperty(tbl.child(wordNS, "tblPr"), "tblStyle")
	if id == "" {
		id = s.defaultTable
	}
	for _, style := range s.chain(id) {
		s.cellProperties(out, style.child(wordNS, "tblPr"), true)
		s.cellProperties(out, style.child(wordNS, "tcPr"), false)
		for _, condition := range style.children {
			if condition.name.Space != wordNS || condition.name.Local != "tblStylePr" {
				continue
			}
			kind := condition.attr("type")
			if (kind == "firstRow" && row == 0) || (kind == "firstCol" && col == 0) {
				s.cellProperties(out, condition.child(wordNS, "tcPr"), false)
			}
		}
	}
	s.cellProperties(out, tbl.child(wordNS, "tblPr"), true)
	s.cellProperties(out, cell.child(wordNS, "tcPr"), false)
	return out
}
func (s *wordStyles) cellProperties(out map[string]string, props *node, table bool) {
	if props == nil {
		return
	}
	marginName, borderName := "tcMar", "tcBorders"
	if table {
		marginName, borderName = "tblCellMar", "tblBorders"
	}
	mar, borders := props.child(wordNS, marginName), props.child(wordNS, borderName)
	for _, side := range []string{"top", "right", "bottom", "left"} {
		key := strings.ToUpper(side[:1]) + side[1:]
		if v, ok := numberAttr(mar.child(wordNS, side), "w", 0, 1440); ok {
			out["padding"+key] = pixel(float64(v) / 15)
		}
		if border := s.border(borders.child(wordNS, side)); border != "" {
			out["border"+key] = border
		}
	}
	if table {
		if border := s.border(borders.child(wordNS, "insideH")); border != "" {
			out["borderTop"] = border
			out["borderBottom"] = border
		}
		if border := s.border(borders.child(wordNS, "insideV")); border != "" {
			out["borderLeft"] = border
			out["borderRight"] = border
		}
	}
	if shading := props.child(wordNS, "shd"); shading != nil && wordFill.MatchString(shading.attr("fill")) {
		out["backgroundColor"] = "#" + shading.attr("fill")
	}
}

func wordPage(body *node) *PageLayout {
	page := &PageLayout{Width: 12240, Height: 15840, Top: 1440, Right: 1440, Bottom: 1440, Left: 1440}
	section := body.child(wordNS, "sectPr")
	if section == nil {
		return page
	}
	size := section.child(wordNS, "pgSz")
	margin := section.child(wordNS, "pgMar")
	for _, v := range []struct {
		node *node
		name string
		dest *int
		max  int
	}{{size, "w", &page.Width, 43200}, {size, "h", &page.Height, 43200}, {margin, "top", &page.Top, 14400}, {margin, "right", &page.Right, 14400}, {margin, "bottom", &page.Bottom, 14400}, {margin, "left", &page.Left, 14400}} {
		if value, ok := numberAttr(v.node, v.name, 0, v.max); ok {
			*v.dest = value
		}
	}
	if page.Width-page.Left-page.Right < 1440 || page.Height-page.Top-page.Bottom < 1440 {
		return &PageLayout{12240, 15840, 1440, 1440, 1440, 1440}
	}
	return page
}
