package main

import (
	"log"

	"github.com/sulewicz/redis-geo-sample/server"
)

const INDEX_NAME = "alaska"

func main() {
	server := server.New()
	if err := server.Bootstrap("localhost:6379", "alaska", "assets/Alaska.geojson"); err != nil {
		log.Println("Setup failed, but moving on with static content... ")
		log.Println(err)
	}

	if err := server.Run(); err != nil {
		panic(err)
	}
}
