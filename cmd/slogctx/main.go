package main

import (
	"github.com/kmtym1998/slogctx"
	"golang.org/x/tools/go/analysis/unitchecker"
)

func main() { unitchecker.Main(slogctx.Analyzer) }
