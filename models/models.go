package models

import "time"

type User struct {
    ID       int
    Username string
    Password string
    Code     string
}

type Lottery struct {
    ID          int
    Name        string
    Description string
    Price       float64
    EndDate     time.Time
}
