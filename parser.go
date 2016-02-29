package goconst

import (
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	testSuffix = "_test.go"
)

type Parser struct {
	// Meant to be passed via New()
	path, ignore               string
	ignoreTests, matchConstant bool

	// Internals
	strs   Strings
	consts Constants
}

func New(path, ignore string, ignoreTests, matchConstant bool) *Parser {
	return &Parser{
		path:          path,
		ignore:        ignore,
		ignoreTests:   ignoreTests,
		matchConstant: matchConstant,

		// Initialize the maps
		strs:   Strings{},
		consts: Constants{},
	}
}

func (p *Parser) ParseTree() (Strings, Constants, error) {
	pathLen := len(p.path)
	// Parse recursively the given path if the recursive notation is found
	if pathLen >= 5 && p.path[pathLen-3:] == "..." {
		filepath.Walk(p.path[:pathLen-3], func(path string, f os.FileInfo, err error) error {
			if err != nil {
				log.Println(err)
				// resume walking
				return nil
			}

			if f.IsDir() {
				p.parseDir(path)
			}
			return nil
		})
	} else {
		p.parseDir(p.path)
	}
	return p.strs, p.consts, nil
}

func (p *Parser) parseDir(dir string) error {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, dir, func(info os.FileInfo) bool {
		valid, name := true, info.Name()

		if p.ignoreTests {
			if strings.HasSuffix(name, testSuffix) {
				valid = false
			}
		}

		if len(p.ignore) != 0 {
			match, err := regexp.MatchString(p.ignore, dir+name)
			if err != nil {
				log.Fatal(err)
				return true
			}
			if match {
				valid = false
			}
		}

		return valid
	}, 0)
	if err != nil {
		return err
	}

	for _, pkg := range pkgs {
		for fn, f := range pkg.Files {
			ast.Walk(&treeVisitor{
				fileSet:     fset,
				packageName: pkg.Name,
				fileName:    fn,
				p:           p,
			}, f)
		}
	}

	return nil
}

type Strings map[string][]ExtendedPos
type Constants map[string]ConstType

type ConstType struct {
	token.Position
	Name, packageName string
}

type ExtendedPos struct {
	token.Position
	packageName string
}
