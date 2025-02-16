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

	// Запускаем планировщики
	go handlers.StartDrawScheduler()

	// Статические файлы
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))

	// Маршруты
	http.HandleFunc("/", handlers.HomeHandler)
	http.HandleFunc("/register", handlers.RegisterHandler)
	http.HandleFunc("/login", handlers.LoginHandler)
	http.HandleFunc("/logout", handlers.LogoutHandler)

	// Маршруты лотерей
	http.HandleFunc("/lotteries", handlers.LotteriesHandler)            // Отображение всех лотерей
	http.HandleFunc("/buy", handlers.BuyLotteryHandler)                 // Покупка билетов
	http.HandleFunc("/lotteries/create", handlers.CreateLotteryHandler) // Создание лотереи
	http.HandleFunc("/lotteries/update", handlers.UpdateLotteryHandler) // Обновление лотереи
	http.HandleFunc("/lotteries/delete", handlers.DeleteLotteryHandler) // Удаление лотереи

	// Маршруты профиля и администрирования
	http.HandleFunc("/profile", handlers.ProfileHandler) // Профиль пользователя

	// Маршруты розыгрышей
	http.HandleFunc("/draws", handlers.DrawsHandler)             // Отображение всех розыгрышей
	http.HandleFunc("/draws/update", handlers.UpdateDrawHandler) // Обновление розыгрыша
	http.HandleFunc("/draws/delete", handlers.DeleteDrawHandler) // Удаление розыгрыша
	
	http.HandleFunc("/process-payment", handlers.ProcessPayment)

	// Пополнение баланса и добавление карты
	http.HandleFunc("/add-card", handlers.AddCardHandler) // Страница добавления карты и пополнения баланса

	log.Println("Server started on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
