package models

import "time"

type User struct {
	ID           int       `json:"id"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}

type Place struct {
	ID            int       `json:"id"`
	UserID        int       `json:"user_id"`
	Name          string    `json:"name"`
	Category      string    `json:"category"`
	Description   string    `json:"description"`
	Address       string    `json:"address"`
	Phone         string    `json:"phone"`
	Website       string    `json:"website"`
	Longitude     float64   `json:"longitude"`
	Latitude      float64   `json:"latitude"`
	AverageRating float64   `json:"average_rating"`
	TotalRatings  int       `json:"total_ratings"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	Photos        []Photo   `json:"photos,omitempty"`
	UserRating    *Rating   `json:"user_rating,omitempty"`
}

type Photo struct {
	ID         int       `json:"id"`
	PlaceID    int       `json:"place_id"`
	UserID     int       `json:"user_id"`
	Filename   string    `json:"filename"`
	FilePath   string    `json:"file_path"`
	FileSize   int       `json:"file_size"`
	MimeType   string    `json:"mime_type"`
	IsPrimary  bool      `json:"is_primary"`
	UploadedAt time.Time `json:"uploaded_at"`
}

type Rating struct {
	ID        int       `json:"id"`
	PlaceID   int       `json:"place_id"`
	UserID    int       `json:"user_id"`
	Rating    int       `json:"rating"`
	Review    string    `json:"review"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CreatePlaceRequest struct {
	Name        string  `json:"name"`
	Category    string  `json:"category"`
	Description string  `json:"description"`
	Address     string  `json:"address"`
	Phone       string  `json:"phone"`
	Website     string  `json:"website"`
	Longitude   float64 `json:"longitude"`
	Latitude    float64 `json:"latitude"`
}

type RatePlaceRequest struct {
	Rating int    `json:"rating"`
	Review string `json:"review"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type RegisterRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}
