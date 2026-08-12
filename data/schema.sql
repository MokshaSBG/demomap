-- Enable PostGIS
CREATE EXTENSION IF NOT EXISTS postgis;

-- Users table (for authentication)
CREATE TABLE users (
                       id SERIAL PRIMARY KEY,
                       username VARCHAR(50) UNIQUE NOT NULL,
                       email VARCHAR(100) UNIQUE NOT NULL,
                       password_hash VARCHAR(255) NOT NULL,
                       created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Places table (enhanced)
CREATE TABLE places (
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

-- Create spatial index
CREATE INDEX idx_places_geom ON places USING GIST (geom);

-- Photos table
CREATE TABLE photos (
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

-- Ratings table
CREATE TABLE ratings (
                         id SERIAL PRIMARY KEY,
                         place_id INTEGER REFERENCES places(id) ON DELETE CASCADE,
                         user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
                         rating INTEGER CHECK (rating >= 1 AND rating <= 5),
                         review TEXT,
                         created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                         updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                         UNIQUE(place_id, user_id)  -- One rating per user per place
);

-- Create indexes for performance
CREATE INDEX idx_ratings_place_id ON ratings(place_id);
CREATE INDEX idx_ratings_user_id ON ratings(user_id);
CREATE INDEX idx_photos_place_id ON photos(place_id);

-- Function to update average rating
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

-- Trigger to auto-update rating
CREATE TRIGGER after_rating_change
    AFTER INSERT OR UPDATE OR DELETE ON ratings
    FOR EACH ROW
    EXECUTE FUNCTION update_place_rating();

-- Insert sample users
INSERT INTO users (username, email, password_hash) VALUES
    ('demo_user', 'demo@example.com', 'hashed_password_here');

-- Insert sample places
INSERT INTO places (user_id, name, category, description, address, geom) VALUES
                                                                             (1, 'Tartine Bakery', 'restaurant', 'Famous for bread and pastries', '600 Guerrero St, San Francisco, CA', ST_SetSRID(ST_MakePoint(-122.4204, 37.7626), 4326)),
                                                                             (1, 'Golden Gate Park', 'park', 'Large urban park with gardens and museums', 'San Francisco, CA', ST_SetSRID(ST_MakePoint(-122.4826, 37.7700), 4326)),
                                                                             (1, 'Sutro Tower', 'landmark', 'Iconic communications tower', '1 Sutro Tower Ave, San Francisco, CA', ST_SetSRID(ST_MakePoint(-122.4529, 37.7552), 4326));

-- Insert sample ratings
INSERT INTO ratings (place_id, user_id, rating, review) VALUES
                                                            (1, 1, 5, 'Best bakery in SF!'),
                                                            (1, 1, 4, 'Great pastries but long lines'),
                                                            (2, 1, 5, 'Beautiful park, perfect for a weekend visit');