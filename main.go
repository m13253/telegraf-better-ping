package main

import (
	"log"
	"os"

	"github.com/m13253/telegraf-better-ping/params"
)

func main() {
	params := params.ParseParams(os.Args)
	state, err := NewApp(&params)
	if err != nil {
		log.Fatalln(err)
	}
	state.startReceivers()
	state.startSenders()
}
