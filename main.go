package main

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/sulewicz/redis-geo-sample/server"
)

const INDEX_NAME = "alaska"

func filename(path string) string {
	name := filepath.Base((path))
	return strings.TrimSuffix(name, filepath.Ext(name))
}

func main() {
	assetPath := "assets/alaska.geojson"
	if len(os.Args) == 2 {
		assetPath = os.Args[1]
	}
	indexName := filename(assetPath)

	log.Println("Asset path: ", assetPath)
	log.Println("Index name: ", indexName)
	server := server.New()
	if err := server.Bootstrap("localhost:6379", indexName, assetPath); err != nil {
		log.Println("Setup failed. Make sure your local Redis instance is running.")
		panic(err)
	}

	if err := server.Run(); err != nil {
		panic(err)
	}
}
