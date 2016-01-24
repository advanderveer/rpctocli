package main

//Arith is a struct that we want to clify
type Arith int

//Args is en axample cli input
type Args struct {
	A        int `cli:"a"`
	B        int `cli:"b"`
	d        int
	NonBasic *Quotient
}

//Quotient is an example output
type Quotient struct {
	Quo, Rem int
}
