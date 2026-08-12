package handlers

import (
	"database/sql"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"demomap/internal/db"
	"demomap/internal/middleware"
	"demomap/internal/utils"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

func UploadPhotoHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	placeID := vars["id"]
	userID := middleware.GetUserID(r)

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		utils.JSONError(w, http.StatusBadRequest, "File too large (max 10MB)")
		return
	}

	file, header, err := r.FormFile("photo")
	if err != nil {
		utils.JSONError(w, http.StatusBadRequest, "No file uploaded")
		return
	}
	defer file.Close()

	contentType := header.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "image/") {
		utils.JSONError(w, http.StatusBadRequest, "Only image files are allowed")
		return
	}

	ext := filepath.Ext(header.Filename)
	filename := uuid.New().String() + ext
	filepath := "./uploads/" + filename

	out, err := os.Create(filepath)
	if err != nil {
		utils.JSONError(w, http.StatusInternalServerError, "Failed to save file")
		return
	}
	defer out.Close()

	_, err = io.Copy(out, file)
	if err != nil {
		os.Remove(filepath)
		utils.JSONError(w, http.StatusInternalServerError, "Failed to save file")
		return
	}

	db := db.GetDB()
	var count int
	db.QueryRow(`SELECT COUNT(*) FROM photos WHERE place_id = $1`, placeID).Scan(&count)
	isPrimary := count == 0

	query := `
        INSERT INTO photos (place_id, user_id, filename, file_path, file_size, mime_type, is_primary)
        VALUES ($1, $2, $3, $4, $5, $6, $7)
        RETURNING id
    `
	var photoID int
	err = db.QueryRow(query, placeID, userID, filename, filepath, header.Size, contentType, isPrimary).Scan(&photoID)
	if err != nil {
		os.Remove(filepath)
		utils.JSONError(w, http.StatusInternalServerError, "Failed to save photo metadata")
		return
	}

	utils.JSONResponse(w, http.StatusCreated, map[string]interface{}{
		"message":  "Photo uploaded successfully",
		"photo_id": photoID,
		"url":      "/uploads/" + filename,
	})
}

func DeletePhotoHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	photoID := vars["photoId"]
	userID := middleware.GetUserID(r)

	db := db.GetDB()
	var filePath string
	var placeID int
	query := `SELECT file_path, place_id FROM photos WHERE id = $1 AND user_id = $2`
	err := db.QueryRow(query, photoID, userID).Scan(&filePath, &placeID)
	if err != nil {
		if err == sql.ErrNoRows {
			utils.JSONError(w, http.StatusNotFound, "Photo not found")
			return
		}
		utils.JSONError(w, http.StatusInternalServerError, "Failed to get photo")
		return
	}

	_, err = db.Exec(`DELETE FROM photos WHERE id = $1`, photoID)
	if err != nil {
		utils.JSONError(w, http.StatusInternalServerError, "Failed to delete photo")
		return
	}

	os.Remove(filePath)

	db.Exec(`
        UPDATE photos 
        SET is_primary = TRUE 
        WHERE place_id = $1 
        ORDER BY id 
        LIMIT 1
    `, placeID)

	utils.JSONResponse(w, http.StatusOK, map[string]string{
		"message": "Photo deleted successfully",
	})
}

func SetPrimaryPhotoHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	placeID := vars["id"]
	photoID := vars["photoId"]
	userID := middleware.GetUserID(r)

	db := db.GetDB()
	var count int
	err := db.QueryRow(`SELECT COUNT(*) FROM photos WHERE id = $1 AND place_id = $2 AND user_id = $3`,
		photoID, placeID, userID).Scan(&count)
	if err != nil || count == 0 {
		utils.JSONError(w, http.StatusNotFound, "Photo not found")
		return
	}

	_, err = db.Exec(`UPDATE photos SET is_primary = FALSE WHERE place_id = $1`, placeID)
	if err != nil {
		utils.JSONError(w, http.StatusInternalServerError, "Failed to update primary photo")
		return
	}

	_, err = db.Exec(`UPDATE photos SET is_primary = TRUE WHERE id = $1`, photoID)
	if err != nil {
		utils.JSONError(w, http.StatusInternalServerError, "Failed to set primary photo")
		return
	}

	utils.JSONResponse(w, http.StatusOK, map[string]string{
		"message": "Primary photo updated successfully",
	})
}
