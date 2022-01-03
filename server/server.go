package server

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

type Server interface {
	Bootstrap(redisAddress string, indexName string, geojsonPath string) error
	Run() error
}

type server struct {
	redisClient *redis.Client
	indexName   string
}

type polygonResponse struct {
	ID   string           `json:"id"`
	Body *json.RawMessage `json:"body"`
}

type fetchMultiplePolygonsRequest struct {
	IDs []string `json:"ids"`
}

type polygonSearchRequest struct {
	Polygon [][][]float32 `json:"polygon"`
}

type pointSearchRequest struct {
	Point []float32 `json:"point"`
}

func New() Server {
	return &server{}
}

func (s *server) Bootstrap(redisAddress string, indexName string, geojsonPath string) error {
	s.indexName = indexName
	log.Println("Testing Redis connection...")
	ctx := context.Background()
	s.redisClient = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})

	_, err := s.redisClient.Ping(ctx).Result()
	if err != nil {
		return err
	}

	result, err := createIndex(ctx, s.redisClient, indexName)
	if err != nil {
		return err
	}
	if result {
		// New index created, populating
		log.Println("Parsing geometries...")
		fc, err := parseFeatureCollection(geojsonPath)
		if err != nil {
			return err
		}
		assignIdentifiers(fc, indexName)
		log.Println("Populating index...")
		err = populateIndex(ctx, s.redisClient, indexName, fc)
		if err != nil {
			return err
		}
	} else {
		log.Println("Index already exists...")
	}
	return nil
}

func (s *server) listPolygons(c *gin.Context) {
	ids, err := s.redisClient.Do(c.Request.Context(), "S2GEO.POLYLIST", s.indexName).StringSlice()
	if err != nil {
		if err == redis.Nil {
			c.JSON(http.StatusOK, gin.H{
				"ids": []string{},
			})
			return
		}
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err,
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"ids": ids,
	})
}

func (s *server) fetchPolygon(c *gin.Context) {
	id := c.Param("id")
	body, err := s.redisClient.Do(c.Request.Context(), "S2GEO.POLYGET", s.indexName, id).Result()
	if err != nil {
		if err == redis.Nil {
			c.JSON(http.StatusNotFound, gin.H{})
			return
		}
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err,
		})
		return
	}
	bodyStr, ok := body.(string)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{})
		return
	}
	bodyBytes := json.RawMessage(bodyStr)
	c.JSON(http.StatusOK, polygonResponse{
		ID:   id,
		Body: &bodyBytes,
	})
}

func (s *server) polygonSearch(c *gin.Context) {
	var requestBody polygonSearchRequest
	if err := c.ShouldBindJSON(&requestBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	polygonBody, err := json.Marshal(requestBody.Polygon)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ids, err := s.redisClient.Do(c.Request.Context(), "S2GEO.POLYSEARCH", s.indexName, polygonBody).StringSlice()
	if err != nil {
		if err == redis.Nil {
			c.JSON(http.StatusOK, gin.H{
				"ids": []string{},
			})
			return
		}
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err,
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"ids": ids,
	})
}

func (s *server) pointSearch(c *gin.Context) {
	var requestBody pointSearchRequest
	if err := c.ShouldBindJSON(&requestBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	pointBody, err := json.Marshal(requestBody.Point)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ids, err := s.redisClient.Do(c.Request.Context(), "S2GEO.POINTSEARCH", s.indexName, pointBody).StringSlice()
	if err != nil {
		if err == redis.Nil {
			c.JSON(http.StatusOK, gin.H{
				"ids": []string{},
			})
			return
		}
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err,
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"ids": ids,
	})
}

func (s *server) fetchMultiplePolygons(c *gin.Context) {
	var requestBody fetchMultiplePolygonsRequest
	if err := c.ShouldBindJSON(&requestBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	args := []interface{}{"S2GEO.POLYMGET", s.indexName}
	for _, id := range requestBody.IDs {
		args = append(args, id)
	}
	bodies, err := s.redisClient.Do(c.Request.Context(), args...).Slice()
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err,
		})
		return
	}
	polygons := []polygonResponse{}
	for idx, body := range bodies {
		bodyStr, ok := body.(string)
		if ok {
			bodyBytes := json.RawMessage(bodyStr)
			polygons = append(polygons, polygonResponse{
				ID:   requestBody.IDs[idx],
				Body: &bodyBytes,
			})
		}

	}
	c.JSON(http.StatusOK, gin.H{
		"polygons": polygons,
	})
}

func (s *server) Run() error {
	r := gin.Default()
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})
	r.POST("/api/polygons", s.fetchMultiplePolygons)
	r.GET("/api/polygons", s.listPolygons)
	r.GET("/api/polygons/:id", s.fetchPolygon)
	r.POST("/api/search/polygons/by_polygon", s.polygonSearch)
	r.POST("/api/search/polygons/by_point", s.pointSearch)
	r.Static("/app", "./frontend")
	r.Run()
	return nil
}
