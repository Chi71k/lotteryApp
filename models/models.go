package models

import "time"

// Структура пользователя
type User struct {
    ID       int
    Username string
    Password string
    Code     string
}

// Структура лотереи
type Lottery struct {
    ID          int
    Name        string
    Description string
    Price       float64
    EndDate     time.Time
}
