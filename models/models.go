package models

import "time"

// User структура для пользователя
type User struct {
	ID             int
	Username       string
	Password       string
	Code           string
	Balance        float64 // Баланс пользователя
	ProfilePicture []byte  // Фото профиля
}

// Lottery структура для лотереи
type Lottery struct {
	ID          int
	Name        string
	Description string
	Price       float64
	EndDate     time.Time
}

// Draw структура для розыгрыша
type Draw struct {
	ID          int
	LotteryID   int
	DrawDate    time.Time
	Winner      string
	PrizeAmount float64
}
