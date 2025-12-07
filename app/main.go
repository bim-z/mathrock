package main

import "github.com/charmbracelet/fang"

func main() {
	_ = fang.Execute(root.Context(), root, fang.WithVersion("0.0.1"))
}
