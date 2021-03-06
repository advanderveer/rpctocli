package main

import (
	"errors"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"

	"github.com/codegangsta/cli"
)

type unexp struct{}

type arith int

//Test8 should is NOT rpc: method is not exported
func (t *Arith) test8(args *Args, reply *int) error {
	return nil
}

//Test7 should is NOT rpc: arith is not exported
func (t arith) Test7(args *Args, reply *int) error {
	return nil
}

//Test6 should is NOT rpc: return must be error
func (t Arith) Test6(args *Args, reply *int) int {
	return 0
}

//Test5 should is NOT rpc: one return only
func (t Arith) Test5(args *Args, reply *int) (int, error) {
	return 0, nil
}

//Test4 should is NOT rpc: second arg is pointer but not exported
func (t Arith) Test4(args *Args, reply *unexp) error {
	return nil
}

//Test3 should is NOT rpc: second arg is not a pointer
func (t Arith) Test3(args *Args, reply int) error {
	return nil
}

//Test2 should is NOT rpc: no two args
func (t Arith) Test2(reply *int) error {
	return nil
}

//Test is rpc: although receiver is not a pointer
func (t Arith) Test(args *Args, reply *int) error {
	return nil
}

//Multiply is rpc
func (t *Arith) Multiply(args *Args, reply *int) error {
	*reply = args.A * args.B
	return nil
}

//Squared multiplies the arg by itself
func (t *Arith) Squared(nr int, reply *int) error {
	*reply = nr * nr
	return nil
}

type Countable int

type TwoAble Countable

func (t *Arith) TimesTwo(nr TwoAble, reply *int) error {
	*reply = int(nr) * 2
	return nil
}

//Divide is rpc
func (t *Arith) Divide(args *Args, quo *Quotient) error {
	if args.B == 0 {
		return errors.New("divide by zero")
	}
	quo.Quo = args.A / args.B
	quo.Rem = args.A % args.B
	return nil
}

func main() {
	arith := new(Arith)
	rpc.Register(arith)
	rpc.HandleHTTP()
	l, e := net.Listen("tcp", "127.0.0.1:1234")
	if e != nil {
		log.Fatal("listen error:", e)
	}
	go http.Serve(l, nil)

	//use the generated commands
	app := cli.NewApp()
	app.Name = "boom"
	app.Usage = "make an explosive entrance"
	app.Commands = GeneratedCommands
	app.Run(os.Args)
}
