// ============================================
// STATE
// ============================================
console.log('🔥 app.js started loading');

const state = {
    token: null,
    user: null,
    places: [],
    map: null,
    markersLayer: null,
    selectedMarker: null,
    selectedLat: null,
    selectedLng: null,
    ratingValue: 0,
    currentPlaceId: null,
    editingPlaceId: null,
};

// ============================================
// API HELPERS
// ============================================

const API_BASE = 'http://localhost:8081/api';

async function apiRequest(endpoint, options = {}) {
    const headers = {
        'Content-Type': 'application/json',
        ...options.headers,
    };

    if (state.token) {
        headers['Authorization'] = `Bearer ${state.token}`;
    }

    const response = await fetch(`${API_BASE}${endpoint}`, {
        ...options,
        headers,
    });

    if (response.status === 401) {
        logout();
        throw new Error('Session expired. Please login again.');
    }

    const data = await response.json();
    if (!response.ok) {
        throw new Error(data.error || 'API request failed');
    }
    return data;
}

// ============================================
// AUTHENTICATION
// ============================================

async function login(username, password) {
    try {
        const data = await apiRequest('/login', {
            method: 'POST',
            body: JSON.stringify({ username, password }),
        });
        state.token = data.token;
        state.user = data.user;
        localStorage.setItem('token', state.token);
        localStorage.setItem('user', JSON.stringify(state.user));
        showApp();
        loadPlaces();
        hideAuthError();
    } catch (error) {
        showAuthError(error.message);
    }
}

async function register(username, email, password) {
    try {
        await apiRequest('/register', {
            method: 'POST',
            body: JSON.stringify({ username, email, password }),
        });
        await login(username, password);
    } catch (error) {
        showAuthError(error.message);
    }
}

function logout() {
    state.token = null;
    state.user = null;
    localStorage.removeItem('token');
    localStorage.removeItem('user');
    document.getElementById('app').style.display = 'none';
    document.getElementById('authModal').classList.add('active');
    if (state.map) {
        state.map.remove();
        state.map = null;
    }
}

function checkAuth() {
    const token = localStorage.getItem('token');
    const user = localStorage.getItem('user');
    if (token && user) {
        state.token = token;
        state.user = JSON.parse(user);
        showApp();
        loadPlaces();
    } else {
        document.getElementById('authModal').classList.add('active');
    }
}

function showApp() {
    document.getElementById('app').style.display = 'flex';
    document.getElementById('authModal').classList.remove('active');
    document.getElementById('currentUser').textContent = state.user.username;
    initMap();
}

function showAuthError(message) {
    const el = document.getElementById('authError');
    el.textContent = message;
    el.classList.add('show');
}

function hideAuthError() {
    document.getElementById('authError').classList.remove('show');
}

// ============================================
// MAP
// ============================================

function initMap() {
    if (state.map) {
        state.map.invalidateSize();
        return;
    }

    state.map = L.map('map').setView([37.7749, -122.4194], 13);

    L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
        attribution: '© OpenStreetMap contributors',
        maxZoom: 19,
    }).addTo(state.map);

    state.markersLayer = L.layerGroup().addTo(state.map);


    state.map.on('click', function(e) {
        const lat = e.latlng.lat;
        const lng = e.latlng.lng;

        if (document.getElementById('placeModal').classList.contains('active')) {
            state.selectedLat = e.latlng.lat;
            state.selectedLng = e.latlng.lng;
            document.getElementById('selectedLat').textContent = state.selectedLat.toFixed(6);
            document.getElementById('selectedLng').textContent = state.selectedLng.toFixed(6);

            if (state.selectedMarker) {
                state.map.removeLayer(state.selectedMarker);
            }
            state.selectedMarker = L.circleMarker([state.selectedLat, state.selectedLng], {
                radius: 12,
                color: '#4F46E5',
                fillColor: '#4F46E5',
                fillOpacity: 0.3,
                weight: 3,
            }).addTo(state.map);
        }else {
            showAddPlacePopup(lat, lng);
            //openPlaceModal(null)
        }
    });

    state.map.on('dblclick', function(e) {
        const lat = e.latlng.lat;
        const lng = e.latlng.lng;
        findNearby(lat, lng);
    });

    setTimeout(() => {
        state.map.invalidateSize();
    }, 100);
}

