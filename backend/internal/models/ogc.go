package models

// OGC Landing Page
type LandingPage struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Links       []Link `json:"links"`
}

// OGC Link
type Link struct {
	Href  string `json:"href"`
	Rel   string `json:"rel"`
	Type  string `json:"type,omitempty"`
	Title string `json:"title,omitempty"`
}

// OGC Conformance
type Conformance struct {
	ConformsTo []string `json:"conformsTo"`
}

// OGC Collections
type Collections struct {
	Collections []Collection `json:"collections"`
}

// OGC Collection
type Collection struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Extent      Extent `json:"extent"`
	Links       []Link `json:"links"`
}

// OGC Extent
type Extent struct {
	Spatial SpatialExtent `json:"spatial"`
}

type SpatialExtent struct {
	BBox [][]float64 `json:"bbox"`
	CRS  string      `json:"crs"`
}

// OGC Feature Collection Response
type FeatureCollectionResponse struct {
	Type           string           `json:"type"`
	NumberMatched  int              `json:"numberMatched,omitempty"`
	NumberReturned int              `json:"numberReturned,omitempty"`
	Features       []GeoJSONFeature `json:"features"`
	Links          []Link           `json:"links"`
}

// OGC Feature Response
type FeatureResponse struct {
	Type       string                 `json:"type"`
	ID         string                 `json:"id"`
	Geometry   GeoJSONGeometry        `json:"geometry"`
	Properties map[string]interface{} `json:"properties"`
	Links      []Link                 `json:"links"`
}

type GeoJSONGeometry struct {
	Type        string      `json:"type"`
	Coordinates interface{} `json:"coordinates"` // Can be []float64, [][]float64, etc.
}

type GeoJSONFeature struct {
	Type       string                 `json:"type"`
	ID         string                 `json:"id"`
	Geometry   GeoJSONGeometry        `json:"geometry"`
	Properties map[string]interface{} `json:"properties"`
}
