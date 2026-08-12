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
	userID := middleware.GetUserID(r)

	db := db.GetDB()
	query := `
        SELECT 
            id, user_id, name, category, description, address, phone, website,
            ST_X(geom) as longitude, ST_Y(geom) as latitude,
            COALESCE(average_rating, 0) as average_rating,
            COALESCE(total_ratings, 0) as total_ratings,
            created_at, updated_at
        FROM places
        WHERE user_id = $1
        ORDER BY created_at DESC
    `

	rows, err := db.Query(query, userID)
	if err != nil {
		utils.JSONError(w, http.StatusInternalServerError, "Failed to fetch places")
		return
	}
	defer rows.Close()

	places := []models.Place{}
	for rows.Next() {
		var p models.Place
		err := rows.Scan(
			&p.ID, &p.UserID, &p.Name, &p.Category, &p.Description,
			&p.Address, &p.Phone, &p.Website,
			&p.Longitude, &p.Latitude,
			&p.AverageRating, &p.TotalRatings,
			&p.CreatedAt, &p.UpdatedAt,
		)
		if err != nil {
			utils.JSONError(w, http.StatusInternalServerError, "Failed to scan place")
			return
		}
		places = append(places, p)
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
	userID := middleware.GetUserID(r)

	db := db.GetDB()
	query := `
        SELECT 
            id, user_id, name, category, description, address, phone, website,
            ST_X(geom) as longitude, ST_Y(geom) as latitude,
            COALESCE(average_rating, 0) as average_rating,
            COALESCE(total_ratings, 0) as total_ratings,
            created_at, updated_at
        FROM places
        WHERE id = $1 AND user_id = $2
    `

	var p models.Place
	err := db.QueryRow(query, placeID, userID).Scan(
		&p.ID, &p.UserID, &p.Name, &p.Category, &p.Description,
		&p.Address, &p.Phone, &p.Website,
		&p.Longitude, &p.Latitude,
		&p.AverageRating, &p.TotalRatings,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			utils.JSONError(w, http.StatusNotFound, "Place not found")
			return
		}
		utils.JSONError(w, http.StatusInternalServerError, "Failed to fetch place")
		return
	}

	utils.JSONResponse(w, http.StatusOK, p)
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
	userID := middleware.GetUserID(r)
	latStr := r.URL.Query().Get("lat")
	lonStr := r.URL.Query().Get("lon")
	radiusStr := r.URL.Query().Get("radius")

	if latStr == "" || lonStr == "" {
		utils.JSONError(w, http.StatusBadRequest, "Missing lat or lon parameters")
		return
	}

	lat, err := strconv.ParseFloat(latStr, 64)
	if err != nil {
		utils.JSONError(w, http.StatusBadRequest, "Invalid lat parameter")
		return
	}

	lon, err := strconv.ParseFloat(lonStr, 64)
	if err != nil {
		utils.JSONError(w, http.StatusBadRequest, "Invalid lon parameter")
		return
	}

	radius := 5.0
	if radiusStr != "" {
		radius, err = strconv.ParseFloat(radiusStr, 64)
		if err != nil {
			utils.JSONError(w, http.StatusBadRequest, "Invalid radius parameter")
			return
		}
	}

	db := db.GetDB()
	query := `
        SELECT 
            id, user_id, name, category, description, address, phone, website,
            ST_X(geom) as longitude, ST_Y(geom) as latitude,
            COALESCE(average_rating, 0) as average_rating,
            COALESCE(total_ratings, 0) as total_ratings,
            ST_Distance(geom, ST_SetSRID(ST_MakePoint($1, $2), 4326)) * 111.319 as distance_km
        FROM places
        WHERE user_id = $3
        AND ST_Distance(geom, ST_SetSRID(ST_MakePoint($1, $2), 4326)) < ($4 / 111.319)
        ORDER BY distance_km
    `

	rows, err := db.Query(query, lon, lat, userID, radius)
	if err != nil {
		utils.JSONError(w, http.StatusInternalServerError, "Failed to find nearby places")
		return
	}
	defer rows.Close()

	type NearbyPlace struct {
		models.Place
		DistanceKm float64 `json:"distance_km"`
	}

	places := []NearbyPlace{}
	for rows.Next() {
		var p NearbyPlace
		err := rows.Scan(
			&p.ID, &p.UserID, &p.Name, &p.Category, &p.Description,
			&p.Address, &p.Phone, &p.Website,
			&p.Longitude, &p.Latitude,
			&p.AverageRating, &p.TotalRatings,
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
	userID := middleware.GetUserID(r)

	db := db.GetDB()
	query := `
        SELECT DISTINCT category 
        FROM places 
        WHERE user_id = $1 
        ORDER BY category
    `

	rows, err := db.Query(query, userID)
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
