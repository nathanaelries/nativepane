package main

import (
	"log"
	"nativepane/internal/testdoc"
	"os"
	"path/filepath"
)

func main() {
	dir := "samples"
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}
	if e := os.MkdirAll(dir, 0755); e != nil {
		log.Fatal(e)
	}
	for _, format := range []string{"docx", "xlsx", "pptx"} {
		if e := os.WriteFile(filepath.Join(dir, "welcome."+format), testdoc.File(format), 0644); e != nil {
			log.Fatal(e)
		}
	}
	if e := os.WriteFile(filepath.Join(dir, "tables.docx"), testdoc.TableFile(), 0644); e != nil {
		log.Fatal(e)
	}
}
