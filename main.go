package main

import (
	"os"

	"github.com/zhide915/footlight/internal/cli"
)

func main() {
	os.Exit(cli.Main(os.Args[1:]))
}
