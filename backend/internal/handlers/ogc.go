package handlers

import (
	"database/sql"
	"demomap/internal/middleware"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"demomap/internal/db"
	"demomap/internal/models"
	"demomap/internal/utils"

	"github.com/gorilla/mux"
)

// Landing Page Handler
func OGCLandingHandler(w http.ResponseWriter, r *http.Request) {
	baseURL := getBaseURL(r)

	response := models.LandingPage{
		Title:       "Map Explorer API",
		Description: "OGC API - Features compliant API for Map Explorer",
		Links: []models.Link{
			{
				Href:  baseURL + "/",
				Rel:   "self",
				Type:  "application/json",
				Title: "This document",
			},
			{
				Href:  baseURL + "/conformance",
				Rel:   "conformance",
				Type:  "application/json",
				Title: "OGC API conformance classes implemented by this server",
			},
			{
				Href:  baseURL + "/collections",
				Rel:   "data",
				Type:  "application/json",
				Title: "Metadata about the feature collections",
			},
			{
				Href:  baseURL + "/api",
				Rel:   "service-desc",
				Type:  "application/vnd.oai.openapi+json;version=3.0",
				Title: "OpenAPI definition",
			},
		},
	}

	utils.JSONResponse(w, http.StatusOK, response)
}

// Conformance Handler
func OGCConformanceHandler(w http.ResponseWriter, r *http.Request) {
	response := models.Conformance{
		ConformsTo: []string{
			"http://www.opengis.net/spec/ogcapi-features-1/1.0/conf/core",
			"http://www.opengis.net/spec/ogcapi-features-1/1.0/conf/geojson",
			"http://www.opengis.net/spec/ogcapi-features-1/1.0/conf/html",
		},
	}

	utils.JSONResponse(w, http.StatusOK, response)
}

// Collections Handler
func OGCCollectionsHandler(w http.ResponseWriter, r *http.Request) {
	db := db.GetDB()
	baseURL := getBaseURL(r)

	// Get all collections (in this case, just "places")
	collections := []models.Collection{
		{
			ID:          "places",
			Title:       "Places",
			Description: "Collection of places with geospatial locations",
			Extent: models.Extent{
				Spatial: models.SpatialExtent{
					BBox: [][]float64{{-180, -90, 180, 90}},
					CRS:  "http://www.opengis.net/def/crs/OGC/1.3/CRS84",
				},
			},
			Links: []models.Link{
				{
					Href:  baseURL + "/collections/places",
					Rel:   "self",
					Type:  "application/json",
					Title: "This collection",
				},
				{
					Href:  baseURL + "/collections/places/items",
					Rel:   "items",
					Type:  "application/geo+json",
					Title: "Places features",
				},
			},
		},
	}

	// Get user's places count
	var count int
	db.QueryRow("SELECT COUNT(*) FROM places").Scan(&count)

	response := models.Collections{
		Collections: collections,
	}

	// Add links
	response.Collections[0].Links = append(response.Collections[0].Links, models.Link{
		Href:  baseURL + "/collections/places/items?limit=10",
		Rel:   "items",
		Type:  "application/geo+json",
		Title: "Places features (first 10)",
	})

	utils.JSONResponse(w, http.StatusOK, response)
}

