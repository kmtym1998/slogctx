package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	rootDir := flag.String("dir", "./", "指定されたディレクトリ配下にある .go ファイルを再帰的に探索します")
	if FromPtr(rootDir) == "" {
		panic("'dir' が指定されていません")
	}
	flag.Parse()

	bannedFuncIdentifiers := []string{
		"Debug",
		"Info",
		"Warn",
		"Error",
	}
	var errList []error
	if err := filepath.WalkDir(*rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			errList = append(errList, err)
			return nil
		}

		// .go ファイル以外はスキップ
		if filepath.Ext(path) != ".go" {
			return nil
		}

		// 設定ファイルの類はスキップ
		if strings.HasPrefix(path, ".") {
			return nil
		}

		f, err := os.Open(path)
		if err != nil {
			errList = append(errList, err)
			return nil
		}
		defer f.Close()

		// ファイルを1行ずつ読み込む
		scanner := bufio.NewScanner(f)
		count := 0
		skip := false
		for scanner.Scan() {
			count++
			if skip {
				skip = false
				continue
			}

			line := scanner.Text()
			for _, bannedIdent := range bannedFuncIdentifiers {
				lineWithoutTab := strings.ReplaceAll(line, "\t", "")
				if strings.Contains(lineWithoutTab, "nolint:slogctx") {
					skip = true
					continue
				}

				// コメントアウト行はスキップ
				if strings.HasPrefix(lineWithoutTab, "//") {
					continue
				}

				if !strings.Contains(line, "."+bannedIdent+"(") {
					continue
				}

				fmt.Printf(
					"%s:%d %s メソッドが呼ばれています。%sContext を使うか、// nolint:slogctx をつけるか、関数名を変更してください\n",
					path, count, bannedIdent, bannedIdent,
				)
			}
		}

		return nil
	}); err != nil {
		panic(err)
	}
}

// from samber/lo
func FromPtr[T any](x *T) T {
	if x == nil {
		return Empty[T]()
	}

	return *x
}

func Empty[T any]() T {
	var zero T
	return zero
}