function showAddPlacePopup(lat, lng) {
    console.log('📍 Showing add place popup at:', lat, lng);

    // Close any existing popup first
    state.map.closePopup();

    // Create the popup
    const popup = L.popup({
        maxWidth: 300,
        className: 'add-place-popup',
        closeButton: true
    })
        .setLatLng([lat, lng])
        .setContent(`
        <div style="text-align:center;padding:8px;">
            <p style="margin:8px 0;font-weight:600;">No place found here</p>
            <button onclick="openPlaceModalWithLocation(${lat}, ${lng})" 
                    style="padding:6px 16px;background:#4F46E5;color:white;border:none;border-radius:4px;cursor:pointer;">
                Add Place Here
            </button>
        </div>
    `)
        .openOn(state.map);

    // Store reference to popup (optional)
    state.currentPopup = popup;
}

function openPlaceModalWithLocation(lat, lng) {
    // Close the popup
    if (state.currentPopup) {
        state.map.closePopup(state.currentPopup);
        state.currentPopup = null;
    }

    // Set coordinates and open modal
    state.selectedLat = lat;
    state.selectedLng = lng;
    document.getElementById('selectedLat').textContent = lat.toFixed(6);
    document.getElementById('selectedLng').textContent = lng.toFixed(6);

    openPlaceModal(null);
}

function updateMap() {
    if (!state.map) return;

    state.markersLayer.clearLayers();

    state.places.forEach(place => {
        const marker = L.circleMarker([place.latitude, place.longitude], {
            radius: 10,
            fillColor: getCategoryColor(place.category),
            color: '#fff',
            weight: 2,
            opacity: 1,
            fillOpacity: 0.9,
        }).addTo(state.markersLayer);

        marker.bindPopup(`
            <div style="min-width:200px;">
                <h3 style="margin:0 0 4px 0;">${escapeHtml(place.name)}</h3>
                <span style="display:inline-block;font-size:11px;background:#e5e7eb;padding:2px 10px;border-radius:12px;margin-bottom:6px;">
                    ${escapeHtml(place.category)}
                </span>
                ${place.description ? `<p style="font-size:13px;color:#6b7280;margin:4px 0;">${escapeHtml(place.description)}</p>` : ''}
                <button onclick="viewPlace(${place.id})" style="margin-top:8px;padding:4px 12px;background:#4F46E5;color:white;border:none;border-radius:4px;cursor:pointer;font-size:12px;">
                    View Details
                </button>
            </div>
        `);
    });

    if (state.places.length > 0) {
        const bounds = state.places.map(p => [p.latitude, p.longitude]);
        state.map.fitBounds(bounds, { padding: [50, 50] });
    }
}

function updateMapFiltered(places) {
    if (!state.map) return;
    state.markersLayer.clearLayers();

    places.forEach(place => {
        const marker = L.circleMarker([place.latitude, place.longitude], {
            radius: 10,
            fillColor: getCategoryColor(place.category),
            color: '#fff',
            weight: 2,
            opacity: 1,
            fillOpacity: 0.9,
        }).addTo(state.markersLayer);
    });
}

// ============================================
// PLACE CRUD
// ============================================

async function loadPlaces() {
    try {
        const data = await apiRequest('/places');
        state.places = data;
        renderPlaces(data);
        updateMap();
    } catch (error) {
        console.error('Failed to load places:', error);
        showToast('Failed to load places', 'error');
    }
}

async function createPlace(data) {
    try {
        await apiRequest('/places', {
            method: 'POST',
            body: JSON.stringify(data),
        });
        showToast('Place created successfully!', 'success');
        closePlaceModal();
        loadPlaces();
    } catch (error) {
        showToast(error.message, 'error');
    }
}

