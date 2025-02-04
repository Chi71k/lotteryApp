package main

import (
	"log"
	"loto/db"
	"loto/handlers"
	"net/http"
)

func main() {
	// Initialize DB
	db.Init()
	defer db.DB.Close()

	// Initialize schema
	if err := db.InitializeSchema(); err != nil {
		log.Fatalf("Failed to initialize schema: %v", err)
	}

	err := db.DrawWinners(1)
	if err != nil {
		log.Printf("Error drawing winners: %v", err)
	}

	// Serve static files
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))

	// Routes
	http.HandleFunc("/", handlers.HomeHandler)
	http.HandleFunc("/register", handlers.RegisterHandler)
	http.HandleFunc("/login", handlers.LoginHandler)
	http.HandleFunc("/logout", handlers.LogoutHandler)

	// Lotteries routes
	http.HandleFunc("/lotteries", handlers.LotteriesHandler)
	http.HandleFunc("/buy", handlers.BuyLotteryHandler)
	http.HandleFunc("/lotteries/create", handlers.CreateLotteryHandler)
	http.HandleFunc("/lotteries/update", handlers.UpdateLotteryHandler)
	http.HandleFunc("/lotteries/delete", handlers.DeleteLotteryHandler)

	// Draws routes (existing)
	http.HandleFunc("/draws", handlers.DrawsHandler)
	http.HandleFunc("/draws/create", handlers.CreateDrawHandler)
	http.HandleFunc("/draws/update", handlers.UpdateDrawHandler)
	http.HandleFunc("/draws/delete", handlers.DeleteDrawHandler)

	// Start server
	log.Println("Server started on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
