package models

import "time"

type User struct {
	ID            int       `json:"id"`
	Email         string    `json:"email,omitempty"`
	Login         string    `json:"login"`
	Phone         string    `json:"phone,omitempty"`
	TelegramID    *int64    `json:"telegram_id,omitempty"`
	Name          string    `json:"name"`
	Age           int       `json:"age,omitempty"`
	Height        float64   `json:"height"`
	Weight        float64   `json:"weight"`
	PhotoURL      string    `json:"photo_url"`
	Description   string    `json:"description"`
	TotalWins     int       `json:"total_wins"`
	CurrentStreak int       `json:"current_streak"`
	BestStreak    int       `json:"best_streak"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type Activity struct {
	ID           int       `json:"id"`
	UserID       int       `json:"user_id"`
	ActivityType string    `json:"activity_type"`
	Duration     int       `json:"duration"`
	Distance     float64   `json:"distance"`
	Calories     int       `json:"calories"`
	Notes        string    `json:"notes"`
	ActivityDate time.Time `json:"activity_date"`
	CreatedAt    time.Time `json:"created_at"`
}

type Challenge struct {
	ID              int       `json:"id"`
	CreatorID       int       `json:"creator_id"`
	Title           string    `json:"title"`
	Description     string    `json:"description"`
	Type            string    `json:"type"`                       // accumulative, consistency
	GoalValue       int       `json:"goal_value"`                 // numeric target or days count
	MaxParticipants *int      `json:"max_participants,omitempty"` // NULL = unlimited
	StartDate       time.Time `json:"start_date"`
	EndDate         time.Time `json:"end_date"`
	Status          string    `json:"status"`
	WinnerID        *int      `json:"winner_id,omitempty"`
	InviteCode      string    `json:"invite_code,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	Participants    []User    `json:"participants,omitempty"`
}

type ChallengeParticipant struct {
	ID              int       `json:"id"`
	ChallengeID     int       `json:"challenge_id"`
	UserID          int       `json:"user_id"`
	JoinedAt        time.Time `json:"joined_at"`
	TotalPoints     int       `json:"total_points"`
	CurrentProgress int       `json:"current_progress"`
}

type ChallengeLog struct {
	ID          int       `json:"id"`
	ChallengeID int       `json:"challenge_id"`
	UserID      int       `json:"user_id"`
	Value       int       `json:"value"`
	PhotoFileID string    `json:"photo_file_id,omitempty"`
	Notes       string    `json:"notes,omitempty"`
	LoggedAt    time.Time `json:"logged_at"`
}

type Achievement struct {
	ID               int    `json:"id"`
	Name             string `json:"name"`
	Description      string `json:"description"`
	RequirementType  string `json:"requirement_type"`
	RequirementValue int    `json:"requirement_value"`
	Icon             string `json:"icon"`
}

type UserAchievement struct {
	ID            int         `json:"id"`
	UserID        int         `json:"user_id"`
	AchievementID int         `json:"achievement_id"`
	UnlockedAt    time.Time   `json:"unlocked_at"`
	Achievement   Achievement `json:"achievement,omitempty"`
}

type RegisterRequest struct {
	Email    string  `json:"email"`
	Login    string  `json:"login"`
	Password string  `json:"password"`
	Name     string  `json:"name"`
	Age      int     `json:"age,omitempty"`
	Height   float64 `json:"height,omitempty"`
	Weight   float64 `json:"weight,omitempty"`
	PhotoURL string  `json:"photo_url,omitempty"`
}

type LoginRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type AuthResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

type LeaderboardEntry struct {
	User
	Rank int `json:"rank"`
}

type CreateChallengeRequest struct {
	Title           string    `json:"title"`
	Description     string    `json:"description"`
	Type            string    `json:"type"`                       // accumulative or consistency
	GoalValue       int       `json:"goal_value"`                 // numeric target or days count
	MaxParticipants *int      `json:"max_participants,omitempty"` // NULL = unlimited
	StartDate       time.Time `json:"start_date"`
	EndDate         time.Time `json:"end_date"`
}

type AddProgressRequest struct {
	Value       int    `json:"value"` // for accumulative: added value, for consistency: 1
	PhotoFileID string `json:"photo_file_id,omitempty"`
	Notes       string `json:"notes,omitempty"`
}
