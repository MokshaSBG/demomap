package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"demomap/internal/db"
	"demomap/internal/models"
	"demomap/internal/utils"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var jwtSecret = []byte("your-secret-key-change-in-production")

func InitJWTSecret(secret string) {
	if secret != "" {
		jwtSecret = []byte(secret)
	}
}

func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	var req models.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.JSONError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Username == "" || req.Email == "" || req.Password == "" {
		utils.JSONError(w, http.StatusBadRequest, "All fields are required")
		return
	}

	if len(req.Password) < 6 {
		utils.JSONError(w, http.StatusBadRequest, "Password must be at least 6 characters")
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		utils.JSONError(w, http.StatusInternalServerError, "Failed to hash password")
		return
	}

	db := db.GetDB()

	query := `INSERT INTO users (username, email, password_hash) VALUES ($1, $2, $3) RETURNING id`
	var userID int
	err = db.QueryRow(query, req.Username, req.Email, string(hashedPassword)).Scan(&userID)
	if err != nil {
		if strings.Contains(err.Error(), "unique constraint") {
			utils.JSONError(w, http.StatusConflict, "Username or email already exists")
			return
		}
		utils.JSONError(w, http.StatusInternalServerError, "Failed to create user")
		return
	}

	utils.JSONResponse(w, http.StatusCreated, map[string]interface{}{
		"message": "User registered successfully",
		"user_id": userID,
	})
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.JSONError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	db := db.GetDB()
	var user models.User
	query := `SELECT id, username, email, password_hash FROM users WHERE username = $1`
	err := db.QueryRow(query, req.Username).Scan(&user.ID, &user.Username, &user.Email, &user.PasswordHash)
	if err != nil {
		utils.JSONError(w, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		utils.JSONError(w, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":  user.ID,
		"username": user.Username,
		"exp":      time.Now().Add(time.Hour * 24 * 7).Unix(),
	})

	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		utils.JSONError(w, http.StatusInternalServerError, "Failed to generate token")
		return
	}

	utils.JSONResponse(w, http.StatusOK, map[string]interface{}{
		"token": tokenString,
		"user": map[string]interface{}{
			"id":       user.ID,
			"username": user.Username,
			"email":    user.Email,
		},
	})
}

func HealthHandler(w http.ResponseWriter, r *http.Request) {
	utils.JSONResponse(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"message": "Map backend is running",
	})
}
