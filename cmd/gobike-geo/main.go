package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"text/template"
)

type GeometryCollection struct {
	Type       string         `json:"type"`
	Geometries []MultiPolygon `json:"geometries"`
}

type MultiPolygon struct {
	Type        string          `json:"type"`
	Coordinates [][][][]float64 `json:"coordinates"`
}

const source = `package geo

var {{.UnexportedPoints}} = [][][][]float64{
{{- range .Points}}
	{
{{- range .}}
		{
{{- range .}}
			{{"{"}}{{index . 0}}, {{index . 1}}{{"}"}},
{{- end}}
		},
{{- end}}
	},
{{- end}}
}

var {{.ExportedCity}} = &City{
	points: {{.UnexportedPoints}},
}
`

type tmplData struct {
	ExportedCity     string
	UnexportedPoints string
	Points           [][][][]float64
}

func build(in, out, name string) error {
	tmpl := template.Must(template.New("city").Parse(source))

	blob, err := ioutil.ReadFile(in)
	if err != nil {
		return err
	}

	var gc GeometryCollection
	if err := json.Unmarshal(blob, &gc); err != nil {
		return err
	}

	g := gc.Geometries[0]

	w, err := os.Create(out)
	if err != nil {
		return err
	}
	defer w.Close()

	data := &tmplData{
		ExportedCity:     name,
		UnexportedPoints: fmt.Sprintf("%sPoints", strings.ToLower(name)),
		Points:           g.Coordinates,
	}
	return tmpl.ExecuteTemplate(w, "city", data)
}

func main() {
	flag.Parse()

	geojsonPath := flag.Arg(0)
	golangPath := flag.Arg(1)
	cityName := flag.Arg(2)

	if err := build(geojsonPath, golangPath, cityName); err != nil {
		log.Fatal(err)
	}
}
