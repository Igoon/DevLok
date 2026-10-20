package main

import (
	"os"

	"devlok/internal/cli"
)

func main() {
	os.Exit(cli.Execute())
}
