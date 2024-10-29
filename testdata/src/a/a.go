package a

import (
	"fmt"
	"log/slog"
)

func F() {
	// The pattern can be written in regular expression.
	var gopher int // want "pattern"
	print(gopher)  // want "identifier is gopher"
	fmt.Println(gopher)
	slog.Info("Hello, World!")
}
