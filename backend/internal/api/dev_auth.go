package api

import (
	"database/sql"
	"encoding/json"
	"fitness-app/internal/auth"
	"fitness-app/internal/database"
	"fitness-app/internal/models"
	"net/http"
	"time"
)

type QuickLoginRequest struct {
	Phone string `json:"phone"`
	Name  string `json:"name"`
}

// QuickLogin - простой логин без OTP для разработки
func QuickLogin(w http.ResponseWriter, r *http.Request) {
	var req QuickLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.Phone == "" {
		http.Error(w, "Phone is required", http.StatusBadRequest)
		return
	}

	// Get or create user
	var user models.User
	var phone, photoURL, description sql.NullString
	var height, weight sql.NullFloat64

	err := database.DB.QueryRow(`
		SELECT id, phone, name, height, weight, photo_url, description,
		       total_wins, current_streak, best_streak, created_at, updated_at
		FROM users WHERE phone = ?
	`, req.Phone).Scan(
		&user.ID, &phone, &user.Name, &height, &weight,
		&photoURL, &description, &user.TotalWins,
		&user.CurrentStreak, &user.BestStreak, &user.CreatedAt, &user.UpdatedAt,
	)

	if err == nil {
		// Handle nullable fields
		if phone.Valid {
			user.Phone = phone.String
		}
		if height.Valid {
			user.Height = height.Float64
		}
		if weight.Valid {
			user.Weight = weight.Float64
		}
		if photoURL.Valid {
			user.PhotoURL = photoURL.String
		}
		if description.Valid {
			user.Description = description.String
		}
	}

	if err != nil {
		// Create new user
		name := req.Name
		if name == "" {
			name = req.Phone
		}

		result, err := database.DB.Exec(`
			INSERT INTO users (phone, name, created_at, updated_at)
			VALUES (?, ?, ?, ?)
		`, req.Phone, name, time.Now(), time.Now())

		if err != nil {
			http.Error(w, "Failed to create user", http.StatusInternalServerError)
			return
		}

		id, _ := result.LastInsertId()
		user.ID = int(id)
		user.Phone = req.Phone
		user.Name = name
		user.CreatedAt = time.Now()
		user.UpdatedAt = time.Now()
	}

	// Generate token
	token, err := auth.GenerateToken(&user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := models.AuthResponse{
		Token: token,
		User:  user,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
