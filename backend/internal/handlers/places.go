package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"

	"demomap/internal/db"
	"demomap/internal/middleware"
	"demomap/internal/models"
	"demomap/internal/utils"

	"github.com/gorilla/mux"
)

func GetPlacesHandler(w http.ResponseWriter, r *http.Request) {
	// Remove userID filter - show all places
	db := db.GetDB()
	query := `
        SELECT 
            p.id, p.user_id, p.name, p.category, p.description, p.address, p.phone, p.website,
            ST_X(p.geom) as longitude, ST_Y(p.geom) as latitude,
            COALESCE(p.average_rating, 0) as average_rating,
            COALESCE(p.total_ratings, 0) as total_ratings,
            p.created_at, p.updated_at,
            u.username as creator_username
        FROM places p
        LEFT JOIN users u ON p.user_id = u.id
        ORDER BY p.created_at DESC
    `

	rows, err := db.Query(query)
	if err != nil {
		utils.JSONError(w, http.StatusInternalServerError, "Failed to fetch places")
		return
	}
	defer rows.Close()

	places := []map[string]interface{}{}
	for rows.Next() {
		var p models.Place
		var username string
		err := rows.Scan(
			&p.ID, &p.UserID, &p.Name, &p.Category, &p.Description,
			&p.Address, &p.Phone, &p.Website,
			&p.Longitude, &p.Latitude,
			&p.AverageRating, &p.TotalRatings,
			&p.CreatedAt, &p.UpdatedAt,
			&username,
		)
		if err != nil {
			utils.JSONError(w, http.StatusInternalServerError, "Failed to scan place")
			return
		}

		// Add creator info to response
		placeData := map[string]interface{}{
			"id":             p.ID,
			"name":           p.Name,
			"category":       p.Category,
			"description":    p.Description,
			"address":        p.Address,
			"phone":          p.Phone,
			"website":        p.Website,
			"longitude":      p.Longitude,
			"latitude":       p.Latitude,
			"average_rating": p.AverageRating,
			"total_ratings":  p.TotalRatings,
			"created_at":     p.CreatedAt,
			"updated_at":     p.UpdatedAt,
			"created_by":     username,
			"is_owner":       p.UserID == middleware.GetUserID(r), // Add this to check ownership
		}
		places = append(places, placeData)
	}

	utils.JSONResponse(w, http.StatusOK, places)
}

func CreatePlaceHandler(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)

	var req models.CreatePlaceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.JSONError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Name == "" || req.Category == "" {
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

	var placeID int
	err := db.QueryRow(query,
		userID, req.Name, req.Category, req.Description,
		req.Address, req.Phone, req.Website,
		req.Longitude, req.Latitude,
	).Scan(&placeID)
	if err != nil {
		utils.JSONError(w, http.StatusInternalServerError, "Failed to create place")
		return
	}

	utils.JSONResponse(w, http.StatusCreated, map[string]interface{}{
		"message":  "Place created successfully",
		"place_id": placeID,
	})
}

func GetPlaceHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	placeID := vars["id"]
	currentUserID := middleware.GetUserID(r)

	db := db.GetDB()
	query := `
        SELECT 
            p.id, p.user_id, p.name, p.category, p.description, p.address, p.phone, p.website,
            ST_X(p.geom) as longitude, ST_Y(p.geom) as latitude,
            COALESCE(p.average_rating, 0) as average_rating,
            COALESCE(p.total_ratings, 0) as total_ratings,
            p.created_at, p.updated_at,
            u.username as creator_username
        FROM places p
        LEFT JOIN users u ON p.user_id = u.id
        WHERE p.id = $1
    `

	var p models.Place
	var username string
	err := db.QueryRow(query, placeID).Scan(
		&p.ID, &p.UserID, &p.Name, &p.Category, &p.Description,
		&p.Address, &p.Phone, &p.Website,
		&p.Longitude, &p.Latitude,
		&p.AverageRating, &p.TotalRatings,
		&p.CreatedAt, &p.UpdatedAt,
		&username,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			utils.JSONError(w, http.StatusNotFound, "Place not found")
			return
		}
		utils.JSONError(w, http.StatusInternalServerError, "Failed to fetch place")
		return
	}

	// Get photos
	//photos, _ := getPlacePhotos(placeID)

	// Get user's rating (if logged in)
	var userRating *models.Rating
	if currentUserID > 0 {
		//userRating, _ = getUserRating(placeID, currentUserID)
	}

	response := map[string]interface{}{
		"id":             p.ID,
		"user_id":        p.UserID,
		"name":           p.Name,
		"category":       p.Category,
		"description":    p.Description,
		"address":        p.Address,
		"phone":          p.Phone,
		"website":        p.Website,
		"longitude":      p.Longitude,
		"latitude":       p.Latitude,
		"average_rating": p.AverageRating,
		"total_ratings":  p.TotalRatings,
		"created_at":     p.CreatedAt,
		"updated_at":     p.UpdatedAt,
		"created_by":     username,
		"is_owner":       p.UserID == currentUserID,
		//"photos": photos,
		"user_rating": userRating,
	}

	utils.JSONResponse(w, http.StatusOK, response)
}

func UpdatePlaceHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	placeID := vars["id"]
	userID := middleware.GetUserID(r)

	var req models.CreatePlaceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.JSONError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	db := db.GetDB()
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
		req.Name, req.Category, req.Description,
		req.Address, req.Phone, req.Website,
		req.Longitude, req.Latitude,
		placeID, userID,
	)
	if err != nil {
		utils.JSONError(w, http.StatusInternalServerError, "Failed to update place")
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		utils.JSONError(w, http.StatusNotFound, "Place not found")
		return
	}

	utils.JSONResponse(w, http.StatusOK, map[string]string{
		"message": "Place updated successfully",
	})
}

func DeletePlaceHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	placeID := vars["id"]
	userID := middleware.GetUserID(r)

	db := db.GetDB()
	query := `DELETE FROM places WHERE id = $1 AND user_id = $2`
	result, err := db.Exec(query, placeID, userID)
	if err != nil {
		utils.JSONError(w, http.StatusInternalServerError, "Failed to delete place")
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		utils.JSONError(w, http.StatusNotFound, "Place not found")
		return
	}

	utils.JSONResponse(w, http.StatusOK, map[string]string{
		"message": "Place deleted successfully",
	})
}

func NearbyPlacesHandler(w http.ResponseWriter, r *http.Request) {
	// Remove userID filter - show all nearby places
	latStr := r.URL.Query().Get("lat")
	lonStr := r.URL.Query().Get("lon")
	radiusStr := r.URL.Query().Get("radius")

	if latStr == "" || lonStr == "" {
		utils.JSONError(w, http.StatusBadRequest, "Missing lat or lon parameters")
		return
	}

	lat, _ := strconv.ParseFloat(latStr, 64)
	lon, _ := strconv.ParseFloat(lonStr, 64)
	radius := 5.0
	if radiusStr != "" {
		radius, _ = strconv.ParseFloat(radiusStr, 64)
	}

	db := db.GetDB()
	query := `
        SELECT 
            p.id, p.user_id, p.name, p.category, p.description, p.address, p.phone, p.website,
            ST_X(p.geom) as longitude, ST_Y(p.geom) as latitude,
            COALESCE(p.average_rating, 0) as average_rating,
            COALESCE(p.total_ratings, 0) as total_ratings,
            u.username as creator_username,
            ST_Distance(p.geom, ST_SetSRID(ST_MakePoint($1, $2), 4326)) * 111.319 as distance_km
        FROM places p
        LEFT JOIN users u ON p.user_id = u.id
        WHERE ST_Distance(p.geom, ST_SetSRID(ST_MakePoint($1, $2), 4326)) < ($3 / 111.319)
        ORDER BY distance_km
    `

	rows, err := db.Query(query, lon, lat, radius)
	if err != nil {
		utils.JSONError(w, http.StatusInternalServerError, "Failed to find nearby places")
		return
	}
	defer rows.Close()

	type NearbyPlace struct {
		models.Place
		CreatorUsername string  `json:"created_by"`
		DistanceKm      float64 `json:"distance_km"`
	}

	places := []NearbyPlace{}
	for rows.Next() {
		var p NearbyPlace
		err := rows.Scan(
			&p.ID, &p.UserID, &p.Name, &p.Category, &p.Description,
			&p.Address, &p.Phone, &p.Website,
			&p.Longitude, &p.Latitude,
			&p.AverageRating, &p.TotalRatings,
			&p.CreatorUsername,
			&p.DistanceKm,
		)
		if err != nil {
			utils.JSONError(w, http.StatusInternalServerError, "Failed to scan place")
			return
		}
		places = append(places, p)
	}

	utils.JSONResponse(w, http.StatusOK, places)
}

func GetCategoriesHandler(w http.ResponseWriter, r *http.Request) {
	db := db.GetDB()
	query := `
        SELECT DISTINCT category 
        FROM places 
        ORDER BY category
    `

	rows, err := db.Query(query)
	if err != nil {
		utils.JSONError(w, http.StatusInternalServerError, "Failed to fetch categories")
		return
	}
	defer rows.Close()

	categories := []string{}
	for rows.Next() {
		var category string
		if err := rows.Scan(&category); err != nil {
			utils.JSONError(w, http.StatusInternalServerError, "Failed to scan category")
			return
		}
		categories = append(categories, category)
	}

	utils.JSONResponse(w, http.StatusOK, categories)
}
