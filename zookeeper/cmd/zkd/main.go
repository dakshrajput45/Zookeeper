package main

import (
	"log"

	"zookeeper/internal/app"
)

const defaultAddress = ":8080"

func main() {
	server := app.NewServer(defaultAddress)
	if err := server.Start(); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
