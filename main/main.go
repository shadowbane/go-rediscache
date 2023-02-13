package main

import (
	"github.com/joho/godotenv"
	"log"
	"rediscache"
)

func main() {

	if err := godotenv.Load(); err != nil {
		log.Printf("Failed to load env vars!")
	}

	rediscache.Init(rediscache.LoadEnvForRedis())
}