async function updatePlace(id, data) {
    try {
        await apiRequest(`/places/${id}`, {
            method: 'PUT',
            body: JSON.stringify(data),
        });
        showToast('Place updated successfully!', 'success');
        closePlaceModal();
        loadPlaces();
    } catch (error) {
        showToast(error.message, 'error');
    }
}

async function deletePlace(id) {
    if (!confirm('Are you sure you want to delete this place?')) return;

    try {
        await apiRequest(`/places/${id}`, {
            method: 'DELETE',
        });
        showToast('Place deleted successfully!', 'success');
        loadPlaces();
    } catch (error) {
        showToast(error.message, 'error');
    }
}

async function viewPlace(id) {
    try {
        const place = await apiRequest(`/places/${id}`);
        state.map.setView([place.latitude, place.longitude], 16);

        const content = `
            <div style="max-width:300px;">
                <h2 style="margin:0 0 8px 0;">${escapeHtml(place.name)}</h2>
                <span style="display:inline-block;font-size:12px;background:#e5e7eb;padding:2px 12px;border-radius:12px;margin-bottom:8px;">
                    ${escapeHtml(place.category)}
                </span>
                ${place.description ? `<p style="color:#6b7280;margin:8px 0;">${escapeHtml(place.description)}</p>` : ''}
                ${place.address ? `<p style="font-size:13px;color:#6b7280;"><i class="fas fa-location-dot"></i> ${escapeHtml(place.address)}</p>` : ''}
                <div style="display:flex;gap:8px;margin-top:12px;padding-top:12px;border-top:1px solid #e5e7eb;">
                    <button onclick="editPlace(${place.id})" style="padding:6px 16px;background:#4F46E5;color:white;border:none;border-radius:6px;cursor:pointer;">
                        Edit
                    </button>
                    <button onclick="ratePlace(${place.id})" style="padding:6px 16px;background:#F59E0B;color:white;border:none;border-radius:6px;cursor:pointer;">
                        Rate
                    </button>
                    <button onclick="uploadPhotos(${place.id})" style="padding:6px 16px;background:#2563EB;color:white;border:none;border-radius:6px;cursor:pointer;">
                        Photos
                    </button>
                </div>
            </div>
        `;

        L.popup()
            .setLatLng([place.latitude, place.longitude])
            .setContent(content)
            .openOn(state.map);

    } catch (error) {
        showToast(error.message, 'error');
    }
}

// ============================================
// RATINGS
// ============================================

async function ratePlace(placeId) {
    state.currentPlaceId = placeId;
    state.ratingValue = 0;
    document.getElementById('ratingModal').classList.add('active');
    document.querySelectorAll('.star-rating i').forEach(star => {
        star.classList.remove('active');
    });
    document.getElementById('reviewText').value = '';
}

document.getElementById('submitRatingBtn').addEventListener('click', async () => {
    if (state.ratingValue === 0) {
        showToast('Please select a rating', 'error');
        return;
    }

    try {
        await apiRequest(`/places/${state.currentPlaceId}/rate`, {
            method: 'POST',
            body: JSON.stringify({
                rating: state.ratingValue,
                review: document.getElementById('reviewText').value,
            }),
        });
        showToast('Rating submitted!', 'success');
        document.getElementById('ratingModal').classList.remove('active');
        loadPlaces();
    } catch (error) {
        showToast(error.message, 'error');
    }
});

document.getElementById('closeRatingModal').addEventListener('click', () => {
    document.getElementById('ratingModal').classList.remove('active');
});

// Star rating
document.querySelectorAll('.star-rating i').forEach(star => {
    star.addEventListener('click', function() {
        state.ratingValue = parseInt(this.dataset.rating);
        document.querySelectorAll('.star-rating i').forEach(s => {
            s.classList.toggle('active', parseInt(s.dataset.rating) <= state.ratingValue);
        });
    });
});

// ============================================
// PHOTOS
// ============================================

let selectedFiles = [];

