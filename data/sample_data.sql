-- Connect to the db first
-- psql -U mapuser -d mapdb

-- Enable PostGIS extension (if not already enabled)
CREATE EXTENSION IF NOT EXISTS postgis;

-- Drop table if exists (for clean setup)
DROP TABLE IF EXISTS locations;

-- Create table
CREATE TABLE locations (
                           id SERIAL PRIMARY KEY,
                           name VARCHAR(100) NOT NULL,
                           category VARCHAR(50) NOT NULL,
                           description TEXT,
                           geom GEOMETRY(Point, 4326) NOT NULL,
                           created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create spatial index
CREATE INDEX idx_locations_geom ON locations USING GIST (geom);

-- Insert sample data (San Francisco landmarks)
INSERT INTO locations (name, category, description, geom) VALUES
                                                              ('Tartine Bakery', 'restaurant', 'Famous for bread and pastries', ST_SetSRID(ST_MakePoint(-122.4204, 37.7626), 4326)),
                                                              ('Zuni Cafe', 'restaurant', 'Iconic California cuisine', ST_SetSRID(ST_MakePoint(-122.4211, 37.7823), 4326)),
                                                              ('The Slanted Door', 'restaurant', 'Modern Vietnamese cuisine', ST_SetSRID(ST_MakePoint(-122.3950, 37.7864), 4326)),
                                                              ('Golden Gate Park', 'park', 'Large urban park with gardens and museums', ST_SetSRID(ST_MakePoint(-122.4826, 37.7700), 4326)),
                                                              ('Dolores Park', 'park', 'Popular city park with skyline views', ST_SetSRID(ST_MakePoint(-122.4269, 37.7595), 4326)),
                                                              ('Sutro Tower', 'landmark', 'Iconic communications tower visible from many parts of the city', ST_SetSRID(ST_MakePoint(-122.4529, 37.7552), 4326)),
                                                              ('Alcatraz Island', 'landmark', 'Historic federal prison on an island', ST_SetSRID(ST_MakePoint(-122.4224, 37.8263), 4326)),
                                                              ('Muir Woods', 'park', 'Coastal redwood forest just north of SF', ST_SetSRID(ST_MakePoint(-122.5816, 37.9027), 4326)),
                                                              ('Souvla', 'restaurant', 'Greek rotisserie with modern flair', ST_SetSRID(ST_MakePoint(-122.4309, 37.7734), 4326)),
                                                              ('Mission Dolores', 'landmark', 'Mission San Francisco de Asís, oldest building in SF', ST_SetSRID(ST_MakePoint(-122.4269, 37.7643), 4326));

-- Verify data
SELECT id, name, category, ST_X(geom) as lon, ST_Y(geom) as lat FROM locations;