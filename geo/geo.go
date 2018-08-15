package geo

type GeometryCollection struct {
	Type       string         `json:"type"`
	Geometries []MultiPolygon `json:"geometries"`
}

type MultiPolygon struct {
	Type        string          `json:"type"`
	Coordinates [][][][]float64 `json:"coordinates"`
}

type FeatureCollection struct {
	Type     string     `json:"type"`
	Features []*Feature `json:"features"`
}

type Feature struct {
	Type       string            `json:"type"`
	Properties map[string]string `json:"properties"`
	Geometry   *MultiPolygon     `json:"geometry"`
}
