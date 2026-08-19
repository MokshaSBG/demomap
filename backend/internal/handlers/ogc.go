package handlers

import (
	"database/sql"
	"demomap/internal/fedratlas"
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

// Global Fedratlas client (initialized in main)
var FedratlasClient *fedratlas.Client

// SetFedratlasClient sets the global Fedratlas client
func SetFedratlasClient(client *fedratlas.Client) {
	FedratlasClient = client
}

// ============================================
// ✅ KEEP: Landing Page Handler (UNCHANGED)
// ============================================
func OGCLandingHandler(w http.ResponseWriter, r *http.Request) {
	baseURL := getBaseURL(r)

	response := models.LandingPage{
		Title:       "Map Explorer API",
		Description: "OGC API - Features compliant API for Map Explorer",
		Links: []models.Link{
			{Href: baseURL + "/", Rel: "self", Type: "application/json", Title: "This document"},
			{Href: baseURL + "/conformance", Rel: "conformance", Type: "application/json", Title: "OGC API conformance classes"},
			{Href: baseURL + "/collections", Rel: "data", Type: "application/json", Title: "Metadata about the feature collections"},
			{Href: baseURL + "/api", Rel: "service-desc", Type: "application/vnd.oai.openapi+json;version=3.0", Title: "OpenAPI definition"},
		},
	}

	utils.JSONResponse(w, http.StatusOK, response)
}

// ============================================
// ✅ KEEP: Conformance Handler (UNCHANGED)
// ============================================
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

// ============================================
// ✅ KEEP: Collections Handler (UNCHANGED)
// ============================================
func OGCCollectionsHandler(w http.ResponseWriter, r *http.Request) {
	db := db.GetDB()
	baseURL := getBaseURL(r)

	var count int
	db.QueryRow("SELECT COUNT(*) FROM places").Scan(&count)

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
				{Href: baseURL + "/collections/places", Rel: "self", Type: "application/json", Title: "This collection"},
				{Href: baseURL + "/collections/places/items", Rel: "items", Type: "application/geo+json", Title: "Places features"},
				{Href: baseURL + "/collections/places/items?limit=10", Rel: "items", Type: "application/geo+json", Title: "Places features (first 10)"},
			},
		},
	}

	response := models.Collections{Collections: collections}
	utils.JSONResponse(w, http.StatusOK, response)
}

// ============================================
// ✅ KEEP: GET Items Handler (UNCHANGED)
// ============================================
func OGCItemsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	collectionID := vars["collectionId"]

	if collectionID != "places" {
		utils.JSONError(w, http.StatusNotFound, "Collection not found")
		return
	}

	userID := middleware.GetUserID(r)
	db := db.GetDB()

	// Parse query parameters
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
            p.id, p.user_id, p.name, p.category, p.description, p.address, p.phone, p.website,
            ST_X(p.geom) as longitude, ST_Y(p.geom) as latitude,
            COALESCE(p.average_rating, 0) as average_rating,
            COALESCE(p.total_ratings, 0) as total_ratings,
            p.created_at, p.updated_at,
            ST_AsGeoJSON(p.geom) as geojson,
            u.username as creator_username
        FROM places p
        LEFT JOIN users u ON p.user_id = u.id
        %s
        ORDER BY p.created_at DESC
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
			creatorUsername                                      string
		)

		err := rows.Scan(
			&id, &userID, &name, &category, &description,
			&address, &phone, &website,
			&longitude, &latitude,
			&avgRating, &totalRatings,
			&createdAt, &updatedAt,
			&geojson,
			&creatorUsername,
		)
		if err != nil {
			continue
		}

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
				"created_by":     creatorUsername,
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
			{Href: baseURL + "/collections/places/items", Rel: "self", Type: "application/geo+json", Title: "This collection"},
		},
	}

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

// ============================================
// ✅ KEEP: GET Single Feature Handler (UNCHANGED)
// ============================================
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
			{Href: baseURL + "/collections/places/items/" + featureID, Rel: "self", Type: "application/geo+json", Title: "This feature"},
			{Href: baseURL + "/collections/places/items", Rel: "collection", Type: "application/geo+json", Title: "Places collection"},
		},
	}

	utils.JSONResponse(w, http.StatusOK, response)
}

