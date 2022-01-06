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
	for _, feature := range fc.Features {
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
		if _, ok := feature.Properties[_ID]; !ok {
			feature.SetProperty(_ID, fmt.Sprintf("%s_%d", prefix, idx))
		}
	}
}
