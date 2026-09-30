package office

import (
	"bytes"
	"fmt"
	"nativepane/internal/testdoc"
	"strings"
	"testing"
)

func TestLargeDocumentsRoundTrip(t *testing.T) {
	const count = 25001
	for _, format := range []string{"docx", "xlsx", "pptx"} {
		t.Run(format, func(t *testing.T) {
			source := testdoc.LargeFile(format, count)
			engine := OOXML{}
			model, err := engine.Open(source, format)
			if err != nil {
				t.Fatal(err)
			}
			var fields []Field
			for _, block := range model.Blocks {
				fields = append(fields, block.Fields...)
			}
			if len(fields) != count {
				t.Fatalf("got %d fields, want %d", len(fields), count)
			}
			edits := []Edit{{fields[count-1].ID, "Changed beyond the old limit"}}
			// Exercise a full-document patch too: bounded spans are copied in one pass.
			if format == "docx" {
				edits = make([]Edit, count)
				for i, field := range fields {
					edits[i] = Edit{field.ID, fmt.Sprintf("Edited %d", i)}
				}
			}
			output, err := engine.Apply(source, format, edits)
			if err != nil {
				t.Fatal(err)
			}
			reopened, err := engine.Open(output, format)
			if err != nil {
				t.Fatal(err)
			}
			var afterFields []Field
			for _, block := range reopened.Blocks {
				afterFields = append(afterFields, block.Fields...)
			}
			if len(afterFields) != count {
				t.Fatal("fields were dropped")
			}
			// IDs remain source-order stable across all replacement lengths.
			if afterFields[count-1].ID != edits[len(edits)-1].ID || afterFields[count-1].Text != edits[len(edits)-1].Text {
				t.Fatal("last-field edit lost")
			}
			if format == "docx" {
				for i, edit := range edits {
					if afterFields[i].Text != edit.Text {
						t.Fatalf("edit %d lost", i)
					}
				}
			}
			before, after := unzip(t, source), unzip(t, output)
			changed := 0
			for name, content := range before {
				if !bytes.Equal(content, after[name]) {
					changed++
				}
			}
			if len(before) != len(after) || changed != 1 {
				t.Fatal("unexpected package changes")
			}
		})
	}
}

func TestConfiguredDocumentCapacity(t *testing.T) {
	engine := OOXML{MaxFields: 10}
	source := testdoc.LargeFile("xlsx", 10)
	model, err := engine.Open(source, "xlsx")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := engine.Apply(source, "xlsx", []Edit{{model.Blocks[0].Fields[9].ID, "42"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := engine.Open(testdoc.LargeFile("xlsx", 11), "xlsx"); err == nil || !strings.Contains(err.Error(), "MAX_DOCUMENT_FIELDS") {
		t.Fatalf("unhelpful capacity error: %v", err)
	}
}
