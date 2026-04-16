package services

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"fmt"
	"log"
	"time"

	"fitness-app/internal/database"
	"fitness-app/internal/models"
)

// generateInviteCode generates a unique invite code
func generateInviteCode() (string, error) {
	b := make([]byte, 8)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b)[:11], nil
}

// CreateChallenge creates a new challenge
func CreateChallenge(userID int, req models.CreateChallengeRequest) (*models.Challenge, error) {
	// Validate challenge type
	if req.Type != "accumulative" && req.Type != "consistency" {
		return nil, fmt.Errorf("invalid challenge type: must be 'accumulative' or 'consistency'")
	}

	// Validate goal value
	if req.GoalValue <= 0 {
		return nil, fmt.Errorf("goal_value must be greater than 0")
	}

	// Use transaction to ensure atomicity
	tx, err := database.DB.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Generate unique invite code
	inviteCode, err := generateInviteCode()
	if err != nil {
		return nil, err
	}

	// Create challenge
	result, err := tx.Exec(`
		INSERT INTO challenges (creator_id, title, description, type, goal_value, max_participants, start_date, end_date, status, invite_code)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, 'pending', ?)
	`, userID, req.Title, req.Description, req.Type, req.GoalValue, req.MaxParticipants, req.StartDate, req.EndDate, inviteCode)
	if err != nil {
		return nil, err
	}

	challengeID, _ := result.LastInsertId()

	// Add creator as participant
	_, err = tx.Exec(`
		INSERT INTO challenge_participants (challenge_id, user_id)
		VALUES (?, ?)
	`, challengeID, userID)
	if err != nil {
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}

	return GetChallenge(int(challengeID))
}