async function uploadPhotos(placeId) {
    state.currentPlaceId = placeId;
    selectedFiles = [];
    document.getElementById('photoInput').value = '';
    document.getElementById('photoPreview').innerHTML = '';
    document.getElementById('photoModal').classList.add('active');
}

document.getElementById('photoInput').addEventListener('change', function() {
    selectedFiles = Array.from(this.files);
    const preview = document.getElementById('photoPreview');
    preview.innerHTML = '';
    selectedFiles.forEach(file => {
        const reader = new FileReader();
        reader.onload = function(e) {
            const img = document.createElement('img');
            img.src = e.target.result;
            preview.appendChild(img);
        };
        reader.readAsDataURL(file);
    });
});

document.getElementById('uploadPhotosBtn').addEventListener('click', async () => {
    if (selectedFiles.length === 0) {
        showToast('Please select photos to upload', 'error');
        return;
    }

    try {
        const formData = new FormData();
        selectedFiles.forEach(file => {
            formData.append('photo', file);
        });

        const response = await fetch(`${API_BASE}/places/${state.currentPlaceId}/photos`, {
            method: 'POST',
            headers: {
                'Authorization': `Bearer ${state.token}`,
            },
            body: formData,
        });

        if (!response.ok) {
            const error = await response.json();
            throw new Error(error.error || 'Upload failed');
        }

        showToast('Photos uploaded successfully!', 'success');
        document.getElementById('photoModal').classList.remove('active');
        loadPlaces();
    } catch (error) {
        showToast(error.message, 'error');
    }
});

document.getElementById('closePhotoModal').addEventListener('click', () => {
    document.getElementById('photoModal').classList.remove('active');
});

// ============================================
// NEARBY
// ============================================

async function findNearby(lat, lon, radius = 5) {
    try {
        const data = await apiRequest(`/places/nearby?lat=${lat}&lon=${lon}&radius=${radius}`);
        state.places = data;
        renderPlaces(data);
        updateMap();
        showToast(`Found ${data.length} nearby places`, 'success');
    } catch (error) {
        showToast(error.message, 'error');
    }
}

// ============================================
// UI HELPERS
// ============================================

function renderPlaces(places) {
    const container = document.getElementById('place-list');

    if (places.length === 0) {
        container.innerHTML = `
            <div class="no-results">
                <i class="fas fa-map-pin" style="font-size:2rem;display:block;margin-bottom:12px;color:#d1d5db;"></i>
                <p>No places yet. Click "Add Place" to get started!</p>
            </div>
        `;
        return;
    }

    container.innerHTML = places.map(place => `
        <div class="place-item" onclick="viewPlace(${place.id})">
            <div class="place-header">
                <h3>${escapeHtml(place.name)}</h3>
                <span class="category-tag ${place.category}">${escapeHtml(place.category)}</span>
            </div>
            ${place.description ? `<div class="place-description">${escapeHtml(place.description)}</div>` : ''}
            <div class="place-meta">
                ${place.created_by ? `<div class="place-creator">👤 ${place.created_by}</div>` : ''}
                ${place.is_owner ? `<span class="owner-badge">Your Place</span>` : ''}
                ${place.address ? `<span><i class="fas fa-location-dot"></i> ${escapeHtml(place.address)}</span>` : ''}
                ${place.phone ? `<span><i class="fas fa-phone"></i> ${escapeHtml(place.phone)}</span>` : ''}
            </div>
            <div class="place-actions">
                <button class="edit-btn" onclick="event.stopPropagation();editPlace(${place.id})">
                    <i class="fas fa-edit"></i> Edit
                </button>
                <button class="rate-btn" onclick="event.stopPropagation();ratePlace(${place.id})">
                    <i class="fas fa-star"></i> Rate
                </button>
                <button class="photo-btn" onclick="event.stopPropagation();uploadPhotos(${place.id})">
                    <i class="fas fa-image"></i> Photos
                </button>
                <button class="delete-btn" onclick="event.stopPropagation();deletePlace(${place.id})">
                    <i class="fas fa-trash"></i>
                </button>
            </div>
        </div>
    `).join('');
}

