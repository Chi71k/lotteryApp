package db

import (
	"database/sql"
	"log"

	_ "github.com/lib/pq" 
)

var DB *sql.DB

const (
	dbAdminConnection = "user=loto_user password=1234 dbname=postgres sslmode=disable"
	dbConnection      = "user=loto_user password=1234 dbname=postgres sslmode=disable"
)

func Init() {
	adminDB, err := sql.Open("postgres", dbAdminConnection)
	if err != nil {
		log.Fatalf("Failed to connect to admin database: %v", err)
	}
	defer adminDB.Close()

	if err := createDatabaseIfNotExists(adminDB, "goproject"); err != nil {
		log.Printf("Warning: %v", err)
	}

	DB, err = sql.Open("postgres", dbConnection)
	if err != nil {
		log.Fatalf("Failed to connect to target database: %v", err)
	}

	if err = DB.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	log.Println("Connected to database 'goproject'")
}

func createDatabaseIfNotExists(adminDB *sql.DB, dbName string) error {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)`
	if err := adminDB.QueryRow(query, dbName).Scan(&exists); err != nil {
		return err
	}

	if !exists {
		_, err := adminDB.Exec("CREATE DATABASE " + dbName)
		if err != nil {
			return err
		}
		log.Printf("Database %s created successfully!", dbName)
	} else {
		log.Printf("Database %s already exists!", dbName)
	}
	return nil
}

func InitializeSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id SERIAL PRIMARY KEY,
		username VARCHAR(50) UNIQUE NOT NULL,
		password VARCHAR(255) NOT NULL
	);

	CREATE TABLE IF NOT EXISTS lotteries (
		id SERIAL PRIMARY KEY,
		name VARCHAR(100),
		description TEXT,
		price NUMERIC,
		end_date TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS purchases (
		id SERIAL PRIMARY KEY,
		username VARCHAR(50),
		lottery_id INT REFERENCES lotteries(id)
	);
	`
	_, err := DB.Exec(schema)
	if err != nil {
		return err
	}

	checkQuery := "SELECT COUNT(*) FROM lotteries"
	var count int
	err = DB.QueryRow(checkQuery).Scan(&count)
	if err != nil {
		return err
	}

	if count == 0 {
		log.Println("Adding sample lotteries...")
		insertQuery := `
		INSERT INTO lotteries (name, description, price, end_date) VALUES
		('Mega Jackpot', 'Win the biggest prize in our history!', 100, '2025-12-31'),
		('Holiday Special', 'Celebrate the holidays with amazing prizes!', 50, '2025-11-30'),
		('Weekly Draw', 'Join our weekly draw for exciting rewards!', 20, '2025-10-15');
		`
		_, err := DB.Exec(insertQuery)
		if err != nil {
			return err
		}
		log.Println("Sample lotteries added.")
	}
	return nil
}
