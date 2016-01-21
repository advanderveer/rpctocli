package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"io/ioutil"
	"log"
	"os"
	"strings"
)

var (
	typeNames = flag.String("type", "", "comma-separated list of type names; must be set")
)

func usage() {
	fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
	flag.PrintDefaults()
}

func main() {
	flag.Usage = usage
	flag.Parse()
	log.SetPrefix("rpctocli: ")
	log.SetFlags(0)

	g := NewGenerator()
	dir, err := os.Getwd()
	if err != nil {
		log.Fatalf("Failed to get working directory: %v", err)
	}

	err = g.Parse(dir)
	if err != nil {
		log.Fatalf("Failed to parse files in directory '%s': %v", dir, err)
	}

}

//The Generator itself
type Generator struct {
	pkg *Package
}

//A Package the generator is building a cli for
type Package struct {
	name     string
	files    []*File
	typesPkg *types.Package
}

//A File in the package
type File struct {
	pkg  *Package
	file *ast.File
}

//NewGenerator initializes a generator
func NewGenerator() *Generator {
	return &Generator{}
}

//Parse all .go files in a directory
func (g *Generator) Parse(dir string) error {
	g.pkg = &Package{}

	fis, err := ioutil.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("Failed to read the directory '%s': %v", dir, err)
	}

	fset := token.NewFileSet()
	for _, fi := range fis {
		if !strings.HasSuffix(fi.Name(), ".go") {
			continue
		}

		astFile, err := parser.ParseFile(fset, fi.Name(), nil, 0)
		if err != nil {
			return fmt.Errorf("Failed to parse file '%s': %v", fi.Name(), err)
		}

		log.Printf("%+v", astFile)

		g.pkg.files = append(g.pkg.files, &File{
			pkg:  g.pkg,
			file: astFile,
		})
	}

	if len(g.pkg.files) == 0 {
		log.Fatalf("%s: no buildable Go files", dir)
	}

	g.pkg.name = g.pkg.files[0].file.Name.Name
	log.Printf("parsed package '%s' in '%s'!", g.pkg.name, dir)
	return nil
}