function escapeHtml(text) {
    if (!text) return '';
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

function getCategoryColor(category) {
    const colors = {
        restaurant: '#DC2626',
        cafe: '#D97706',
        park: '#059669',
        landmark: '#2563EB',
        museum: '#4F46E5',
        shopping: '#DC2626',
        hotel: '#7C3AED',
        other: '#6B7280',
    };
    return colors[category] || '#6B7280';
}

function showToast(message, type = 'success') {
    const colors = {
        success: '#10B981',
        error: '#EF4444',
        warning: '#F59E0B',
        info: '#3B82F6',
    };

    const toast = document.createElement('div');
    toast.style.cssText = `
        position: fixed;
        bottom: 20px;
        right: 20px;
        padding: 12px 24px;
        background: ${colors[type] || '#6B7280'};
        color: white;
        border-radius: 8px;
        font-size: 0.95rem;
        font-weight: 500;
        box-shadow: 0 4px 6px -1px rgba(0,0,0,0.1);
        z-index: 9999;
        animation: slideIn 0.3s ease;
        max-width: 400px;
    `;
    toast.textContent = message;
    document.body.appendChild(toast);

    setTimeout(() => {
        toast.style.opacity = '0';
        toast.style.transition = 'opacity 0.3s ease';
        setTimeout(() => toast.remove(), 300);
    }, 3000);
}

// ============================================
// PLACE MODAL
// ============================================

function openPlaceModal(place = null) {
    state.editingPlaceId = place ? place.id : null;
    document.getElementById('placeModalTitle').textContent = place ? 'Edit Place' : 'Add New Place';

    if (place) {
        document.getElementById('placeName').value = place.name;
        document.getElementById('placeCategory').value = place.category;
        document.getElementById('placeDescription').value = place.description || '';
        document.getElementById('placeAddress').value = place.address || '';
        document.getElementById('placePhone').value = place.phone || '';
        document.getElementById('placeWebsite').value = place.website || '';
        state.selectedLat = place.latitude;
        state.selectedLng = place.longitude;
        document.getElementById('selectedLat').textContent = place.latitude.toFixed(6);
        document.getElementById('selectedLng').textContent = place.longitude.toFixed(6);

        if (state.selectedMarker) {
            state.map.removeLayer(state.selectedMarker);
        }
        state.selectedMarker = L.circleMarker([place.latitude, place.longitude], {
            radius: 12,
            color: '#4F46E5',
            fillColor: '#4F46E5',
            fillOpacity: 0.3,
            weight: 3,
        }).addTo(state.map);
        state.map.setView([place.latitude, place.longitude], 16);
    } else {
        //document.getElementById('placeForm').reset();
        //document.getElementById('selectedLat').textContent = '-';
        //document.getElementById('selectedLng').textContent = '-';
        //state.selectedLat = null;
        //state.selectedLng = null;
        if (state.selectedMarker) {
            state.map.removeLayer(state.selectedMarker);
            state.selectedMarker = null;
        }
    }

    document.getElementById('placeModal').classList.add('active');
}

function closePlaceModal() {
    document.getElementById('placeModal').classList.remove('active');
    if (state.selectedMarker) {
        state.map.removeLayer(state.selectedMarker);
        state.selectedMarker = null;
    }
}

function editPlace(id) {
    const place = state.places.find(p => p.id === id);
    if (place) {
        openPlaceModal(place);
    }
}

// ============================================
// EVENT LISTENERS
// ============================================

// Auth
document.getElementById('authForm').addEventListener('submit', async (e) => {
    e.preventDefault();
    const username = document.getElementById('loginUsername').value;
    const password = document.getElementById('loginPassword').value;

    if (document.getElementById('authSubmitBtn').textContent === 'Register') {
        const email = document.getElementById('regEmail').value;
        await register(username, email, password);
    } else {
        await login(username, password);
    }
});

document.addEventListener('click', (e) => {
    if (e.target.id === 'switchToRegister') {
        e.preventDefault();
        const isRegister = document.getElementById('authSubmitBtn').textContent === 'Register';
        document.getElementById('authTitle').textContent = isRegister ? 'Login' : 'Register';
        document.getElementById('authSubmitBtn').textContent = isRegister ? 'Login' : 'Register';
        document.getElementById('registerFields').style.display = isRegister ? 'none' : 'block';
        document.getElementById('authSwitch').innerHTML = isRegister
            ? 'Don\'t have an account? <a href="#" id="switchToRegister">Register</a>'
            : 'Already have an account? <a href="#" id="switchToRegister">Login</a>';
        hideAuthError();
    }
});

document.getElementById('logoutBtn').addEventListener('click', logout);

// Places
document.getElementById('addPlaceBtn').addEventListener('click', () => openPlaceModal(null));
document.getElementById('closePlaceModal').addEventListener('click', closePlaceModal);

document.getElementById('placeForm').addEventListener('submit', async (e) => {
    e.preventDefault();

    const data = {
        name: document.getElementById('placeName').value,
        category: document.getElementById('placeCategory').value,
        description: document.getElementById('placeDescription').value,
        address: document.getElementById('placeAddress').value,
        phone: document.getElementById('placePhone').value,
        website: document.getElementById('placeWebsite').value,
        latitude: state.selectedLat,
        longitude: state.selectedLng,
    };

    if (!data.latitude || !data.longitude) {
        showToast('Please click on the map to select a location', 'error');
        return;
    }

    if (state.editingPlaceId) {
        await updatePlace(state.editingPlaceId, data);
    } else {
        await createPlace(data);
    }
});

// Search & Filter
document.getElementById('searchInput').addEventListener('input', (e) => {
    const query = e.target.value.toLowerCase();
    const filtered = state.places.filter(p =>
        p.name.toLowerCase().includes(query) ||
        p.category.toLowerCase().includes(query) ||
        (p.description && p.description.toLowerCase().includes(query))
    );
    renderPlaces(filtered);
    updateMapFiltered(filtered);
});

document.getElementById('categoryFilter').addEventListener('change', (e) => {
    const category = e.target.value;
    const filtered = category === 'all'
        ? state.places
        : state.places.filter(p => p.category === category);
    renderPlaces(filtered);
    updateMapFiltered(filtered);
});

// Nearby
document.getElementById('findNearbyBtn').addEventListener('click', () => {
    if ('geolocation' in navigator) {
        navigator.geolocation.getCurrentPosition(
            (position) => {
                const lat = position.coords.latitude;
                const lon = position.coords.longitude;
                const radius = parseFloat(document.getElementById('radiusInput').value) || 5;
                findNearby(lat, lon, radius);
                state.map.setView([lat, lon], 14);
            },
            () => {
                showToast('Unable to get location. Click on the map or enter coordinates.', 'error');
            }
        );
    } else {
        showToast('Geolocation not supported', 'error');
    }
});

// Close modals on backdrop click
document.querySelectorAll('.modal').forEach(modal => {
    modal.addEventListener('click', (e) => {
        if (e.target === modal) {
            modal.classList.remove('active');
        }
    });
});

// Escape key
document.addEventListener('keydown', (e) => {
    if (e.key === 'Escape') {
        document.querySelectorAll('.modal.active').forEach(modal => {
            modal.classList.remove('active');
        });
    }
});

// ============================================
// LOAD CATEGORIES
// ============================================

async function loadCategories() {
    try {
        const data = await apiRequest('/categories');
        const select = document.getElementById('categoryFilter');
        data.forEach(cat => {
            const option = document.createElement('option');
            option.value = cat;
            option.textContent = cat.charAt(0).toUpperCase() + cat.slice(1);
            select.appendChild(option);
        });
    } catch (error) {
        console.error('Failed to load categories:', error);
    }
}

// ============================================
// INIT
// ============================================

checkAuth();
loadCategories();

console.log('📍 Map Explorer loaded!');