// ============================================
// ✅ NEW: POST - Create Feature with Fedratlas Sync
// ============================================
func OGCCreateFeatureHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	collectionID := vars["collectionId"]

	if collectionID != "places" {
		utils.JSONError(w, http.StatusNotFound, "Collection not found")
		return
	}

	userID := middleware.GetUserID(r)

	var feature models.GeoJSONFeature
	if err := json.NewDecoder(r.Body).Decode(&feature); err != nil {
		utils.JSONError(w, http.StatusBadRequest, "Invalid GeoJSON")
		return
	}

	// Extract properties
	props := feature.Properties
	name, _ := props["name"].(string)
	category, _ := props["category"].(string)
	description, _ := props["description"].(string)
	address, _ := props["address"].(string)
	phone, _ := props["phone"].(string)
	website, _ := props["website"].(string)

	// ✅ FIXED: Extract coordinates from struct field
	var lon, lat float64
	if coords, ok := feature.Geometry.Coordinates.([]interface{}); ok && len(coords) >= 2 {
		lon, _ = coords[0].(float64)
		lat, _ = coords[1].(float64)
	}

	if name == "" || category == "" {
		utils.JSONError(w, http.StatusBadRequest, "Name and category are required")
		return
	}

	db := db.GetDB()

	query := `
        INSERT INTO places 
            (user_id, name, category, description, address, phone, website, geom)
        VALUES ($1, $2, $3, $4, $5, $6, $7, ST_SetSRID(ST_MakePoint($8, $9), 4326))
        RETURNING id
    `

	var placeID int64
	err := db.QueryRow(query,
		userID, name, category, description, address, phone, website,
		lon, lat,
	).Scan(&placeID)

	if err != nil {
		utils.JSONError(w, http.StatusInternalServerError, "Failed to create place")
		return
	}

	// ✅ Sync to Fedratlas
	if FedratlasClient != nil {
		featureData, _ := json.Marshal(feature)
		go FedratlasClient.OnFeatureChange(
			placeID,
			"Create",
			featureData,
			1,
			collectionID,
		)
	}

	feature.ID = strconv.FormatInt(placeID, 10)
	w.Header().Set("Location", r.URL.Path+"/"+feature.ID)

	utils.JSONResponse(w, http.StatusCreated, map[string]interface{}{
		"id":      placeID,
		"message": "Feature created successfully",
		"synced":  FedratlasClient != nil,
	})
}

// ============================================
// ✅ NEW: PUT - Update Feature with Fedratlas Sync
// ============================================
func OGCUpdateFeatureHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	collectionID := vars["collectionId"]
	featureID := vars["featureId"]

	if collectionID != "places" {
		utils.JSONError(w, http.StatusNotFound, "Collection not found")
		return
	}

	userID := middleware.GetUserID(r)

	var feature models.GeoJSONFeature
	if err := json.NewDecoder(r.Body).Decode(&feature); err != nil {
		utils.JSONError(w, http.StatusBadRequest, "Invalid GeoJSON")
		return
	}

	props := feature.Properties
	name, _ := props["name"].(string)
	category, _ := props["category"].(string)
	description, _ := props["description"].(string)
	address, _ := props["address"].(string)
	phone, _ := props["phone"].(string)
	website, _ := props["website"].(string)

	// ✅ FIXED: Extract coordinates using the correct type
	var lon, lat float64
	if coords, ok := feature.Geometry.Coordinates.([]interface{}); ok && len(coords) >= 2 {
		lon, _ = coords[0].(float64)
		lat, _ = coords[1].(float64)
	}

	db := db.GetDB()
	id, _ := strconv.Atoi(featureID)

	// Get current version
	var currentVersion int
	db.QueryRow("SELECT COALESCE(version, 1) FROM places WHERE id = $1", id).Scan(&currentVersion)

	query := `
        UPDATE places
        SET 
            name = $1, category = $2, description = $3,
            address = $4, phone = $5, website = $6,
            geom = ST_SetSRID(ST_MakePoint($7, $8), 4326),
            updated_at = CURRENT_TIMESTAMP
        WHERE id = $9 AND user_id = $10
    `

	result, err := db.Exec(query,
		name, category, description, address, phone, website,
		lon, lat,
		id, userID,
	)

	if err != nil {
		utils.JSONError(w, http.StatusInternalServerError, "Failed to update feature")
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		utils.JSONError(w, http.StatusNotFound, "Feature not found")
		return
	}

	// Sync to Fedratlas
	if FedratlasClient != nil {
		featureData, _ := json.Marshal(feature)
		go FedratlasClient.OnFeatureChange(
			int64(id),
			"Update",
			featureData,
			int32(currentVersion+1),
			collectionID,
		)
	}

	utils.JSONResponse(w, http.StatusOK, map[string]interface{}{
		"id":      id,
		"message": "Feature updated successfully",
	})
}

// ============================================
// ✅ NEW: DELETE - Delete Feature with Fedratlas Sync
// ============================================
func OGCDeleteFeatureHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	collectionID := vars["collectionId"]
	featureID := vars["featureId"]

	if collectionID != "places" {
		utils.JSONError(w, http.StatusNotFound, "Collection not found")
		return
	}

	userID := middleware.GetUserID(r)
	id, _ := strconv.Atoi(featureID)

	db := db.GetDB()
	query := `DELETE FROM places WHERE id = $1 AND user_id = $2`
	result, err := db.Exec(query, id, userID)

	if err != nil {
		utils.JSONError(w, http.StatusInternalServerError, "Failed to delete feature")
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		utils.JSONError(w, http.StatusNotFound, "Feature not found")
		return
	}

	// ✅ Sync to Fedratlas
	if FedratlasClient != nil {
		go FedratlasClient.OnFeatureChange(
			int64(id),
			"Delete",
			[]byte{},
			0,
			collectionID,
		)
	}

	utils.JSONResponse(w, http.StatusNoContent, nil)
}

// ============================================
// ✅ KEEP: Helper function (UNCHANGED)
// ============================================
func getBaseURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s", scheme, r.Host)
}
