// Configuration
const API_BASE_URL = 'http://localhost:8080/api';
let map;
let markersLayer;
let currentData = [];

// Initialize map
function initMap() {
    map = L.map('map').setView([37.7749, -122.4194], 13);

    L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
        attribution: '© OpenStreetMap contributors'
    }).addTo(map);

    markersLayer = L.layerGroup().addTo(map);

    // Load initial data
    loadLocations();

    // Map click event - find nearby
    map.on('click', function(e) {
        const lat = e.latlng.lat;
        const lng = e.latlng.lng;
        findNearby(lat, lng);
    });
}

// Load all locations
async function loadLocations() {
    try {
        showLoading(true);
        const response = await fetch(`${API_BASE_URL}/locations`);
        if (!response.ok) throw new Error('Failed to load locations');
        const data = await response.json();

        currentData = data.features;
        renderLocations(data.features);
        loadCategories(data.features);
        showLoading(false);
    } catch (error) {
        console.error('Error loading locations:', error);
        showLoading(false);
        document.getElementById('location-list').innerHTML =
            '<p class="no-results">❌ Failed to load locations</p>';
    }
}

// Render locations on map and list
function renderLocations(features) {
    // Clear existing markers
    markersLayer.clearLayers();

    const listEl = document.getElementById('location-list');
    listEl.innerHTML = '';

    if (features.length === 0) {
        listEl.innerHTML = '<p class="no-results">No locations found</p>';
        return;
    }

    features.forEach(feature => {
        const props = feature.properties;
        const coords = feature.geometry.coordinates;
        const lat = coords[1];
        const lng = coords[0];

        // Add marker to map
        const marker = L.circleMarker([lat, lng], {
            radius: 10,
            fillColor: getCategoryColor(props.category),
            color: '#fff',
            weight: 2,
            opacity: 1,
            fillOpacity: 0.9
        }).addTo(markersLayer);

        // Popup content
        const popupContent = `
            <div class="popup-content">
                <h3>${props.name}</h3>
                <span class="category">${props.category}</span>
                <p class="description">${props.description || ''}</p>
                ${props.distance_km ? `<p>📍 ${props.distance_km} km away</p>` : ''}
            </div>
        `;
        marker.bindPopup(popupContent);

        // Add to list
        const item = document.createElement('div');
        item.className = 'location-item';
        item.innerHTML = `
            <h3>${props.name}</h3>
            <span class="category">${props.category}</span>
            ${props.description ? `<p class="description">${props.description}</p>` : ''}
            ${props.distance_km ? `<p class="distance">📍 ${props.distance_km} km away</p>` : ''}
        `;
        item.addEventListener('click', () => {
            map.setView([lat, lng], 16);
            marker.openPopup();
        });
        listEl.appendChild(item);
    });

    // Fit map to show all markers
    if (features.length > 0) {
        const group = L.featureGroup(
            features.map(f => L.latLng(f.geometry.coordinates[1], f.geometry.coordinates[0]))
        );
        map.fitBounds(group.getBounds(), { padding: [50, 50] });
    }
}

// Load categories into filter dropdown
function loadCategories(features) {
    const categories = new Set();
    features.forEach(f => {
        if (f.properties.category) {
            categories.add(f.properties.category);
        }
    });

    const select = document.getElementById('categoryFilter');
    // Keep "All Categories" option
    while (select.options.length > 1) {
        select.remove(1);
    }

    categories.forEach(cat => {
        const option = document.createElement('option');
        option.value = cat;
        option.textContent = cat.charAt(0).toUpperCase() + cat.slice(1);
        select.appendChild(option);
    });
}

// Get color for category
function getCategoryColor(category) {
    const colors = {
        restaurant: '#dc3545',
        park: '#28a745',
        landmark: '#ffc107'
    };
    return colors[category] || '#0d6efd';
}

// Search locations
function searchLocations(query) {
    const filtered = currentData.filter(feature => {
        const props = feature.properties;
        const searchTerm = query.toLowerCase();
        return props.name.toLowerCase().includes(searchTerm) ||
            props.category.toLowerCase().includes(searchTerm) ||
            (props.description && props.description.toLowerCase().includes(searchTerm));
    });
    renderLocations(filtered);
}

// Filter by category
function filterByCategory(category) {
    if (category === 'all') {
        renderLocations(currentData);
        return;
    }
    const filtered = currentData.filter(feature =>
        feature.properties.category === category
    );
    renderLocations(filtered);
}

// Find nearby locations
async function findNearby(lat, lon, radius = 5) {
    try {
        showLoading(true);
        const url = `${API_BASE_URL}/locations/nearby?lat=${lat}&lon=${lon}&radius=${radius}`;
        const response = await fetch(url);
        if (!response.ok) throw new Error('Failed to find nearby locations');
        const data = await response.json();

        currentData = data.features;
        renderLocations(data.features);
        showLoading(false);

        // Add a marker at the search location
        L.circleMarker([lat, lon], {
            radius: 8,
            color: '#0d6efd',
            weight: 3,
            opacity: 0.8,
            fillColor: '#0d6efd',
            fillOpacity: 0.3
        }).addTo(markersLayer);

    } catch (error) {
        console.error('Error finding nearby:', error);
        showLoading(false);
        alert('Failed to find nearby locations');
    }
}

// Show/hide loading state
function showLoading(show) {
    const listEl = document.getElementById('location-list');
    if (show) {
        listEl.innerHTML = '<p class="loading-msg">Loading locations...</p>';
    }
}

// Event Listeners
document.addEventListener('DOMContentLoaded', () => {
    initMap();

    // Search input
    document.getElementById('searchInput').addEventListener('input', (e) => {
        const query = e.target.value.trim();
        if (query.length > 0) {
            searchLocations(query);
        } else {
            renderLocations(currentData);
        }
    });

    // Category filter
    document.getElementById('categoryFilter').addEventListener('change', (e) => {
        filterByCategory(e.target.value);
    });

    // Find nearby button
    document.getElementById('findNearbyBtn').addEventListener('click', () => {
        if ('geolocation' in navigator) {
            navigator.geolocation.getCurrentPosition(
                (position) => {
                    const lat = position.coords.latitude;
                    const lon = position.coords.longitude;
                    const radius = parseFloat(document.getElementById('radiusInput').value) || 5;
                    findNearby(lat, lon, radius);
                    map.setView([lat, lon], 14);
                },
                (error) => {
                    alert('Unable to get your location. Please enable location services.');
                    console.error('Geolocation error:', error);
                }
            );
        } else {
            alert('Geolocation is not supported by your browser');
        }
    });
});