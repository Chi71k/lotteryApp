package main

import (
	"log"
	"loto/db"
	"loto/handlers"
	"net/http"
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

	go handlers.StartDrawScheduler()

	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))

	http.HandleFunc("/", handlers.HomeHandler)
	http.HandleFunc("/login", handlers.LoginHandler)
	http.HandleFunc("/logout", handlers.LogoutHandler)

	http.HandleFunc("/lotteries", handlers.LotteriesHandler)            // Отображение всех лотерей
	http.HandleFunc("/buy", handlers.BuyLotteryHandler)                 // Покупка билетов
	http.HandleFunc("/lotteries/create", handlers.CreateLotteryHandler) // Создание лотереи
	http.HandleFunc("/lotteries/update", handlers.UpdateLotteryHandler) // Обновление лотереи
	http.HandleFunc("/lotteries/delete", handlers.DeleteLotteryHandler) // Удаление лотереи

	http.HandleFunc("/profile", handlers.ProfileHandler) // Профиль пользователя

	http.HandleFunc("/draws", handlers.DrawsHandler)             // Отображение всех розыгрышей
	http.HandleFunc("/draws/update", handlers.UpdateDrawHandler) // Обновление розыгрыша
	http.HandleFunc("/draws/delete", handlers.DeleteDrawHandler) // Удаление розыгрыша
	
	http.HandleFunc("/process-payment", handlers.ProcessPayment)

	http.HandleFunc("/add-card", handlers.AddCardHandler) // Страница добавления карты и пополнения баланса

	http.HandleFunc("/register", handlers.RegisterHandler)


	log.Println("Server started on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
