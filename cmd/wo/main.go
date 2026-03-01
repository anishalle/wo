package main

import (
	"os"

	"github.com/anishalle/wo/internal/cli"
)

var version = "dev"

func main() {
	os.Exit(cli.Execute(version))
}
