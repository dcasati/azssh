package main

import (
	"io"
	"log"

	"github.com/noelbundick/azssh/cmd"
)

func main() {
	log.SetFlags(0)
	log.SetOutput(io.Discard)

	cmd.Execute()
}
