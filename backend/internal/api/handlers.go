package api

import (
	"database/sql"
	"encoding/json"
	"fitness-app/internal/auth"
	"fitness-app/internal/database"
	"fitness-app/internal/models"
	"fitness-app/internal/services"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

// Register creates a new user account
func Register(w http.ResponseWriter, r *http.Request) {
	var req models.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Validate input
	if req.Login == "" || req.Password == "" || req.Email == "" || req.Name == "" {
		http.Error(w, "login, password, email and name are required", http.StatusBadRequest)
		return
	}

	user, err := auth.Register(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	token, err := auth.GenerateToken(user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := models.AuthResponse{
		Token: token,
		User:  *user,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Login authenticates user and returns JWT token
func Login(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	user, err := auth.Login(req.Login, req.Password)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	token, err := auth.GenerateToken(user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := models.AuthResponse{
		Token: token,
		User:  *user,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetProfile returns user profile
func GetProfile(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r)

	var user models.User
	var email, phone, photoURL, description sql.NullString
	var telegramID sql.NullInt64
	var age sql.NullInt64
	var height, weight sql.NullFloat64

	err := database.DB.QueryRow(`
		SELECT id, email, login, phone, telegram_id, name, age, height, weight, photo_url, description,
		       total_wins, current_streak, best_streak, created_at, updated_at
		FROM users WHERE id = ?
	`, userID).Scan(
		&user.ID, &email, &user.Login, &phone, &telegramID, &user.Name, &age, &height, &weight,
		&photoURL, &description, &user.TotalWins,
		&user.CurrentStreak, &user.BestStreak, &user.CreatedAt, &user.UpdatedAt,
	)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Handle nullable fields
	if email.Valid {
		user.Email = email.String
	}
	if phone.Valid {
		user.Phone = phone.String
	}
	if telegramID.Valid {
		user.TelegramID = &telegramID.Int64
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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

// UpdateProfile updates user profile
func UpdateProfile(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r)

	var user models.User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	_, err := database.DB.Exec(`
		UPDATE users
		SET name = ?, height = ?, weight = ?, photo_url = ?, description = ?, updated_at = ?
		WHERE id = ?
	`, user.Name, user.Height, user.Weight, user.PhotoURL, user.Description, time.Now(), userID)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Profile updated"})
}

// CreateActivity creates a new activity
func CreateActivity(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r)

	var activity models.Activity
	if err := json.NewDecoder(r.Body).Decode(&activity); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	result, err := database.DB.Exec(`
		INSERT INTO activities (user_id, activity_type, duration, distance, calories, notes, activity_date)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, userID, activity.ActivityType, activity.Duration, activity.Distance, activity.Calories, activity.Notes, activity.ActivityDate)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	id, _ := result.LastInsertId()
	activity.ID = int(id)
	activity.UserID = userID

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(activity)
}

// GetActivities returns user activities with optional time filter and pagination
func GetActivities(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r)
	period := r.URL.Query().Get("period") // week, month, year, all

	// Pagination parameters
	limit := 100 // default limit
	offset := 0
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 && parsedLimit <= 500 {
			limit = parsedLimit
		}
	}
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if parsedOffset, err := strconv.Atoi(offsetStr); err == nil && parsedOffset >= 0 {
			offset = parsedOffset
		}
	}

	var query string
	var args []interface{}
	now := time.Now()

	baseQuery := "SELECT id, user_id, activity_type, duration, distance, calories, notes, activity_date, created_at FROM activities WHERE user_id = ?"

	switch period {
	case "week":
		weekAgo := now.AddDate(0, 0, -7)
		query = baseQuery + " AND activity_date >= ? ORDER BY activity_date DESC LIMIT ? OFFSET ?"
		args = []interface{}{userID, weekAgo, limit, offset}
	case "month":
		monthAgo := now.AddDate(0, -1, 0)
		query = baseQuery + " AND activity_date >= ? ORDER BY activity_date DESC LIMIT ? OFFSET ?"
		args = []interface{}{userID, monthAgo, limit, offset}
	case "year":
		yearAgo := now.AddDate(-1, 0, 0)
		query = baseQuery + " AND activity_date >= ? ORDER BY activity_date DESC LIMIT ? OFFSET ?"
		args = []interface{}{userID, yearAgo, limit, offset}
	default: // all
		query = baseQuery + " ORDER BY activity_date DESC LIMIT ? OFFSET ?"
		args = []interface{}{userID, limit, offset}
	}

	rows, err := database.DB.Query(query, args...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	respondWithActivities(w, rows)
}

func respondWithActivities(w http.ResponseWriter, rows *sql.Rows) {
	var activities []models.Activity
	for rows.Next() {
		var a models.Activity
		err := rows.Scan(&a.ID, &a.UserID, &a.ActivityType, &a.Duration,
			&a.Distance, &a.Calories, &a.Notes, &a.ActivityDate, &a.CreatedAt)
		if err != nil {
			log.Printf("Error scanning activity row: %v", err)
			continue
		}
		activities = append(activities, a)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(activities)
}

// CreateChallenge creates a new challenge
func CreateChallenge(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r)

	var req models.CreateChallengeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	challenge, err := services.CreateChallenge(userID, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(challenge)
}

// AddParticipant adds a user to challenge by login (creator only)
func AddParticipant(w http.ResponseWriter, r *http.Request) {
	creatorID := GetUserID(r)
	vars := mux.Vars(r)
	challengeID, _ := strconv.Atoi(vars["id"])

	var req struct {
		Login string `json:"login"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.Login == "" {
		http.Error(w, "login is required", http.StatusBadRequest)
		return
	}

	err := services.AddParticipantByLogin(challengeID, creatorID, req.Login)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Participant added"})
}

// JoinChallenge adds user to a challenge
func JoinChallenge(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r)
	vars := mux.Vars(r)
	challengeID, _ := strconv.Atoi(vars["id"])

	err := services.JoinChallenge(challengeID, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Joined challenge"})
}

// GetChallenge returns challenge details
func GetChallenge(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	challengeID, _ := strconv.Atoi(vars["id"])

	challenge, err := services.GetChallenge(challengeID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(challenge)
}

// GetUserChallenges returns all challenges for user
func GetUserChallenges(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r)

	challenges, err := services.GetUserChallenges(userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(challenges)
}

// CompleteChallenge marks challenge as completed
func CompleteChallenge(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r)
	vars := mux.Vars(r)
	challengeID, _ := strconv.Atoi(vars["id"])

	var req struct {
		WinnerID int `json:"winner_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Check if user is the creator of the challenge
	var creatorID int
	err := database.DB.QueryRow("SELECT creator_id FROM challenges WHERE id = ?", challengeID).Scan(&creatorID)
	if err != nil {
		http.Error(w, "Challenge not found", http.StatusNotFound)
		return
	}

	if userID != creatorID {
		http.Error(w, "Only the creator can complete the challenge", http.StatusForbidden)
		return
	}

	err = services.CompleteChallengeAndSelectWinner(challengeID, req.WinnerID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Challenge completed"})
}

// GetAchievements returns user achievements
func GetAchievements(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r)

	achievements, err := services.GetUserAchievements(userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(achievements)
}

// GetLeaderboard returns leaderboard by type
func GetLeaderboard(w http.ResponseWriter, r *http.Request) {
	leaderboardType := r.URL.Query().Get("type") // wins, streak

	var query string
	switch leaderboardType {
	case "wins":
		query = `SELECT id, email, login, phone, telegram_id, name, age, height, weight, photo_url, description,
		         total_wins, current_streak, best_streak, created_at, updated_at
		         FROM users ORDER BY total_wins DESC LIMIT 100`
	case "streak":
		query = `SELECT id, email, login, phone, telegram_id, name, age, height, weight, photo_url, description,
		         total_wins, current_streak, best_streak, created_at, updated_at
		         FROM users ORDER BY best_streak DESC LIMIT 100`
	default:
		query = `SELECT id, email, login, phone, telegram_id, name, age, height, weight, photo_url, description,
		         total_wins, current_streak, best_streak, created_at, updated_at
		         FROM users ORDER BY total_wins DESC LIMIT 100`
	}

	rows, err := database.DB.Query(query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var leaderboard []models.LeaderboardEntry
	rank := 1
	for rows.Next() {
		var entry models.LeaderboardEntry
		var email, phone, photoURL, description sql.NullString
		var telegramID sql.NullInt64
		var age sql.NullInt64
		var height, weight sql.NullFloat64

		err := rows.Scan(
			&entry.ID, &email, &entry.Login, &phone, &telegramID, &entry.Name, &age, &height, &weight,
			&photoURL, &description, &entry.TotalWins,
			&entry.CurrentStreak, &entry.BestStreak, &entry.CreatedAt, &entry.UpdatedAt,
		)
		if err != nil {
			log.Printf("Error scanning leaderboard row: %v", err)
			continue
		}

		// Handle nullable fields
		if email.Valid {
			entry.Email = email.String
		}
		if phone.Valid {
			entry.Phone = phone.String
		}
		if telegramID.Valid {
			entry.TelegramID = &telegramID.Int64
		}
		if age.Valid {
			entry.Age = int(age.Int64)
		}
		if height.Valid {
			entry.Height = height.Float64
		}
		if weight.Valid {
			entry.Weight = weight.Float64
		}
		if photoURL.Valid {
			entry.PhotoURL = photoURL.String
		}
		if description.Valid {
			entry.Description = description.String
		}

		entry.Rank = rank
		leaderboard = append(leaderboard, entry)
		rank++
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(leaderboard)
}

// AddProgress adds progress to a challenge
func AddProgress(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r)
	vars := mux.Vars(r)
	challengeID, _ := strconv.Atoi(vars["id"])

	var req models.AddProgressRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err := services.AddProgress(challengeID, userID, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Progress added"})
}

// GetChallengeProgress returns progress for all participants in a challenge
func GetChallengeProgress(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	challengeID, _ := strconv.Atoi(vars["id"])

	progress, err := services.GetChallengeProgress(challengeID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(progress)
}

// GetChallengeLogs returns progress logs for a challenge
func GetChallengeLogs(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	challengeID, _ := strconv.Atoi(vars["id"])

	logs, err := services.GetChallengeLogs(challengeID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(logs)
}

// JoinChallengeByInviteCode allows user to join challenge using invite code
func JoinChallengeByInviteCode(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r)
	vars := mux.Vars(r)
	inviteCode := vars["code"]

	err := services.JoinChallengeByInviteCode(inviteCode, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Joined challenge"})
}
