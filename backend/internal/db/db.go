package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
)

var DB *sql.DB

func InitDB() {
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

	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname,
	)

	var err error
	DB, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	if err = DB.Ping(); err != nil {
		log.Fatal("Failed to ping database:", err)
	}

	log.Println("✅ Connected to PostGIS database")

	// Create tables if they don't exist
	createTables()
}

func createTables() {
	log.Println("📋 Creating tables if they don't exist...")

	// Create users table
	usersTable := `
        CREATE TABLE IF NOT EXISTS users (
            id SERIAL PRIMARY KEY,
            username VARCHAR(50) UNIQUE NOT NULL,
            email VARCHAR(100) UNIQUE NOT NULL,
            password_hash VARCHAR(255) NOT NULL,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        );
    `

	_, err := DB.Exec(usersTable)
	if err != nil {
		log.Fatalf("❌ Failed to create users table: %v", err)
	}
	log.Println("✅ Users table ready")

	// Create places table with PostGIS
	placesTable := `
        CREATE TABLE IF NOT EXISTS places (
            id SERIAL PRIMARY KEY,
            user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
            name VARCHAR(100) NOT NULL,
            category VARCHAR(50) NOT NULL,
            description TEXT,
            address TEXT,
            phone VARCHAR(20),
            website VARCHAR(255),
            geom GEOMETRY(Point, 4326) NOT NULL,
            average_rating DECIMAL(3,2) DEFAULT 0,
            total_ratings INTEGER DEFAULT 0,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        );
    `

	_, err = DB.Exec(placesTable)
	if err != nil {
		log.Fatalf("❌ Failed to create places table: %v", err)
	}
	log.Println("✅ Places table ready")

	// Create spatial index
	indexQuery := `
        CREATE INDEX IF NOT EXISTS idx_places_geom 
        ON places USING GIST (geom);
    `
	_, err = DB.Exec(indexQuery)
	if err != nil {
		log.Printf("⚠️ Failed to create spatial index: %v", err)
	}

	// Create photos table
	photosTable := `
        CREATE TABLE IF NOT EXISTS photos (
            id SERIAL PRIMARY KEY,
            place_id INTEGER REFERENCES places(id) ON DELETE CASCADE,
            user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
            filename VARCHAR(255) NOT NULL,
            file_path VARCHAR(500) NOT NULL,
            file_size INTEGER,
            mime_type VARCHAR(100),
            is_primary BOOLEAN DEFAULT FALSE,
            uploaded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        );
    `

	_, err = DB.Exec(photosTable)
	if err != nil {
		log.Printf("⚠️ Failed to create photos table: %v", err)
	}
	log.Println("✅ Photos table ready")

	// Create ratings table
	ratingsTable := `
        CREATE TABLE IF NOT EXISTS ratings (
            id SERIAL PRIMARY KEY,
            place_id INTEGER REFERENCES places(id) ON DELETE CASCADE,
            user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
            rating INTEGER CHECK (rating >= 1 AND rating <= 5),
            review TEXT,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            UNIQUE(place_id, user_id)
        );
    `

	_, err = DB.Exec(ratingsTable)
	if err != nil {
		log.Printf("⚠️ Failed to create ratings table: %v", err)
	}
	log.Println("✅ Ratings table ready")

	// Create function to update average rating
	functionSQL := `
        CREATE OR REPLACE FUNCTION update_place_rating()
        RETURNS TRIGGER AS $$
        BEGIN
            UPDATE places
            SET 
                average_rating = (
                    SELECT AVG(rating)::DECIMAL(3,2)
                    FROM ratings
                    WHERE place_id = NEW.place_id
                ),
                total_ratings = (
                    SELECT COUNT(*)
                    FROM ratings
                    WHERE place_id = NEW.place_id
                )
            WHERE id = NEW.place_id;
            RETURN NEW;
        END;
        $$ LANGUAGE plpgsql;
    `

	_, err = DB.Exec(functionSQL)
	if err != nil {
		log.Printf("⚠️ Failed to create rating function: %v", err)
	}

	// Create trigger
	triggerSQL := `
        DROP TRIGGER IF EXISTS after_rating_change ON ratings;
        CREATE TRIGGER after_rating_change
        AFTER INSERT OR UPDATE OR DELETE ON ratings
        FOR EACH ROW
        EXECUTE FUNCTION update_place_rating();
    `

	_, err = DB.Exec(triggerSQL)
	if err != nil {
		log.Printf("⚠️ Failed to create rating trigger: %v", err)
	}

	log.Println("✅ Database schema initialized successfully!")
}

func GetDB() *sql.DB {
	return DB
}