// AddParticipantByLogin adds a user to a challenge by login (only creator can do this)
func AddParticipantByLogin(challengeID, creatorID int, login string) error {
	// Use transaction to prevent race conditions
	tx, err := database.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Check if requester is the creator
	var actualCreatorID int
	var participantCount int
	var status string
	var maxParticipants sql.NullInt64
	err = tx.QueryRow(`
		SELECT c.creator_id, c.status, c.max_participants, COUNT(cp.id)
		FROM challenges c
		LEFT JOIN challenge_participants cp ON c.id = cp.challenge_id
		WHERE c.id = ?
		GROUP BY c.id
	`, challengeID).Scan(&actualCreatorID, &status, &maxParticipants, &participantCount)
	if err != nil {
		return err
	}

	if creatorID != actualCreatorID {
		return fmt.Errorf("only creator can add participants")
	}

	if status != "pending" {
		return fmt.Errorf("challenge already started")
	}

	// Check max participants limit (if set)
	if maxParticipants.Valid && participantCount >= int(maxParticipants.Int64) {
		return fmt.Errorf("challenge is full (%d/%d participants)", participantCount, int(maxParticipants.Int64))
	}

	// Find user by login
	var userID int
	err = tx.QueryRow("SELECT id FROM users WHERE login = ?", login).Scan(&userID)
	if err == sql.ErrNoRows {
		return fmt.Errorf("user not found")
	} else if err != nil {
		return err
	}

	// Check if user already participant
	var existingCount int
	err = tx.QueryRow(`
		SELECT COUNT(*) FROM challenge_participants
		WHERE challenge_id = ? AND user_id = ?
	`, challengeID, userID).Scan(&existingCount)
	if err != nil {
		return err
	}
	if existingCount > 0 {
		return fmt.Errorf("user already in challenge")
	}

	// Add participant
	_, err = tx.Exec(`
		INSERT INTO challenge_participants (challenge_id, user_id)
		VALUES (?, ?)
	`, challengeID, userID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// JoinChallenge adds a user to a challenge
func JoinChallenge(challengeID, userID int) error {
	// Use transaction to prevent race conditions
	tx, err := database.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Get challenge with current participant count
	var participantCount int
	var status string
	var maxParticipants sql.NullInt64
	err = tx.QueryRow(`
		SELECT c.status, c.max_participants, COUNT(cp.id)
		FROM challenges c
		LEFT JOIN challenge_participants cp ON c.id = cp.challenge_id
		WHERE c.id = ?
		GROUP BY c.id
	`, challengeID).Scan(&status, &maxParticipants, &participantCount)
	if err != nil {
		return err
	}

	if status != "pending" {
		return fmt.Errorf("challenge already started")
	}

	// Check max participants limit (if set)
	if maxParticipants.Valid && participantCount >= int(maxParticipants.Int64) {
		return fmt.Errorf("challenge is full (%d/%d participants)", participantCount, int(maxParticipants.Int64))
	}

	// Check if user already joined
	var existingCount int
	err = tx.QueryRow(`
		SELECT COUNT(*) FROM challenge_participants
		WHERE challenge_id = ? AND user_id = ?
	`, challengeID, userID).Scan(&existingCount)
	if err != nil {
		return err
	}
	if existingCount > 0 {
		return fmt.Errorf("already joined this challenge")
	}

	// Add participant
	_, err = tx.Exec(`
		INSERT INTO challenge_participants (challenge_id, user_id)
		VALUES (?, ?)
	`, challengeID, userID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// GetChallenge returns challenge with participants
func GetChallenge(challengeID int) (*models.Challenge, error) {
	var ch models.Challenge
	var winnerID sql.NullInt64
	var maxParticipants sql.NullInt64

	// Get challenge info
	err := database.DB.QueryRow(`
		SELECT id, creator_id, title, description, type, goal_value, max_participants,
		       start_date, end_date, status, winner_id, invite_code, created_at
		FROM challenges WHERE id = ?
	`, challengeID).Scan(
		&ch.ID, &ch.CreatorID, &ch.Title, &ch.Description, &ch.Type, &ch.GoalValue,
		&maxParticipants, &ch.StartDate, &ch.EndDate, &ch.Status,
		&winnerID, &ch.InviteCode, &ch.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	if winnerID.Valid {
		winnerIDInt := int(winnerID.Int64)
		ch.WinnerID = &winnerIDInt
	}

	if maxParticipants.Valid {
		maxPInt := int(maxParticipants.Int64)
		ch.MaxParticipants = &maxPInt
	}

	// Get participants in a single query
	rows, err := database.DB.Query(`
		SELECT u.id, u.email, u.login, u.phone, u.telegram_id, u.name, u.height, u.weight, u.photo_url,
		       u.description, u.total_wins, u.current_streak, u.best_streak
		FROM users u
		JOIN challenge_participants cp ON u.id = cp.user_id
		WHERE cp.challenge_id = ?
	`, challengeID)
	if err != nil {
		log.Printf("Error fetching participants for challenge %d: %v", challengeID, err)
		return &ch, nil
	}
	defer rows.Close()

	for rows.Next() {
		var u models.User
		var email, login, phone, photoURL, description sql.NullString
		var telegramID sql.NullInt64
		var height, weight sql.NullFloat64

		err = rows.Scan(&u.ID, &email, &login, &phone, &telegramID, &u.Name, &height, &weight, &photoURL,
			&description, &u.TotalWins, &u.CurrentStreak, &u.BestStreak)
		if err != nil {
			log.Printf("Error scanning participant row: %v", err)
			continue
		}

		if email.Valid {
			u.Email = email.String
		}
		if login.Valid {
			u.Login = login.String
		}
		if phone.Valid {
			u.Phone = phone.String
		}
		if telegramID.Valid {
			tgID := telegramID.Int64
			u.TelegramID = &tgID
		}
		if height.Valid {
			u.Height = height.Float64
		}
		if weight.Valid {
			u.Weight = weight.Float64
		}
		if photoURL.Valid {
			u.PhotoURL = photoURL.String
		}
		if description.Valid {
			u.Description = description.String
		}

		ch.Participants = append(ch.Participants, u)
	}

	return &ch, nil
}

// CompleteChallengeAndSelectWinner completes a challenge and selects winner
func CompleteChallengeAndSelectWinner(challengeID, winnerID int) error {
	tx, err := database.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Verify winner is a participant
	var isParticipant int
	err = tx.QueryRow(`
		SELECT COUNT(*) FROM challenge_participants
		WHERE challenge_id = ? AND user_id = ?
	`, challengeID, winnerID).Scan(&isParticipant)
	if err != nil {
		return err
	}
	if isParticipant == 0 {
		return fmt.Errorf("winner must be a participant of the challenge")
	}

	// Update challenge
	_, err = tx.Exec(`
		UPDATE challenges SET status = 'completed', winner_id = ? WHERE id = ?
	`, winnerID, challengeID)
	if err != nil {
		return err
	}

	// Update winner stats
	_, err = tx.Exec(`
		UPDATE users
		SET total_wins = total_wins + 1,
		    current_streak = current_streak + 1,
		    best_streak = CASE WHEN current_streak + 1 > best_streak
		                      THEN current_streak + 1
		                      ELSE best_streak END
		WHERE id = ?
	`, winnerID)
	if err != nil {
		return err
	}

	// Reset streak for losers
	_, err = tx.Exec(`
		UPDATE users SET current_streak = 0
		WHERE id IN (
			SELECT user_id FROM challenge_participants
			WHERE challenge_id = ? AND user_id != ?
		)
	`, challengeID, winnerID)
	if err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	// Check achievements for winner
	CheckAndUnlockAchievements(winnerID)

	return nil
}

// GetUserChallenges returns all challenges for a user
func GetUserChallenges(userID int) ([]models.Challenge, error) {
	rows, err := database.DB.Query(`
		SELECT DISTINCT c.id, c.creator_id, c.title, c.description, c.type, c.goal_value,
		       c.max_participants, c.start_date, c.end_date, c.status,
		       c.winner_id, c.invite_code, c.created_at
		FROM challenges c
		JOIN challenge_participants cp ON c.id = cp.challenge_id
		WHERE cp.user_id = ?
		ORDER BY c.created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var challenges []models.Challenge
	for rows.Next() {
		var ch models.Challenge
		var winnerID sql.NullInt64
		var maxParticipants sql.NullInt64
		err := rows.Scan(
			&ch.ID, &ch.CreatorID, &ch.Title, &ch.Description, &ch.Type, &ch.GoalValue,
			&maxParticipants, &ch.StartDate, &ch.EndDate, &ch.Status,
			&winnerID, &ch.InviteCode, &ch.CreatedAt,
		)
		if err != nil {
			continue
		}
		if winnerID.Valid {
			winnerIDInt := int(winnerID.Int64)
			ch.WinnerID = &winnerIDInt
		}
		if maxParticipants.Valid {
			maxPInt := int(maxParticipants.Int64)
			ch.MaxParticipants = &maxPInt
		}
		challenges = append(challenges, ch)
	}

	return challenges, nil
}

// AddProgress adds progress to a challenge for a user
func AddProgress(challengeID, userID int, req models.AddProgressRequest) error {
	tx, err := database.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Get challenge info
	var challengeType string
	var status string
	err = tx.QueryRow(`
		SELECT type, status FROM challenges WHERE id = ?
	`, challengeID).Scan(&challengeType, &status)
	if err != nil {
		return fmt.Errorf("challenge not found")
	}

	// Check if challenge is active
	if status != "active" && status != "pending" {
		return fmt.Errorf("challenge is not active")
	}

	// Check if user is a participant
	var participantID int
	var currentProgress int
	err = tx.QueryRow(`
		SELECT id, current_progress FROM challenge_participants
		WHERE challenge_id = ? AND user_id = ?
	`, challengeID, userID).Scan(&participantID, &currentProgress)
	if err == sql.ErrNoRows {
		return fmt.Errorf("user is not a participant of this challenge")
	} else if err != nil {
		return err
	}

	// Handle different challenge types
	var newProgress int
	var logValue int

	if challengeType == "accumulative" {
		// For accumulative: add the value to current progress
		if req.Value <= 0 {
			return fmt.Errorf("value must be greater than 0 for accumulative challenges")
		}
		newProgress = currentProgress + req.Value
		logValue = req.Value
	} else if challengeType == "consistency" {
		// For consistency: check if already logged today
		var lastLog time.Time
		err = tx.QueryRow(`
			SELECT MAX(logged_at) FROM challenge_logs
			WHERE challenge_id = ? AND user_id = ?
		`, challengeID, userID).Scan(&lastLog)

		// If logged today, don't increment
		today := time.Now().Format("2006-01-02")
		if err == nil && lastLog.Format("2006-01-02") == today {
			return fmt.Errorf("already logged progress today")
		}

		// Increment day count
		newProgress = currentProgress + 1
		logValue = 1
	} else {
		return fmt.Errorf("unknown challenge type")
	}

	// Update participant progress
	_, err = tx.Exec(`
		UPDATE challenge_participants
		SET current_progress = ?
		WHERE id = ?
	`, newProgress, participantID)
	if err != nil {
		return err
	}

	// Insert log entry
	_, err = tx.Exec(`
		INSERT INTO challenge_logs (challenge_id, user_id, value, photo_file_id, notes, logged_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, challengeID, userID, logValue, req.PhotoFileID, req.Notes, time.Now())
	if err != nil {
		return err
	}

	// Check if user reached the goal (auto-win)
	var goalValue int
	err = tx.QueryRow(`SELECT goal_value FROM challenges WHERE id = ?`, challengeID).Scan(&goalValue)
	if err != nil {
		return err
	}

	if newProgress >= goalValue {
		// User reached the goal! Complete challenge with this user as winner
		log.Printf("🏆 User %d reached goal %d/%d in challenge %d - auto-completing", userID, newProgress, goalValue, challengeID)

		// Update challenge status and set winner
		_, err = tx.Exec(`
			UPDATE challenges SET status = 'completed', winner_id = ? WHERE id = ?
		`, userID, challengeID)
		if err != nil {
			return err
		}

		// Update winner statistics
		_, err = tx.Exec(`
			UPDATE users
			SET total_wins = total_wins + 1,
			    current_streak = current_streak + 1,
			    best_streak = CASE WHEN current_streak + 1 > best_streak
			                      THEN current_streak + 1
			                      ELSE best_streak END
			WHERE id = ?
		`, userID)
		if err != nil {
			return err
		}

		// Reset streak for losers
		_, err = tx.Exec(`
			UPDATE users SET current_streak = 0
			WHERE id IN (
				SELECT user_id FROM challenge_participants
				WHERE challenge_id = ? AND user_id != ?
			)
		`, challengeID, userID)
		if err != nil {
			return err
		}

		// Commit transaction
		if err = tx.Commit(); err != nil {
			return err
		}

		// Check achievements for winner (outside transaction)
		CheckAndUnlockAchievements(userID)

		return nil
	}

	return tx.Commit()
}

// GetChallengeProgress returns progress for all participants in a challenge
func GetChallengeProgress(challengeID int) ([]models.ChallengeParticipant, error) {
	rows, err := database.DB.Query(`
		SELECT cp.id, cp.challenge_id, cp.user_id, cp.joined_at, cp.total_points, cp.current_progress
		FROM challenge_participants cp
		WHERE cp.challenge_id = ?
		ORDER BY cp.current_progress DESC
	`, challengeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var participants []models.ChallengeParticipant
	for rows.Next() {
		var p models.ChallengeParticipant
		err := rows.Scan(&p.ID, &p.ChallengeID, &p.UserID, &p.JoinedAt, &p.TotalPoints, &p.CurrentProgress)
		if err != nil {
			log.Printf("Error scanning participant progress: %v", err)
			continue
		}
		participants = append(participants, p)
	}

	return participants, nil
}

// GetChallengeLogs returns progress logs for a challenge
func GetChallengeLogs(challengeID int) ([]models.ChallengeLog, error) {
	rows, err := database.DB.Query(`
		SELECT id, challenge_id, user_id, value, photo_file_id, notes, logged_at
		FROM challenge_logs
		WHERE challenge_id = ?
		ORDER BY logged_at DESC
	`, challengeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []models.ChallengeLog
	for rows.Next() {
		var l models.ChallengeLog
		err := rows.Scan(&l.ID, &l.ChallengeID, &l.UserID, &l.Value, &l.PhotoFileID, &l.Notes, &l.LoggedAt)
		if err != nil {
			log.Printf("Error scanning challenge log: %v", err)
			continue
		}
		logs = append(logs, l)
	}

	return logs, nil
}

// JoinChallengeByInviteCode allows a user to join a challenge using invite code
func JoinChallengeByInviteCode(inviteCode string, userID int) error {
	tx, err := database.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Find challenge by invite code
	var challengeID int
	var status string
	var participantCount int
	var maxParticipants sql.NullInt64
	err = tx.QueryRow(`
		SELECT c.id, c.status, c.max_participants, COUNT(cp.id)
		FROM challenges c
		LEFT JOIN challenge_participants cp ON c.id = cp.challenge_id
		WHERE c.invite_code = ?
		GROUP BY c.id
	`, inviteCode).Scan(&challengeID, &status, &maxParticipants, &participantCount)
	if err == sql.ErrNoRows {
		return fmt.Errorf("invalid invite code")
	} else if err != nil {
		return err
	}

	if status != "pending" {
		return fmt.Errorf("challenge already started")
	}

	// Check max participants limit (if set)
	if maxParticipants.Valid && participantCount >= int(maxParticipants.Int64) {
		return fmt.Errorf("challenge is full (%d/%d participants)", participantCount, int(maxParticipants.Int64))
	}

	// Check if user already joined
	var existingCount int
	err = tx.QueryRow(`
		SELECT COUNT(*) FROM challenge_participants
		WHERE challenge_id = ? AND user_id = ?
	`, challengeID, userID).Scan(&existingCount)
	if err != nil {
		return err
	}
	if existingCount > 0 {
		return fmt.Errorf("already joined this challenge")
	}

	// Add participant
	_, err = tx.Exec(`
		INSERT INTO challenge_participants (challenge_id, user_id)
		VALUES (?, ?)
	`, challengeID, userID)
	if err != nil {
		return err
	}

	return tx.Commit()
}
