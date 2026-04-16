package main

import (
	"fitness-app/internal/api"
	"fitness-app/internal/database"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env file - try multiple locations
	if err := godotenv.Load("../.env"); err != nil {
		if err := godotenv.Load("../../.env"); err != nil {
			if err := godotenv.Load(".env"); err != nil {
				log.Println("No .env file found, using environment variables")
			}
		}
	}

	// Initialize database
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "fitness.db"
	}

	if err := database.Initialize(dbPath); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	// Setup router
	r := mux.NewRouter()

	// Apply CORS middleware
	r.Use(api.CORSMiddleware)

	// Public routes
	r.HandleFunc("/api/register", api.Register).Methods("POST")
	r.HandleFunc("/api/login", api.Login).Methods("POST")
	r.HandleFunc("/api/telegram-auth", api.TelegramAuth).Methods("POST")

	// Protected routes
	protected := r.PathPrefix("/api").Subrouter()
	protected.Use(api.AuthMiddleware)

	// Profile
	protected.HandleFunc("/profile", api.GetProfile).Methods("GET")
	protected.HandleFunc("/profile", api.UpdateProfile).Methods("PUT")

	// Activities
	protected.HandleFunc("/activities", api.CreateActivity).Methods("POST")
	protected.HandleFunc("/activities", api.GetActivities).Methods("GET")

	// Challenges
	protected.HandleFunc("/challenges", api.CreateChallenge).Methods("POST")
	protected.HandleFunc("/challenges", api.GetUserChallenges).Methods("GET")
	protected.HandleFunc("/challenges/{id}", api.GetChallenge).Methods("GET")
	protected.HandleFunc("/challenges/{id}/add-participant", api.AddParticipant).Methods("POST")
	protected.HandleFunc("/challenges/{id}/join", api.JoinChallenge).Methods("POST")
	protected.HandleFunc("/challenges/{id}/complete", api.CompleteChallenge).Methods("POST")
	protected.HandleFunc("/challenges/{id}/progress", api.AddProgress).Methods("POST")
	protected.HandleFunc("/challenges/{id}/progress", api.GetChallengeProgress).Methods("GET")
	protected.HandleFunc("/challenges/{id}/logs", api.GetChallengeLogs).Methods("GET")
	protected.HandleFunc("/challenges/join/{code}", api.JoinChallengeByInviteCode).Methods("POST")

	// Achievements
	protected.HandleFunc("/achievements", api.GetAchievements).Methods("GET")

	// Leaderboard
	protected.HandleFunc("/leaderboard", api.GetLeaderboard).Methods("GET")

	// Serve static files
	r.PathPrefix("/").Handler(http.FileServer(http.Dir("../frontend")))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatal(err)
	}
}
