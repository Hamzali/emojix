package main

import (
	"emojix"
	"fmt"
	"log"
)

func main() {
	fmt.Println("server is runnning on 9000...")

	e, err := emojix.NewEmojix()

	if err != nil {
		log.Printf("failed to init err: %v", err)
		return
	}

	e.StartServer()
}
