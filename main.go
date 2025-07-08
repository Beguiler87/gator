package main

import (
	"fmt"

	"github.com/Beguiler87/gator/internal/config"
)

func main() {
	incoming, err := config.Read()
	if err != nil {
		fmt.Println("error reading config:", err)
		return
	}
	err = incoming.SetUser("Andrew")
	if err != nil {
		fmt.Println("error setting user:", err)
		return
	}
	updated, err := config.Read()
	if err != nil {
		fmt.Println("error reading updated config:", err)
		return
	}
	fmt.Println(updated)
}
