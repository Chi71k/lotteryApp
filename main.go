package main

import (
	"log"
	"net/http"

	"loto/db"
	"loto/handlers"
)

func main() {
	// Инициализация БД
	db.Init()
	defer db.DB.Close()

	// Создание таблиц и примерных записей
	if err := db.InitializeSchema(); err != nil {
		log.Fatalf("Failed to initialize schema: %v", err)
	}

	// Раздача статических файлов (CSS, JS)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))

	// Маршруты
	http.HandleFunc("/", handlers.HomeHandler)
	http.HandleFunc("/register", handlers.RegisterHandler)
	http.HandleFunc("/login", handlers.LoginHandler)
	http.HandleFunc("/logout", handlers.LogoutHandler)
	http.HandleFunc("/lotteries", handlers.LotteriesHandler)
	http.HandleFunc("/buy", handlers.BuyLotteryHandler)

	log.Println("Server started on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
