package auth

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	"fitness-app/internal/database"
	"fitness-app/internal/models"

	"github.com/golang-jwt/jwt/v5"
)

func getJWTSecret() []byte {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "your-secret-key-change-in-production"
	}
	return []byte(secret)
}

var jwtSecret = getJWTSecret()

type Claims struct {
	UserID int    `json:"user_id"`
	Login  string `json:"login"`
	jwt.RegisteredClaims
}

// Register creates a new user
func Register(req models.RegisterRequest) (*models.User, error) {
	// Insert user with plain password
	result, err := database.DB.Exec(`
		INSERT INTO users (email, login, password_hash, name, age, height, weight, photo_url, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, req.Email, req.Login, req.Password, req.Name, req.Age, req.Height, req.Weight, req.PhotoURL, time.Now(), time.Now())

	if err != nil {
		return nil, fmt.Errorf("failed to create user: %v", err)
	}

	id, _ := result.LastInsertId()

	user := &models.User{
		ID:        int(id),
		Email:     req.Email,
		Login:     req.Login,
		Name:      req.Name,
		Age:       req.Age,
		Height:    req.Height,
		Weight:    req.Weight,
		PhotoURL:  req.PhotoURL,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	return user, nil
}

// Login authenticates user with login/password
func Login(login, password string) (*models.User, error) {
	var user models.User
	var storedPassword string
	var email, phone, photoURL, description sql.NullString
	var telegramID sql.NullInt64
	var age sql.NullInt64
	var height, weight sql.NullFloat64

	err := database.DB.QueryRow(`
		SELECT id, email, login, password_hash, phone, telegram_id, name, age, height, weight, photo_url, description,
		       total_wins, current_streak, best_streak, created_at, updated_at
		FROM users WHERE login = ?
	`, login).Scan(
		&user.ID, &email, &user.Login, &storedPassword, &phone, &telegramID, &user.Name, &age, &height, &weight,
		&photoURL, &description, &user.TotalWins,
		&user.CurrentStreak, &user.BestStreak, &user.CreatedAt, &user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("invalid login or password")
	} else if err != nil {
		return nil, err
	}

	// Check password (plain comparison)
	if password != storedPassword {
		return nil, fmt.Errorf("invalid login or password")
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

	return &user, nil
}

// GenerateToken creates a JWT token for the user
func GenerateToken(user *models.User) (string, error) {
	claims := &Claims{
		UserID: user.ID,
		Login:  user.Login,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour * 30)), // 30 days
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

// ValidateToken validates JWT token and returns claims
func ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}
