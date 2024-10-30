package slogctx

import (
	"fmt"
	"go/ast"
	"slices"

	"github.com/kr/pretty"
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
	// FIXME: 調査対象のファイルを絞れるようにしたい
	// FIXME: nolint で ignore できるようにしたい

	// メソッド、または関数となっている Debug, Info, Warn, Error, Log が呼ばれているかどうかを確認する
	inspectorImpl := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	nodeFilter := []ast.Node{
		(*ast.CallExpr)(nil),
	}

	bannedIdentifiers := []string{"Debug", "Info", "Warn", "Error", "Log"}
	inspectorImpl.Preorder(nodeFilter, func(n ast.Node) {
		switch x := n.(type) {
		// NOTE: 関数やメソッドの呼び出しが Call Expression (ast.CallExpr)
		case *ast.CallExpr:
			// NOTE: `slog.Debug()` や `http.DefaultClient.Do()` など、`.` で繋がった式が Selector Expression (ast.SelectorExpr)
			// `.` の左側にはパッケージや何かしらの値 (変数、定数、構造体のフィールド etc.) が来る
			// 右側には何かしらの値、または関数・メソッドの呼び出しが来る。
			// ここでは x が CallExpr なので、x.Fun が SelectorExpr であるかどうかを確認する
			funExpr, ok := x.Fun.(*ast.SelectorExpr)
			if !ok {
				return
			}

			selectorExpr, ok := funExpr.X.(*ast.Ident)
			if !ok {
				return
			}

			// FIXME: import name を変えられていた場合を考慮できてない
			if selectorExpr.Name == "slog" && slices.Contains(bannedIdentifiers, funExpr.Sel.Name) {
				// slog が使われている箇所を検証する
				pass.Reportf(x.Pos(), "slog.%s が呼ばれています", funExpr.Sel.Name)
			}

			// TODO FOR NEXT TIME

			if selectorExpr.Obj != nil {
				pretty.Println("🏀 selectorExpr.Obj.Decl", selectorExpr.Obj.Decl)
				objDeclAssignStmt, ok := selectorExpr.Obj.Decl.(*ast.AssignStmt)
				if !ok {
					fmt.Printf("👺 %T は ast.AssignStmt ではないためスキップします\n", selectorExpr)
					return
				}

				pretty.Println("🏀 objDeclAssignStmt", objDeclAssignStmt)
			}
			return
		}
	})

	// inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	return nil, nil
}
