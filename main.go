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
		log.Fatalf("Failed to parse and type-check files in directory '%s': %v", dir, err)
	}

	err = g.Extract()
	if err != nil {
		log.Fatalf("Failed to extract rpc info: %v", err)
	}

}

//A Package the generator is building a cli for
type Package struct {
	dir   string
	name  string
	files []*File
	defs  map[*ast.Ident]types.Object
}

func (pkg *Package) check(fs *token.FileSet, astFiles []*ast.File) error {
	pkg.defs = make(map[*ast.Ident]types.Object)
	config := types.Config{Importer: importer.Default(), FakeImportC: true}
	info := &types.Info{
		Defs: pkg.defs,
	}

	_, err := config.Check(pkg.dir, fs, astFiles, info)
	if err != nil {
		return fmt.Errorf("Failed to type check package: %v", err)
	}

	return nil
}

//A File in the package
type File struct {
	pkg  *Package
	file *ast.File
}

//The Generator itself
type Generator struct {
	pkg      *Package
	services map[string]*RPCService
}

//A RPCMethod is an in-memory representation of method on a rpc service
type RPCMethod struct {
	recv *types.Named
	sig  *types.Signature
}

//A RPCService is an in-memory representation of a parsed rpc service
type RPCService struct {
	methods map[string]*RPCMethod
}

//NewGenerator initializes a generator
func NewGenerator() *Generator {
	return &Generator{
		services: map[string]*RPCService{},
	}
}

//Extract rpc methods by analysing package definitions
func (g *Generator) Extract() error {
	rpcmethods := map[string]*RPCMethod{}

	for _, obj := range g.pkg.defs {
		if obj == nil {
			continue
		}

	DEFSWITCH:
		switch t := obj.Type().(type) {

		//as per "net/rpc" are we looking for methods with the following:
		// [x] the fn is a method (has receiver)
		// [x] the method's type is exported.
		// [x] the method is exported.
		// [x] the method has two arguments...
		// [x] both exported (or builtin) types.
		// [x] the method's second argument is a pointer.
		// [x] the method has return type error.
		case *types.Signature:

			//needs to be a method (have a receiver) and be exported
			if t.Recv() == nil || !obj.Exported() {
				break
			}

			//receiver must be named and exported
			var recvt types.Type
			if recvpt, ok := t.Recv().Type().(*types.Pointer); ok {
				recvt = recvpt.Elem()
			} else {
				recvt = t.Recv().Type()
			}

			recv, ok := recvt.(*types.Named)
			if ok {
				if !recv.Obj().Exported() {
					break
				}
			}

			//method must have two params
			if t.Params().Len() != 2 {
				break
			}

			//all param types must be exported or builtin
			for i := 0; i < t.Params().Len(); i++ {
				var paramt types.Type
				if parampt, ok := t.Params().At(i).Type().(*types.Pointer); ok {
					paramt = parampt.Elem()
				} else {

					//second arg must be a pointer
					if i == 1 {
						break DEFSWITCH
					}

					paramt = t.Params().At(i).Type()
				}

				//if param type is Named, it must be exported, else it must be buildin
				if paramnamedt, ok := paramt.(*types.Named); ok {
					if !paramnamedt.Obj().Exported() {
						break DEFSWITCH
					}
				} else if strings.Contains(paramt.String(), ".") {
					break
				}
			}

			//must have one result: error
			if t.Results().Len() != 1 || t.Results().At(0).Type().String() != "error" {
				break
			}

			rpcmethods[obj.Name()] = &RPCMethod{
				recv: recv,
				sig:  t,
			}

			// log.Printf("%s on %s (pos: %d) ", obj.Name(), recv.Obj().Name(), recv.Obj().Pos())
			//
			// for _, f := range g.pkg.files {
			// 	nodes, _ := astutil.PathEnclosingInterval(f.file, recv.Obj().Pos(), recv.Obj().Pos())
			// 	if len(nodes) == 0 {
			// 		continue
			// 	}
			//
			// 	for _, n := range nodes {
			// 		switch n := n.(type) {
			// 		case *ast.GenDecl:
			// 			log.Println("\n", n.Doc.Text())
			// 		case *ast.FuncDecl:
			// 			log.Println("\n", n.Doc.Text())
			// 		}
			// 	}
			// }

			//recv pos:
			// n, _ := astutil.PathEnclosingInterval(root *ast.File, start, end token.Pos) (path []ast.Node, exact bool)

			//results:
		}
	}

	for n, rpcm := range rpcmethods {
		rpcs, ok := g.services[rpcm.recv.Obj().Name()]
		if !ok {
			rpcs = &RPCService{
				methods: map[string]*RPCMethod{},
			}

			g.services[rpcm.recv.Obj().Name()] = rpcs
		}

		rpcs.methods[n] = rpcm
	}

	log.Printf("%+v", g.services["Arith"].methods["Multiply"].sig.String())

	return nil
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

		astFile, err := parser.ParseFile(fset, fi.Name(), nil, parser.ParseComments)
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
	return g.pkg.check(fset, astFiles)
}
