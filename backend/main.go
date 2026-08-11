package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/rs/cors"
)

type Location struct {
	ID          int     `json:"id"`
	Name        string  `json:"name"`
	Category    string  `json:"category"`
	Description string  `json:"description"`
	Longitude   float64 `json:"longitude"`
	Latitude    float64 `json:"latitude"`
}

type GeoJSONFeature struct {
	Type       string                 `json:"type"`
	Geometry   GeoJSONGeometry        `json:"geometry"`
	Properties map[string]interface{} `json:"properties"`
}

type GeoJSONGeometry struct {
	Type        string    `json:"type"`
	Coordinates []float64 `json:"coordinates"`
}

type GeoJSONResponse struct {
	Type     string           `json:"type"`
	Features []GeoJSONFeature `json:"features"`
}

var db *sql.DB

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using default settings")
	}

	// Initialize database connection
	initDB()
	defer db.Close()

	// Set up routes
	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", healthHandler)
	mux.HandleFunc("/api/locations", locationsHandler)
	mux.HandleFunc("/api/locations/nearby", nearbyLocationsHandler)
	mux.HandleFunc("/api/categories", categoriesHandler)

	// Serve static files (frontend)
	mux.Handle("/", http.FileServer(http.Dir("../frontend")))

	// Enable CORS
	handler := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	}).Handler(mux)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on http://localhost:%s", port)
	log.Printf("Map available at http://localhost:%s", port)
	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatal(err)
	}
}

func initDB() {
	// Database connection parameters
	host := os.Getenv("DB_HOST")
	if host == "" {
		host = "localhost"
	}
	port := os.Getenv("DB_PORT")
	if port == "" {
		port = "5432"
	}
	user := os.Getenv("DB_USER")
	if user == "" {
		user = "demomapuser"
	}
	password := os.Getenv("DB_PASSWORD")
	if password == "" {
		password = "password"
	}
	dbname := os.Getenv("DB_NAME")
	if dbname == "" {
		dbname = "demomap"
	}

	// Connection string
	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname,
	)

	var err error
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Test connection
	if err = db.Ping(); err != nil {
		log.Fatal("Failed to ping database:", err)
	}

	log.Println("✅ Connected to PostGIS database successfully!")
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "ok",
		"message": "Map backend is running",
	})
}

func locationsHandler(w http.ResponseWriter, r *http.Request) {
	query := `
		SELECT 
			id, 
			name, 
			category, 
			description,
			ST_X(geom) as longitude,
			ST_Y(geom) as latitude
		FROM locations
		ORDER BY id
	`

	rows, err := db.Query(query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	features := []GeoJSONFeature{}
	for rows.Next() {
		var loc Location
		err := rows.Scan(
			&loc.ID,
			&loc.Name,
			&loc.Category,
			&loc.Description,
			&loc.Longitude,
			&loc.Latitude,
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		feature := GeoJSONFeature{
			Type: "Feature",
			Geometry: GeoJSONGeometry{
				Type:        "Point",
				Coordinates: []float64{loc.Longitude, loc.Latitude},
			},
			Properties: map[string]interface{}{
				"id":          loc.ID,
				"name":        loc.Name,
				"category":    loc.Category,
				"description": loc.Description,
			},
		}
		features = append(features, feature)
	}

	response := GeoJSONResponse{
		Type:     "FeatureCollection",
		Features: features,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func nearbyLocationsHandler(w http.ResponseWriter, r *http.Request) {
	latStr := r.URL.Query().Get("lat")
	lonStr := r.URL.Query().Get("lon")
	radiusStr := r.URL.Query().Get("radius")

	if latStr == "" || lonStr == "" {
		http.Error(w, "Missing lat or lon parameters", http.StatusBadRequest)
		return
	}

	lat, err := strconv.ParseFloat(latStr, 64)
	if err != nil {
		http.Error(w, "Invalid lat parameter", http.StatusBadRequest)
		return
	}

	lon, err := strconv.ParseFloat(lonStr, 64)
	if err != nil {
		http.Error(w, "Invalid lon parameter", http.StatusBadRequest)
		return
	}

	radius := 5.0 // default 5km
	if radiusStr != "" {
		radius, err = strconv.ParseFloat(radiusStr, 64)
		if err != nil {
			http.Error(w, "Invalid radius parameter", http.StatusBadRequest)
			return
		}
	}

	query := `
		SELECT 
			id, 
			name, 
			category, 
			description,
			ST_X(geom) as longitude,
			ST_Y(geom) as latitude,
			ST_Distance(
				geom, 
				ST_SetSRID(ST_MakePoint($1, $2), 4326)
			) * 111.319 as distance_km
		FROM locations
		WHERE ST_Distance(
			geom, 
			ST_SetSRID(ST_MakePoint($1, $2), 4326)
		) < ($3 / 111.319)
		ORDER BY distance_km
	`

	rows, err := db.Query(query, lon, lat, radius)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	features := []GeoJSONFeature{}
	for rows.Next() {
		var loc Location
		var distanceKm float64
		err := rows.Scan(
			&loc.ID,
			&loc.Name,
			&loc.Category,
			&loc.Description,
			&loc.Longitude,
			&loc.Latitude,
			&distanceKm,
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		feature := GeoJSONFeature{
			Type: "Feature",
			Geometry: GeoJSONGeometry{
				Type:        "Point",
				Coordinates: []float64{loc.Longitude, loc.Latitude},
			},
			Properties: map[string]interface{}{
				"id":          loc.ID,
				"name":        loc.Name,
				"category":    loc.Category,
				"description": loc.Description,
				"distance_km": fmt.Sprintf("%.2f", distanceKm),
			},
		}
		features = append(features, feature)
	}

	response := GeoJSONResponse{
		Type:     "FeatureCollection",
		Features: features,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func categoriesHandler(w http.ResponseWriter, r *http.Request) {
	query := `SELECT DISTINCT category FROM locations ORDER BY category`

	rows, err := db.Query(query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	categories := []string{}
	for rows.Next() {
		var category string
		if err := rows.Scan(&category); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		categories = append(categories, category)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(categories)
}
