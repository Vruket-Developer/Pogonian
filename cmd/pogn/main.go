package main

import (
	"os"

	"pogonian/internal/cli"
)

func main() {
	if err := cli.Run(os.Args[1:]); err != nil {
		cli.ExitOnError(err)
	}
}
