package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/importer"
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

	for _, f := range g.pkg.files {
		ast.Inspect(f.file, func(node ast.Node) bool {

		DECLSWITCH:
			switch decl := node.(type) {
			case *ast.GenDecl:
				if decl.Tok != token.TYPE {
					break
				}

				for _, spec := range decl.Specs {
					tspec := spec.(*ast.TypeSpec)

					//we only care about exported types as thats a requirement
					//for associated rpc methods
					if !ast.IsExported(tspec.Name.String()) {
						break
					}

					log.Printf("type %v", tspec.Name.String())

					//for Arg types we may need to look at fields and
					//if not just skip
					t, ok := tspec.Type.(*ast.StructType)
					if !ok {
						continue
					}

					for _, f := range t.Fields.List {
						log.Printf("\tfield %s", f.Names)
					}
				}

			case *ast.FuncDecl:

				//as per "net/rpc" are we looking for methods with the following:
				// [x] the fn is a method (has receiver)
				// [x] the method's type is exported.
				// [x] the method is exported.
				// [x] the method has two arguments...
				// [ ] both exported (or builtin) types.
				// [x] the method's second argument is a pointer.
				// [x] the method has return type error.

				//additionally,
				// - argType (T1) fields can have annotations to customize cli options
				// - replyType (T2), cli output?

				//func must be exported
				if !ast.IsExported(decl.Name.Name) {
					break
				}

				//function needs to be a method and thus have a receiver
				if decl.Recv == nil || decl.Recv.NumFields() == 0 {
					break
				}

				//get receiver identifier, possibly a pointer
				var recvid *ast.Ident
				if recvexp, ok := decl.Recv.List[0].Type.(*ast.StarExpr); !ok {
					recvid, ok = decl.Recv.List[0].Type.(*ast.Ident)
					if !ok {
						log.Printf("Func declaration '%s' had a receiver that was not a StarExpr or Ident, skipping...", decl.Name)
						break
					}
				} else {
					recvid, ok = recvexp.X.(*ast.Ident)
					if !ok {
						log.Printf("Func declaration '%s' pointer receiver didn't have Ident, skipping...", decl.Name)
						break
					}
				}

				//receiver type must be exported
				if !ast.IsExported(recvid.Name) {
					break
				}

				//must have two parameters
				if decl.Type.Params.NumFields() != 2 {
					break
				}

				for i, paramf := range decl.Type.Params.List {
					var paramid *ast.Ident
					if paramexp, ok := paramf.Type.(*ast.StarExpr); !ok {

						//second param MUST be a pointer
						if i == 1 {
							break DECLSWITCH
						}

						paramid, ok = paramf.Type.(*ast.Ident)
						if !ok {
							log.Printf("Func declaration '%s' had a param that was not a StarExpr or Ident, skipping...", decl.Name)
							break
						}
					} else {
						paramid, ok = paramexp.X.(*ast.Ident)
						if !ok {
							log.Printf("Func declaration '%s' param didn't have Ident, skipping...", decl.Name)
							break
						}
					}

					//both (all) param types must be exported or builtin
					//@todo implment this
					// if !ast.IsExported(paramid.String()) {
					// 	break DECLSWITCH
					// }
					_ = paramid
				}

				//must have one result
				if decl.Type.Results.NumFields() != 1 {
					break
				}

				//result must be an simple identifier (named error)
				retid, ok := decl.Type.Results.List[0].Type.(*ast.Ident)
				if !ok || retid.Name != "error" {
					break
				}

				log.Printf("func %s", decl.Name)
				log.Printf("\trecv: %+v", recvid.Name)

				// //second param must be pointer
				// if _, ok := decl.Type.Params.List[1].Type.(*ast.StarExpr); !ok {
				// 	break
				// }

				// log.Printf("func %s", decl.Name)
				// log.Printf("\trecv: %+v", recvid.Name)

			}

			// 36			case *ast.BasicLit:
			// 37				s = x.Value
			// 38			case *ast.Ident:
			// 39				s = x.Name
			// 40			}

			//we care only about generic declaration node
			// gdecl, ok := node.(*ast.GenDecl)
			// if !ok || gdecl.Tok != token.TYPE {
			// 	return true
			// }

			return true
		})
	}
}

//The Generator itself
type Generator struct {
	pkg *Package
}

//A Package the generator is building a cli for
type Package struct {
	dir      string
	name     string
	files    []*File
	defs     map[*ast.Ident]types.Object
	typesPkg *types.Package
}

func (pkg *Package) check(fs *token.FileSet, astFiles []*ast.File) error {
	pkg.defs = make(map[*ast.Ident]types.Object)
	config := types.Config{Importer: importer.Default(), FakeImportC: true}
	info := &types.Info{
		Defs: pkg.defs,
	}
	typesPkg, err := config.Check(pkg.dir, fs, astFiles, info)
	if err != nil {
		return fmt.Errorf("Failed to type check package: %v", err)
	}

	pkg.typesPkg = typesPkg
	return nil
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

	astFiles := []*ast.File{}
	fset := token.NewFileSet()
	for _, fi := range fis {
		if !strings.HasSuffix(fi.Name(), ".go") {
			continue
		}

		astFile, err := parser.ParseFile(fset, fi.Name(), nil, 0)
		if err != nil {
			return fmt.Errorf("Failed to parse file '%s': %v", fi.Name(), err)
		}

		astFiles = append(astFiles, astFile)
		g.pkg.files = append(g.pkg.files, &File{
			pkg:  g.pkg,
			file: astFile,
		})
	}

	if len(g.pkg.files) == 0 {
		log.Fatalf("%s: no buildable Go files", dir)
	}

	g.pkg.dir = dir
	g.pkg.name = g.pkg.files[0].file.Name.Name
	log.Printf("parsed package '%s' in '%s'!", g.pkg.name, dir)

	return g.pkg.check(fset, astFiles)
}
