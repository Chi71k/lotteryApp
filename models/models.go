// models/models.go
package models

import "time"

type User struct {
	ID             int
	Username       string
	Password       string
	Code           string // если есть
	Balance        float64
	ProfilePicture []byte
   }
   

type Lottery struct {
	ID               int
	Name             string
	Description      string
	Price            float64
	EndDate          time.Time
	TicketLimit      int
	TicketsTableName string
}

type Draw struct {
	ID          int
	LotteryID   int
	DrawDate    time.Time
	Winner      string
	PrizeAmount float64
}

type PaymentCard struct {
	ID         int
	UserID     int
	CardNumber string
}
