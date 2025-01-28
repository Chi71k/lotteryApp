package config

import (
	"database/sql"
	"log"

	_ "github.com/lib/pq"
)

var DB *sql.DB

func InitDB() {
	dsn := "postgres://postgres:1234@localhost:5432/lottery?sslmode=disable"

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatal("Error opening DB:", err)
	}
	if err = db.Ping(); err != nil {
		log.Fatal("Error connecting to DB:", err)
	}

	DB = db

	createTables()
}

func createTables() {
	userTable := `
    CREATE TABLE IF NOT EXISTS users (
        id SERIAL PRIMARY KEY,
        name VARCHAR(100),
        email VARCHAR(100) UNIQUE,
        password_hash VARCHAR(255)
    );
    `
	ticketTable := `
    CREATE TABLE IF NOT EXISTS tickets (
        id SERIAL PRIMARY KEY,
        user_id INT REFERENCES users(id),
        numbers VARCHAR(100),
        status VARCHAR(50)
    );
    `
	drawTable := `
    CREATE TABLE IF NOT EXISTS draws (
        id SERIAL PRIMARY KEY,
        winning_numbers VARCHAR(100),
        draw_date TIMESTAMP
    );
    `

	_, err := DB.Exec(userTable)
	if err != nil {
		log.Println("Error creating users table:", err)
	}
	_, err = DB.Exec(ticketTable)
	if err != nil {
		log.Println("Error creating tickets table:", err)
	}
	_, err = DB.Exec(drawTable)
	if err != nil {
		log.Println("Error creating draws table:", err)
	}
}
