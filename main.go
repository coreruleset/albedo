package main

import (
	"log"

	"github.com/coreruleset/albedo/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		log.Fatal(err.Error())
	}
}
