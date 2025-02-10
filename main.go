// main.go
package main

import (
	"log"
	"loto/db"
	"loto/handlers"
	"net/http"
)

func main() {
	// Инициализируем БД
	db.Init()
	defer db.DB.Close()

	// Инициализируем схему (таблицы)
	if err := db.InitializeSchema(); err != nil {
		log.Fatalf("Failed to initialize schema: %v", err)
	}

	// Пример: розыгрыш победителей для лотереи с ID = 1
	err := db.DrawWinners(1)
	if err != nil {
		log.Printf("Error drawing winners: %v", err)
	}

	// Статические файлы
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))

	// Маршруты
	http.HandleFunc("/", handlers.HomeHandler)
	http.HandleFunc("/register", handlers.RegisterHandler)
	http.HandleFunc("/login", handlers.LoginHandler)
	http.HandleFunc("/logout", handlers.LogoutHandler)

	// Маршруты лотерей
	http.HandleFunc("/lotteries", handlers.LotteriesHandler)
	http.HandleFunc("/buy", handlers.BuyLotteryHandler)
	http.HandleFunc("/lotteries/create", handlers.CreateLotteryHandler)
	http.HandleFunc("/lotteries/update", handlers.UpdateLotteryHandler)
	http.HandleFunc("/lotteries/delete", handlers.DeleteLotteryHandler)


	/////
	http.HandleFunc("/admin", handlers.AdminDashboardHandler)
	http.HandleFunc("/profile", handlers.ProfileHandler)

	// Маршруты розыгрышей
	http.HandleFunc("/draws", handlers.DrawsHandler)
	http.HandleFunc("/draws/create", handlers.CreateDrawHandler)
	http.HandleFunc("/draws/update", handlers.UpdateDrawHandler)
	http.HandleFunc("/draws/delete", handlers.DeleteDrawHandler)

	// Маршрут для добавления платёжной карты

	log.Println("Server started on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
