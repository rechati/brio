package main

import (
	"github.com/rechati/brio/cmd"
	_ "github.com/rechati/brio/cmd/plugins"
)

func main() {
	cmd.Execute()
}
