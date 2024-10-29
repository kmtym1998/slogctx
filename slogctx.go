package slogctx

import (
	"fmt"
	"go/ast"
	"slices"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

const doc = `slogctx は log/slog を使ってログを吐くときに context.Context が呼び元から渡されているかどうかを検証します。
例えば、slog.InfoContext() ではなく slog.Info() が呼ばれている場合に警告を出します。`

// Analyzer is ...
var Analyzer = &analysis.Analyzer{
	Name: "slogctx",
	Doc:  doc,
	Run:  run,
	Requires: []*analysis.Analyzer{
		inspect.Analyzer,
	},
}

var target string

func init() {
	Analyzer.Flags.StringVar(&target, "target", "", "走査対象とするファイル名。複数ファイルを指定する場合はカンマ区切りにする。指定しない場合は全ファイルを対象とする。")
}

// Analyzer のエントリーポイント。パッケージごとに呼び出される。
// testdata ディレクトリ配下のファイルは無視される。
func run(pass *analysis.Pass) (any, error) {
	for _, file := range pass.Files {
		currentFilePath := pass.Fset.File(file.Pos()).Name()
		fmt.Println("👺 currentFilePath", currentFilePath)

		// log/slog または golang.org/x/exp/slog が import されているかどうかを確認する
		hasSlogImport := slices.ContainsFunc(file.Imports, func(item *ast.ImportSpec) bool {
			path := strings.Trim(item.Path.Value, "\"")
			return path == "log/slog" || path == "golang.org/x/exp/slog"
		})

		// slog が import されていない場合はスキップ
		if !hasSlogImport {
			continue
		}

		// slog が import されている場合は slog が使われている箇所を検証する
		// メソッド、または関数となっている Debug, Info, Warn, Error, Log が呼ばれているかどうかを確認する
		inspectorImpl := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
		nodeFilter := []ast.Node{
			// TODO FOR NEXT TIME
			// ここに走査対象のノードを追加する
			// https://chatgpt.com/c/67202f0b-eee8-8000-a925-7fc64e2aab7e
			(*ast.FuncDecl)(nil),
			(*ast.CallExpr)(nil),
			(*ast.ValueSpec)(nil),
		}
		inspectorImpl.Preorder(nodeFilter, func(n ast.Node) {
			fmt.Printf("👺 n: %T\n", n)
			switch x := n.(type) {
			case *ast.CallExpr:
				if x.Fun == nil {
					return
				}
				ident, ok := x.Fun.(*ast.Ident)
				if !ok {
					return
				}
				fmt.Println("👺 ident.Name", ident.Name)
			}
		})

	}
	// inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	return nil, nil
}
