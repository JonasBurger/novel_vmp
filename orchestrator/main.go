package main

import (
	"log"

	"my.org/novel_vmp/internal/cli"
)

func main() {
	err := cli.NewRootCommand().Execute()
	if err != nil {
		log.Fatal(err)
	}
}

// func main() {
// 	server := master.NewServer()
// 	err := server.Start()
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// }
