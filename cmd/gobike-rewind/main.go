package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
)

type GeometryCollection struct {
	Type       string         `json:"type"`
	Geometries []MultiPolygon `json:"geometries"`
}

type MultiPolygon struct {
	Type        string          `json:"type"`
	Coordinates [][][][]float64 `json:"coordinates"`
}

func rewind(in, out string) error {
	blob, err := ioutil.ReadFile(in)
	if err != nil {
		return err
	}

	var gc GeometryCollection
	if err := json.Unmarshal(blob, &gc); err != nil {
		return err
	}

	g := gc.Geometries[0].Coordinates

	// The GeoJSON we get from OpenStreetMaps is wound in the incorrect
	// direction. Rewind all the polygons to ensure they work corectly with golang/geo
	for k, _ := range g {
		for j, _ := range g[k] {
			for i := len(g[k][j])/2 - 1; i >= 0; i-- {
				opp := len(g[k][j]) - 1 - i
				g[k][j][i], g[k][j][opp] = g[k][j][opp], g[k][j][i]
			}
		}
	}

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
