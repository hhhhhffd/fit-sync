package services

import (
	"fitness-app/internal/database"
	"fitness-app/internal/models"
)

// CheckAndUnlockAchievements checks if user has unlocked any new achievements
func CheckAndUnlockAchievements(userID int) error {
	// Get user stats
	var wins, streak int
	err := database.DB.QueryRow(
		"SELECT total_wins, best_streak FROM users WHERE id = ?",
		userID,
	).Scan(&wins, &streak)
	if err != nil {
		return err
	}

	// Get all achievements
	rows, err := database.DB.Query(`
		SELECT id, name, description, requirement_type, requirement_value, icon
		FROM achievements
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var ach models.Achievement
		if err := rows.Scan(&ach.ID, &ach.Name, &ach.Description,
			&ach.RequirementType, &ach.RequirementValue, &ach.Icon); err != nil {
			continue
		}

		// Check if already unlocked
		var count int
		err := database.DB.QueryRow(
			"SELECT COUNT(*) FROM user_achievements WHERE user_id = ? AND achievement_id = ?",
			userID, ach.ID,
		).Scan(&count)
		if err != nil || count > 0 {
			continue
		}

		// Check if requirements are met
		shouldUnlock := false
		switch ach.RequirementType {
		case "wins":
			shouldUnlock = wins >= ach.RequirementValue
		case "streak":
			shouldUnlock = streak >= ach.RequirementValue
		}

		if shouldUnlock {
			_, err := database.DB.Exec(
				"INSERT INTO user_achievements (user_id, achievement_id) VALUES (?, ?)",
				userID, ach.ID,
			)
			if err != nil {
				continue
			}
		}
	}

	return nil
}

// GetUserAchievements returns all achievements for a user
func GetUserAchievements(userID int) ([]models.UserAchievement, error) {
	rows, err := database.DB.Query(`
		SELECT ua.id, ua.user_id, ua.achievement_id, ua.unlocked_at,
		       a.name, a.description, a.requirement_type, a.requirement_value, a.icon
		FROM user_achievements ua
		JOIN achievements a ON ua.achievement_id = a.id
		WHERE ua.user_id = ?
		ORDER BY ua.unlocked_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var achievements []models.UserAchievement
	for rows.Next() {
		var ua models.UserAchievement
		var ach models.Achievement
		err := rows.Scan(
			&ua.ID, &ua.UserID, &ua.AchievementID, &ua.UnlockedAt,
			&ach.Name, &ach.Description, &ach.RequirementType, &ach.RequirementValue, &ach.Icon,
		)
		if err != nil {
			continue
		}
		ua.Achievement = ach
		achievements = append(achievements, ua)
	}

	return achievements, nil
}
