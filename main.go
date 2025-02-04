package main

import (
	"log"
	"net/http"

	"loto/db"
	"loto/handlers"
)

func main() {
	db.Init()
	defer db.DB.Close()

	if err := db.InitializeSchema(); err != nil {
		log.Fatalf("Failed to initialize schema: %v", err)
	}

	err := db.DrawWinners(1)
	if err != nil {
		log.Printf("Error drawing winners: %v", err)
	}

	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))

	http.HandleFunc("/", handlers.HomeHandler)
	http.HandleFunc("/register", handlers.RegisterHandler)
	http.HandleFunc("/login", handlers.LoginHandler)
	http.HandleFunc("/logout", handlers.LogoutHandler)
	http.HandleFunc("/lotteries", handlers.LotteriesHandler)
	http.HandleFunc("/buy", handlers.BuyLotteryHandler)

	log.Println("Server started on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