// Collection Items Handler (Get Features)
func OGCItemsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	collectionID := vars["collectionId"]

	if collectionID != "places" {
		utils.JSONError(w, http.StatusNotFound, "Collection not found")
		return
	}

	userID := middleware.GetUserID(r)
	db := db.GetDB()

	// Parse query parameters for filtering
	limit := 10
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	offset := 0
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	// BBOX filter
	bbox := r.URL.Query().Get("bbox")
	var whereClause string
	var args []interface{}
	argIndex := 1

	if bbox != "" {
		coords := strings.Split(bbox, ",")
		if len(coords) == 4 {
			minLon, _ := strconv.ParseFloat(coords[0], 64)
			minLat, _ := strconv.ParseFloat(coords[1], 64)
			maxLon, _ := strconv.ParseFloat(coords[2], 64)
			maxLat, _ := strconv.ParseFloat(coords[3], 64)

			whereClause = fmt.Sprintf(`WHERE user_id = $%d AND ST_Intersects(
				geom, 
				ST_MakeEnvelope($%d, $%d, $%d, $%d, 4326)
			)`, argIndex, argIndex+1, argIndex+2, argIndex+3, argIndex+4)

			args = append(args, userID, minLon, minLat, maxLon, maxLat)
			argIndex += 5
		}
	} else {
		whereClause = fmt.Sprintf("WHERE user_id = $%d", argIndex)
		args = append(args, userID)
		argIndex++
	}

	// Count total
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM places %s", whereClause)
	var total int
	err := db.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		utils.JSONError(w, http.StatusInternalServerError, "Failed to count features")
		return
	}

	// Query features
	query := fmt.Sprintf(`
		SELECT 
			id, user_id, name, category, description, address, phone, website,
			ST_X(geom) as longitude, ST_Y(geom) as latitude,
			COALESCE(average_rating, 0) as average_rating,
			COALESCE(total_ratings, 0) as total_ratings,
			created_at, updated_at,
			ST_AsGeoJSON(geom) as geojson
		FROM places
		%s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIndex, argIndex+1)

	args = append(args, limit, offset)

	rows, err := db.Query(query, args...)
	if err != nil {
		utils.JSONError(w, http.StatusInternalServerError, "Failed to fetch features")
		return
	}
	defer rows.Close()

	baseURL := getBaseURL(r)
	features := []models.GeoJSONFeature{}

	for rows.Next() {
		var (
			id, userID                                           int
			name, category, description, address, phone, website string
			longitude, latitude, avgRating                       float64
			totalRatings                                         int
			createdAt, updatedAt                                 string
			geojson                                              string
		)

		err := rows.Scan(
			&id, &userID, &name, &category, &description,
			&address, &phone, &website,
			&longitude, &latitude,
			&avgRating, &totalRatings,
			&createdAt, &updatedAt,
			&geojson,
		)
		if err != nil {
			continue
		}

		// Parse GeoJSON geometry
		var geom models.GeoJSONGeometry
		json.Unmarshal([]byte(geojson), &geom)

		feature := models.GeoJSONFeature{
			Type:     "Feature",
			ID:       strconv.Itoa(id),
			Geometry: geom,
			Properties: map[string]interface{}{
				"id":             id,
				"user_id":        userID,
				"name":           name,
				"category":       category,
				"description":    description,
				"address":        address,
				"phone":          phone,
				"website":        website,
				"longitude":      longitude,
				"latitude":       latitude,
				"average_rating": avgRating,
				"total_ratings":  totalRatings,
				"created_at":     createdAt,
				"updated_at":     updatedAt,
			},
		}
		features = append(features, feature)
	}

	response := models.FeatureCollectionResponse{
		Type:           "FeatureCollection",
		NumberMatched:  total,
		NumberReturned: len(features),
		Features:       features,
		Links: []models.Link{
			{
				Href:  baseURL + "/collections/places/items",
				Rel:   "self",
				Type:  "application/geo+json",
				Title: "This collection",
			},
		},
	}

	// Add pagination links
	if offset > 0 {
		response.Links = append(response.Links, models.Link{
			Href:  fmt.Sprintf("%s/collections/places/items?limit=%d&offset=%d", baseURL, limit, offset-limit),
			Rel:   "prev",
			Type:  "application/geo+json",
			Title: "Previous page",
		})
	}

	if offset+limit < total {
		response.Links = append(response.Links, models.Link{
			Href:  fmt.Sprintf("%s/collections/places/items?limit=%d&offset=%d", baseURL, limit, offset+limit),
			Rel:   "next",
			Type:  "application/geo+json",
			Title: "Next page",
		})
	}

	utils.JSONResponse(w, http.StatusOK, response)
}

// Single Feature Handler
func OGCFeatureHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	collectionID := vars["collectionId"]
	featureID := vars["featureId"]

	if collectionID != "places" {
		utils.JSONError(w, http.StatusNotFound, "Collection not found")
		return
	}

	userID := middleware.GetUserID(r)
	db := db.GetDB()

	query := `
		SELECT 
			id, user_id, name, category, description, address, phone, website,
			ST_X(geom) as longitude, ST_Y(geom) as latitude,
			COALESCE(average_rating, 0) as average_rating,
			COALESCE(total_ratings, 0) as total_ratings,
			created_at, updated_at,
			ST_AsGeoJSON(geom) as geojson
		FROM places
		WHERE id = $1 AND user_id = $2
	`

	var (
		id, uid                                              int
		name, category, description, address, phone, website string
		longitude, latitude, avgRating                       float64
		totalRatings                                         int
		createdAt, updatedAt                                 string
		geojson                                              string
	)

	err := db.QueryRow(query, featureID, userID).Scan(
		&id, &uid, &name, &category, &description,
		&address, &phone, &website,
		&longitude, &latitude,
		&avgRating, &totalRatings,
		&createdAt, &updatedAt,
		&geojson,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			utils.JSONError(w, http.StatusNotFound, "Feature not found")
			return
		}
		utils.JSONError(w, http.StatusInternalServerError, "Failed to fetch feature")
		return
	}

	var geom models.GeoJSONGeometry
	json.Unmarshal([]byte(geojson), &geom)

	baseURL := getBaseURL(r)
	response := models.FeatureResponse{
		Type:     "Feature",
		ID:       featureID,
		Geometry: geom,
		Properties: map[string]interface{}{
			"id":             id,
			"user_id":        uid,
			"name":           name,
			"category":       category,
			"description":    description,
			"address":        address,
			"phone":          phone,
			"website":        website,
			"longitude":      longitude,
			"latitude":       latitude,
			"average_rating": avgRating,
			"total_ratings":  totalRatings,
			"created_at":     createdAt,
			"updated_at":     updatedAt,
		},
		Links: []models.Link{
			{
				Href:  baseURL + "/collections/places/items/" + featureID,
				Rel:   "self",
				Type:  "application/geo+json",
				Title: "This feature",
			},
			{
				Href:  baseURL + "/collections/places/items",
				Rel:   "collection",
				Type:  "application/geo+json",
				Title: "Places collection",
			},
		},
	}

	utils.JSONResponse(w, http.StatusOK, response)
}

// Helper function to get base URL
func getBaseURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s", scheme, r.Host)
}
