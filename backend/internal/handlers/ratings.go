package handlers

import (
	"encoding/json"
	"net/http"

	"demomap/internal/db"
	"demomap/internal/middleware"
	"demomap/internal/models"
	"demomap/internal/utils"

	"github.com/gorilla/mux"
)

func RatePlaceHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	placeID := vars["id"]
	userID := middleware.GetUserID(r)

	var req models.RatePlaceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.JSONError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Rating < 1 || req.Rating > 5 {
		utils.JSONError(w, http.StatusBadRequest, "Rating must be between 1 and 5")
		return
	}

	db := db.GetDB()
	var exists bool
	err := db.QueryRow(`SELECT EXISTS(SELECT 1 FROM places WHERE id = $1)`, placeID).Scan(&exists)
	if err != nil || !exists {
		utils.JSONError(w, http.StatusNotFound, "Place not found")
		return
	}
	query := `
        INSERT INTO ratings (place_id, user_id, rating, review)
        VALUES ($1, $2, $3, $4)
        ON CONFLICT (place_id, user_id) 
        DO UPDATE SET rating = $3, review = $4, updated_at = CURRENT_TIMESTAMP
        RETURNING id
    `
	var ratingID int
	err = db.QueryRow(query, placeID, userID, req.Rating, req.Review).Scan(&ratingID)
	if err != nil {
		utils.JSONError(w, http.StatusInternalServerError, "Failed to save rating")
		return
	}

	utils.JSONResponse(w, http.StatusOK, map[string]interface{}{
		"message":   "Rating saved successfully",
		"rating_id": ratingID,
	})
}

func GetPlaceRatingsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	placeID := vars["id"]

	db := db.GetDB()
	query := `
        SELECT 
            r.id, r.place_id, r.user_id, r.rating, r.review, 
            r.created_at, r.updated_at,
            u.username
        FROM ratings r
        JOIN users u ON r.user_id = u.id
        WHERE r.place_id = $1
        ORDER BY r.created_at DESC
    `

	rows, err := db.Query(query, placeID)
	if err != nil {
		utils.JSONError(w, http.StatusInternalServerError, "Failed to fetch ratings")
		return
	}
	defer rows.Close()

	type RatingWithUser struct {
		models.Rating
		Username string `json:"username"`
	}

	ratings := []RatingWithUser{}
	for rows.Next() {
		var r RatingWithUser
		err := rows.Scan(
			&r.ID, &r.PlaceID, &r.UserID, &r.Rating, &r.Review,
			&r.CreatedAt, &r.UpdatedAt, &r.Username,
		)
		if err != nil {
			utils.JSONError(w, http.StatusInternalServerError, "Failed to scan rating")
			return
		}
		ratings = append(ratings, r)
	}

	utils.JSONResponse(w, http.StatusOK, ratings)
}
