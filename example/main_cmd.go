// Code generated by "rpctocli "; DO NOT EDIT
package main

import(
	"github.com/codegangsta/cli"
)

var ArithCommand = cli.Command{Name: "Arith",Subcommands: []cli.Command{MultiplySubCommand, DivideSubCommand, TestSubCommand, },}

var MultiplySubCommand = cli.Command{Name: "Multiply",Action: func(ctx *cli.Context) {},}

var DivideSubCommand = cli.Command{Name: "Divide",Action: func(ctx *cli.Context) {},}

var TestSubCommand = cli.Command{Name: "Test",Action: func(ctx *cli.Context) {},}

var GeneratedCommands = []cli.Command{ArithCommand, }