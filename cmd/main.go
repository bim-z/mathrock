package main

import "github.com/charmbracelet/fang"

func main() {
	fang.WithVersion("0.0.1")
	_ = fang.Execute(root.Context(), root)
}
