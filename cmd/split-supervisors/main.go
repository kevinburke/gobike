package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"os"

	"github.com/kevinburke/gobike/geo"
)

func main() {
	flag.Parse()
	data, err := ioutil.ReadFile("geojson/supervisors.geojson")
	if err != nil {
		log.Fatal(err)
	}
	fc := new(geo.FeatureCollection)
	if err := json.Unmarshal(data, fc); err != nil {
		log.Fatal(err)
	}
	for i := 0; i < len(fc.Features); i++ {
		district := fc.Features[i].Properties["supervisor"]
		f, err := os.Create("geojson/sf-districts/" + district + ".geojson")
		if err != nil {
			log.Fatal(err)
		}
		multipoly := *fc.Features[i].Geometry
		geo.Rewind(multipoly.Coordinates)
		out := &geo.GeometryCollection{
			Type:       "GeometryCollection",
			Geometries: []geo.MultiPolygon{multipoly},
		}
		if err := json.NewEncoder(f).Encode(out); err != nil {
			log.Fatal(err)
		}
	}
}
