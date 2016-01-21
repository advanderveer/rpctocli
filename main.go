package main

import (
	"flag"
	"log"
)

func main() {
	flag.Parse()
	log.SetPrefix("rpctocli: ")
	log.SetFlags(0)

	log.Println("hello world")
}
