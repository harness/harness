package main

import (
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

func findBoxes(pkg *build.Package) map[string]bool {
	// create one list of files for this package
	filenames := make([]string, 0, len(pkg.GoFiles)+len(pkg.CgoFiles))
	filenames = append(filenames, pkg.GoFiles...)
	filenames = append(filenames, pkg.CgoFiles...)

	// create map of boxes to embed
	var boxMap = make(map[string]bool)

	// loop over files, search for rice.FindBox(..) calls
	for _, filename := range filenames {
		// find full filepath
		fullpath := filepath.Join(pkg.Dir, filename)
		if strings.HasSuffix(filename, "rice-box.go") {
			// Ignore *.rice-box.go files
			verbosef("skipping file %q\n", fullpath)
			continue
		}
		verbosef("scanning file %q\n", fullpath)

		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, fullpath, nil, 0)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		var riceIsImported bool
		ricePkgName := "rice"
		for _, imp := range f.Imports {
			if strings.HasSuffix(imp.Path.Value, "go.rice\"") {
				if imp.Name != nil {
					ricePkgName = imp.Name.Name
				}
				riceIsImported = true
				break
			}
		}
		if !riceIsImported {
			// Rice wasn't imported, so we won't find a box.
			continue
		}
		if ricePkgName == "_" {
			// Rice pkg is unnamed, so we won't find a box.
			continue
		}

		// Inspect AST, looking for calls to (Must)?FindBox.
		// First parameter of the func must be a basic literal.
		// Identifiers won't be resolved.
		var nextIdentIsBoxFunc bool
		var nextBasicLitParamIsBoxName bool
		ast.Inspect(f, func(node ast.Node) bool {
			if node == nil {
				return false
			}
			switch x := node.(type) {
			case *ast.Ident:
				if nextIdentIsBoxFunc || ricePkgName == "." {
					nextIdentIsBoxFunc = false
					if x.Name == "FindBox" || x.Name == "MustFindBox" {
						nextBasicLitParamIsBoxName = true
					}
				} else {
					if x.Name == ricePkgName {
						nextIdentIsBoxFunc = true
					}
				}
			case *ast.BasicLit:
				if nextBasicLitParamIsBoxName && x.Kind == token.STRING {
					nextBasicLitParamIsBoxName = false
					// trim "" or ``
					name := x.Value[1 : len(x.Value)-1]
					boxMap[name] = true
					verbosef("\tfound box %q\n", name)
				}

			default:
				if nextIdentIsBoxFunc {
					nextIdentIsBoxFunc = false
				}
				if nextBasicLitParamIsBoxName {
					nextBasicLitParamIsBoxName = false
				}
			}
			return true
		})
	}

	return boxMap
}
