package main

import (
	"log"
	"net/http"
	"os"

	"demomap/internal/db"
	"demomap/internal/handlers"
	"demomap/internal/middleware"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"github.com/rs/cors"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using default settings")
	}

	// Initialize JWT
	middleware.InitJWT()
	handlers.InitJWTSecret(os.Getenv("JWT_SECRET"))

	// Initialize db
	db.InitDB()
	defer db.GetDB().Close()

	// Create uploads directory
	os.MkdirAll("./uploads", 0755)
	os.MkdirAll("./static", 0755)

	// Set up router
	router := mux.NewRouter()

	// Public routes
	router.HandleFunc("/api/health", handlers.HealthHandler).Methods("GET")
	router.HandleFunc("/api/register", handlers.RegisterHandler).Methods("POST")
	router.HandleFunc("/api/login", handlers.LoginHandler).Methods("POST")

	// Protected routes
	api := router.PathPrefix("/api").Subrouter()
	api.Use(middleware.AuthMiddleware)

	// Place CRUD
	api.HandleFunc("/places", handlers.GetPlacesHandler).Methods("GET")
	api.HandleFunc("/places", handlers.CreatePlaceHandler).Methods("POST")
	api.HandleFunc("/places/{id}", handlers.GetPlaceHandler).Methods("GET")
	api.HandleFunc("/places/{id}", handlers.UpdatePlaceHandler).Methods("PUT")
	api.HandleFunc("/places/{id}", handlers.DeletePlaceHandler).Methods("DELETE")
	api.HandleFunc("/places/nearby", handlers.NearbyPlacesHandler).Methods("GET")

	// Photo upload
	api.HandleFunc("/places/{id}/photos", handlers.UploadPhotoHandler).Methods("POST")
	api.HandleFunc("/places/{id}/photos/{photoId}", handlers.DeletePhotoHandler).Methods("DELETE")
	api.HandleFunc("/places/{id}/photos/{photoId}/primary", handlers.SetPrimaryPhotoHandler).Methods("PUT")

	// Ratings
	api.HandleFunc("/places/{id}/rate", handlers.RatePlaceHandler).Methods("POST")
	api.HandleFunc("/places/{id}/ratings", handlers.GetPlaceRatingsHandler).Methods("GET")

	// Categories
	api.HandleFunc("/categories", handlers.GetCategoriesHandler).Methods("GET")

	// Serve static files
	router.PathPrefix("/uploads/").Handler(http.StripPrefix("/uploads/", http.FileServer(http.Dir("./uploads"))))
	router.PathPrefix("/").Handler(http.FileServer(http.Dir("../frontend")))

	// ============================================
	// OGC API - Features Routes (Public)
	// ============================================
	router.HandleFunc("/", handlers.OGCLandingHandler).Methods("GET")
	router.HandleFunc("/conformance", handlers.OGCConformanceHandler).Methods("GET")
	router.HandleFunc("/collections", handlers.OGCCollectionsHandler).Methods("GET")
	router.HandleFunc("/collections/{collectionId}/items", handlers.OGCItemsHandler).Methods("GET")
	router.HandleFunc("/collections/{collectionId}/items/{featureId}", handlers.OGCFeatureHandler).Methods("GET")

	// Enable CORS
	handler := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	}).Handler(router)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	log.Printf("🚀 Server starting on http://localhost:%s", port)
	log.Printf("📱 Map available at http://localhost:%s", port)
	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatal(err)
	}
}
