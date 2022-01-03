package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/go-redis/redis/v8"
	geojson "github.com/paulmach/go.geojson"
	"github.com/pkg/errors"
)

const _ID = "ID"

func createIndex(ctx context.Context, client *redis.Client, indexName string) (bool, error) {
	ret, err := client.Do(ctx, "S2GEO.ISET", indexName).Result()

	if err != nil {
		if err.Error() == "index already exists" {
			return false, nil
		}
		return false, errors.Wrap(err, "index creation failed")
	}

	if ret != "OK" {
		return false, errors.New(fmt.Sprint("unexpected response returned: ", ret))
	}

	return true, nil
}

func populateIndex(ctx context.Context, client *redis.Client, indexName string, fc *geojson.FeatureCollection) error {
	const STEP = 10
	increment := len(fc.Features) / STEP
	for idx, feature := range fc.Features {
		if (idx+1)%increment == 0 {
			log.Printf("Populating progress: %d%%\n", (100 * (idx + 1) / len(fc.Features)))
		}

		if feature.Geometry.Type != geojson.GeometryPolygon {
			continue
		}
		id, _ := feature.PropertyString(_ID)
		body, _ := json.Marshal(feature.Geometry.Polygon)
		_, err := client.Do(ctx, "S2GEO.POLYSET", indexName, id, string(body)).Result()
		if err != nil {
			log.Printf("error while storing polygon %s (%s): %v\n", id, body, err)
		}
	}
	log.Printf("Populating finished!")
	return nil
}

func parseFeatureCollection(path string) (*geojson.FeatureCollection, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	data, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}

	fc, err := geojson.UnmarshalFeatureCollection(data)
	if err != nil {
		return nil, err
	}

	return fc, nil
}

func assignIdentifiers(fc *geojson.FeatureCollection, prefix string) {
	for idx, feature := range fc.Features {
		feature.SetProperty(_ID, fmt.Sprintf("%s_%d", prefix, idx))
	}
}
