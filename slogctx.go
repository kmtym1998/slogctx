package slogctx

import (
	"go/ast"
	"go/importer"
	"go/token"
	"go/types"
	"slices"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
	"golang.org/x/tools/go/packages"
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

	// import alias をマッピング
	importAliasList := map[string]string{}
	// 型情報をファイル毎にまとめる
	typeInfoList := map[string]*types.Info{}
	for _, file := range pass.Files {
		typeInfo, err := getTypeInfoInAFile(pass.Fset, file)
		if err != nil {
			typesError, ok := err.(types.Error)
			if !ok {
				continue
			}

			if strings.Contains(typesError.Msg, "could not import") {
			}
			continue
		}
		filePath := file.Name.Name
		typeInfoList[filePath] = typeInfo

		for _, decl := range file.Decls {
			genDecl, ok := decl.(*ast.GenDecl)
			if !ok || genDecl.Tok != token.IMPORT {
				// GenDecl でない場合、または import でない場合はスキップ
				continue
			}

			for _, spec := range genDecl.Specs {
				importSpec := spec.(*ast.ImportSpec)
				if importSpec.Name != nil {
					// import alias がつけられている場合
					importAliasList[importSpec.Name.Name] = importSpec.Path.Value // "foo" -> "io"
				}
			}
		}
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

			// NOTE: SelectorExpr における X (= Expression) はベースとなる値やパッケージ
			// SelectorExpr における Sel (= Selector) はアクセスするメソッドやフィールド
			if funExpr.X == nil || funExpr.Sel == nil {
				return
			}

			// "." の右側の呼び出し先 (関数、メソッド etc.) が bannedIdentifiers に含まれている場合、
			// "." の左側の呼び出し元 (変数, 関数呼び出し結果の返り値, フィールド, etc.) の型が slog.Logger 構造体 または slog パッケージであるかどうかを確認する
			if !slices.Contains(bannedIdentifiers, funExpr.Sel.Name) {
				return
			}

			// TODO FOR NEXT TIME

			funExprXIdent, ok := funExpr.X.(*ast.Ident)
			if !ok {
				return
			}

			// log/slog を import しているかどうかを確認
			importPath, found := importAliasList[funExprXIdent.Name]
			if !found && funExprXIdent.Name != "slog" {
				return
			}
			if importPath != `"log/slog"` && funExprXIdent.Name != "slog" {
				return
			}

			// いずれにも該当しない場合、bannedIdentifiers に含まれる log/slog の関数・メソッドが呼ばれていると判断
			// FIXME; "slog" というというインスタンス変数から (log/slog のものでない) Info や Debug を呼んだ場合もレポート対象となってしまう
			pass.Reportf(x.Pos(), "log/slog の %s が呼ばれています", funExpr.Sel.Name)
			return

			// NOTE: log/slog が別名 import されているケースがある
		}
	})

	// inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	return nil, nil
}

func getTypeInfoInAFile(fileSet *token.FileSet, fileNode *ast.File) (*types.Info, error) {
	conf := types.Config{
		Importer:                 importer.Default(),
		DisableUnusedImportCheck: true,
	}
	info := &types.Info{
		Uses: make(map[*ast.Ident]types.Object),
	}
	_, err := conf.Check("pkg", fileSet, []*ast.File{fileNode}, info)
	if err != nil {
		return nil, err
	}

	return info, nil
}

func getTypeInfoInAFileV2() (*types.Info, error) {
	conf := &packages.Config{
		Mode: packages.NeedTypes | packages.NeedImports,
	}
	pkgs, err := packages.Load(conf, ".")
	if err != nil {
		return nil, err
	}
	// `pkgs[0].TypesInfo`が希望の型情報になるはずです
	return pkgs[0].TypesInfo, nil
}
