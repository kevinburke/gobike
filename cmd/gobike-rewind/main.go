package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"

	"github.com/kevinburke/gobike/geo"
)

func rewind(in, out string) error {
	blob, err := ioutil.ReadFile(in)
	if err != nil {
		return err
	}

	var gc geo.GeometryCollection
	if err := json.Unmarshal(blob, &gc); err != nil {
		return err
	}

	g := gc.Geometries[0].Coordinates

	// The GeoJSON we get from OpenStreetMaps is wound in the incorrect
	// direction. Rewind all the polygons to ensure they work corectly with golang/geo
	geo.Rewind(g)

	output, err := json.Marshal(gc)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(out, output, 0644)
}

func main() {
	flag.Parse()

	geojsonPath := flag.Arg(0)
	golangPath := flag.Arg(1)

	if err := rewind(geojsonPath, golangPath); err != nil {
		log.Fatal(err)
	}
}
