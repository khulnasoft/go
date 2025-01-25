package main

import (
	"context"
	"log"
	"os"

	"github.com/khulnasoft/go/run"
)

func main() {
	if len(os.Args) != 2 {
		log.Fatal("Expected jq argument")
	}

	ctx := context.Background()
	res, err := run.Cmd(ctx, "cat").Input(os.Stdin).Run().JQ(os.Args[1])
	if err != nil {
		log.Fatal(err.Error())
	}
	println(string(res))
}
