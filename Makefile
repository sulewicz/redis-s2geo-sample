BINARY_NAME=redis-s2geo-sample

all: build test

build:
	go build -o ${BINARY_NAME} main.go

run: build
	./${BINARY_NAME} assets/alaska.geojson

clean:
	go clean
	rm ${BINARY_NAME}
