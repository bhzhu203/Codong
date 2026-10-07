package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/codong-lang/codong/engine/goirgen"
	"github.com/codong-lang/codong/engine/lexer"
	"github.com/codong-lang/codong/engine/parser"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "Usage: dumpgen <input.cod> <output.go>")
		os.Exit(2)
	}
	src, err := os.ReadFile(os.Args[1])
	if err != nil {
		panic(err)
	}
	// Imports resolve against the input file's directory, as in the runner.
	sourceDir := filepath.Dir(os.Args[1])
	if abs, err := filepath.Abs(sourceDir); err == nil {
		sourceDir = abs
	}
	l := lexer.New(string(src))
	p := parser.New(l)
	prog := p.ParseProgram()
	if len(p.Errors()) > 0 {
		for _, e := range p.Errors() {
			fmt.Println("ERR:", e)
		}
		os.Exit(1)
	}
	out := goirgen.Generate(prog, sourceDir)
	os.WriteFile(os.Args[2], []byte(out), 0644)
	fmt.Println("wrote", os.Args[2])
}
