package models

import "time"

type Draw struct {
	ID             int
	WinningNumbers string
	DrawDate       time.Time
}
