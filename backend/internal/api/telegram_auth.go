package api

import (
	"database/sql"
	"encoding/json"
	"fitness-app/internal/auth"
	"fitness-app/internal/database"
	"fitness-app/internal/models"
	"net/http"
	"strconv"
	"time"
)

type TelegramAuthRequest struct {
	TelegramID int64  `json:"telegram_id"`
	FirstName  string `json:"first_name"`
	LastName   string `json:"last_name"`
	Username   string `json:"username"`
	InitData   string `json:"init_data"`
}

// TelegramAuth authenticates user via Telegram Web App data
func TelegramAuth(w http.ResponseWriter, r *http.Request) {
	var req TelegramAuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Get or create user by telegram_id
	var user models.User
	var name, photoURL, description sql.NullString
	var dbTelegramID sql.NullInt64
	var phone sql.NullString
	var login, email sql.NullString
	var age sql.NullInt64
	var height, weight sql.NullFloat64

	err := database.DB.QueryRow(`
		SELECT id, email, login, phone, telegram_id, name, age, height, weight, photo_url, description,
		       total_wins, current_streak, best_streak, created_at, updated_at
		FROM users WHERE telegram_id = ?
	`, req.TelegramID).Scan(
		&user.ID, &email, &login, &phone, &dbTelegramID, &name, &age, &height, &weight,
		&photoURL, &description, &user.TotalWins,
		&user.CurrentStreak, &user.BestStreak, &user.CreatedAt, &user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		// Create new user with telegram_id
		userName := req.FirstName
		if req.LastName != "" {
			userName += " " + req.LastName
		}
		if userName == "" {
			userName = "User"
		}

		// Generate login from telegram username or ID
		generatedLogin := req.Username
		if generatedLogin == "" {
			generatedLogin = userName
		}
		if generatedLogin == "" {
			// Fallback to telegram_id if no username and no name
			generatedLogin = "tg_" + strconv.FormatInt(req.TelegramID, 10)
		}

		// Check if login already exists and make it unique if needed
		var existingID int
		err = database.DB.QueryRow(`SELECT id FROM users WHERE login = ?`, generatedLogin).Scan(&existingID)
		if err == nil {
			// Login exists, append telegram_id to make it unique
			generatedLogin = generatedLogin + "_" + strconv.FormatInt(req.TelegramID, 10)
		}

		result, err := database.DB.Exec(`
			INSERT INTO users (telegram_id, login, name, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?)
		`, req.TelegramID, generatedLogin, userName, time.Now(), time.Now())

		if err != nil {
			http.Error(w, "Failed to create user", http.StatusInternalServerError)
			return
		}

		id, _ := result.LastInsertId()
		user.ID = int(id)
		user.Login = generatedLogin
		user.TelegramID = &req.TelegramID
		user.Name = userName
		user.CreatedAt = time.Now()
		user.UpdatedAt = time.Now()
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	} else {
		// User found
		if email.Valid {
			user.Email = email.String
		}
		if login.Valid {
			user.Login = login.String
		}
		if phone.Valid {
			user.Phone = phone.String
		}
		if dbTelegramID.Valid {
			user.TelegramID = &dbTelegramID.Int64
		}
		if name.Valid {
			user.Name = name.String
		}
		if age.Valid {
			user.Age = int(age.Int64)
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
