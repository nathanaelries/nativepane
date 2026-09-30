// Collect notices from the exact Go packages compiled into NativePane.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

type pkg struct {
	ImportPath, Dir         string
	Standard                bool
	GoFiles, SFiles, HFiles []string
	Module                  *struct {
		Path string
		Main bool
	}
}

func main() {
	target := "THIRD_PARTY_NOTICES.txt"
	if len(os.Args) > 1 {
		target = os.Args[1]
	}
	command := exec.Command("go", "list", "-deps", "-json", ".")
	raw, e := command.Output()
	if e != nil {
		log.Fatal(e)
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	notices := map[string]string{}
	var inventory []string
	root := runtime.GOROOT()
	base, e := os.ReadFile(filepath.Join(root, "LICENSE"))
	if e != nil {
		log.Fatal(e)
	}
	notices["Go LICENSE"] = string(base)
	for {
		var p pkg
		e = decoder.Decode(&p)
		if e == io.EOF {
			break
		}
		if e != nil {
			log.Fatal(e)
		}
		if p.Module != nil && !p.Module.Main {
			log.Fatalf("external module requires license review: %s", p.Module.Path)
		}
		if !p.Standard {
			continue
		}
		inventory = append(inventory, p.ImportPath)
		for dir := p.Dir; strings.HasPrefix(dir, root) && dir != root; dir = filepath.Dir(dir) {
			entries, _ := os.ReadDir(dir)
			for _, entry := range entries {
				name := strings.ToUpper(entry.Name())
				if !entry.IsDir() && (strings.HasPrefix(name, "LICENSE") || strings.HasPrefix(name, "NOTICE") || name == "PATENTS" || name == "COPYING" || name == "AUTHORS") {
					path := filepath.Join(dir, entry.Name())
					b, e := os.ReadFile(path)
					if e != nil {
						log.Fatal(e)
					}
					rel, _ := filepath.Rel(root, path)
					notices[filepath.ToSlash(rel)] = string(b)
				}
			}
		}
		files := append(append(p.GoFiles, p.SFiles...), p.HFiles...)
		for _, file := range files {
			b, e := os.ReadFile(filepath.Join(p.Dir, file))
			if e != nil {
				log.Fatal(e)
			}
			text := string(b)
			if i := strings.Index(text, "\npackage "); i >= 0 {
				text = text[:i]
			}
			lines := strings.Split(text, "\n")
			if len(lines) > 200 {
				lines = lines[:200]
			}
			text = strings.Join(lines, "\n")
			if strings.Contains(strings.ToLower(text), "copyright") {
				rel, _ := filepath.Rel(root, filepath.Join(p.Dir, file))
				notices[filepath.ToSlash(rel)+" (source preamble)"] = text
			}
		}
	}
	var out strings.Builder
	fmt.Fprintf(&out, "NativePane third-party notices\nToolchain: %s\nTarget: %s/%s\n\nNo external Go modules or JavaScript dependencies.\nStandard-library package inventory:\n", runtime.Version(), runtime.GOOS, runtime.GOARCH)
	sort.Strings(inventory)
	for _, p := range inventory {
		fmt.Fprintln(&out, p)
	}
	keys := make([]string, 0, len(notices))
	for k := range notices {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Fprintf(&out, "\n===== %s =====\n%s\n", k, notices[k])
	}
	if e = os.WriteFile(target, []byte(out.String()), 0644); e != nil {
		log.Fatal(e)
	}
}
