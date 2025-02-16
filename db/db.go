package db

import (
	"database/sql"
	"log"

	_ "github.com/lib/pq"
)

var DB *sql.DB

const (
	dbAdminConnection = "user=loto_user password=1234 dbname=postgres sslmode=disable"
	dbConnection      = "user=loto_user password=1234 dbname=goproject sslmode=disable"
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

	// Initialize schema after successful connection
	if err := InitializeSchema(); err != nil {
		log.Fatalf("Failed to initialize database schema: %v", err)
	}
}

func createDatabaseIfNotExists(adminDB *sql.DB, dbName string) error {
	var exists bool
	query := "SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)"
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
		password VARCHAR(255) NOT NULL,
		balance NUMERIC DEFAULT 0,
		profile_picture BYTEA
	);

	CREATE TABLE IF NOT EXISTS lotteries (
		id SERIAL PRIMARY KEY,
		name VARCHAR(100),
		description TEXT,
		price NUMERIC,
		end_date TIMESTAMP,
		status VARCHAR(20) DEFAULT 'active'
	);

	CREATE TABLE IF NOT EXISTS purchases (
		id SERIAL PRIMARY KEY,
		username VARCHAR(50),
		lottery_id INT,
		chosen_numbers VARCHAR(100),
		card_number VARCHAR(20),
		purchase_time TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS winning_tickets (
		id SERIAL PRIMARY KEY,
		purchase_id INT REFERENCES purchases(id),
		winning_amount NUMERIC NOT NULL
	);

	CREATE TABLE IF NOT EXISTS draws (
		id SERIAL PRIMARY KEY,
		lottery_id INT REFERENCES lotteries(id),
		draw_date TIMESTAMP NOT NULL,
		winning_numbers VARCHAR(100),
		winner VARCHAR(100),
		prize_amount NUMERIC
	);

	CREATE TABLE IF NOT EXISTS lottery_analysis (
		lottery_id INT PRIMARY KEY,
		total_sales INT,
		remaining_tickets INT,
		winners_count INT,
		total_revenue NUMERIC,
		sponsor_share NUMERIC,
		charity_share NUMERIC
	);
	`
	_, err := DB.Exec(schema)
	if err != nil {
		return err
	}

	// Check if there are any lotteries in the database
	checkQuery := "SELECT COUNT(*) FROM lotteries"
	var count int
	err = DB.QueryRow(checkQuery).Scan(&count)
	if err != nil {
		return err
	}

	// If no lotteries, insert sample ones
	if count == 0 {
		log.Println("Adding sample lotteries...")
		insertQuery := `
		INSERT INTO lotteries (name, description, price, end_date) VALUES
		('Mega Jackpot', 'Win the biggest prize in our history!', 100, '2025-12-31'),
		('Holiday Special', 'Celebrate the holidays with amazing prizes!', 50, '2025-11-30'),
		('Weekly Draw', 'Join our weekly draw for exciting rewards!', 20, '2025-10-15');`

		_, err := DB.Exec(insertQuery)
		if err != nil {
			return err
		}
		log.Println("Sample lotteries added.")
	}
	return nil
}

// Function for purchasing tickets
func PurchaseTicket(username string, lotteryID int, ticketsCount int, cardNumber string) error {
	tx, err := DB.Begin()
	if err != nil {
		log.Printf("Error starting transaction: %v", err)
		return err
	}

	// Insert the purchase
	query := "INSERT INTO purchases (username, lottery_id, tickets_count, card_number, purchase_time) VALUES ($1, $2, $3, $4, NOW()) RETURNING id"
	var purchaseID int
	err = tx.QueryRow(query, username, lotteryID, ticketsCount, cardNumber).Scan(&purchaseID)
	if err != nil {
		log.Printf("Error inserting purchase: %v", err)
		tx.Rollback()
		return err
	}
	log.Printf("Ticket purchased: Purchase ID %d", purchaseID)

	// Logic for randomly selecting the winner
	randQuery := "UPDATE purchases SET is_winner = TRUE WHERE id = $1 AND random() < 0.6 RETURNING id"
	var winnerID int
	err = tx.QueryRow(randQuery, purchaseID).Scan(&winnerID)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Println("No winner this time.")
		} else {
			log.Printf("Error marking winner: %v", err)
			tx.Rollback()
			return err
		}
	} else {
		// Insert the winning ticket into the winning_tickets table
		winningAmount := 500.0 // Example win amount
		_, err = tx.Exec("INSERT INTO winning_tickets (purchase_id, winning_amount) VALUES ($1, $2)", winnerID, winningAmount)
		if err != nil {
			log.Printf("Error inserting winning ticket: %v", err)
			tx.Rollback()
			return err
		}
		log.Printf("Winner immediately added: Purchase ID %d with winning amount %f", winnerID, winningAmount)
	}

	if err := tx.Commit(); err != nil {
		log.Printf("Error committing transaction: %v", err)
		return err
	}

	return nil
}

// Function for drawing winners
func DrawWinners(lotteryID int) error {
	tx, err := DB.Begin()
	if err != nil {
		log.Printf("Error starting transaction: %v", err)
		return err
	}

	// Update purchases table and select winners
	query := "UPDATE purchases SET is_winner = TRUE WHERE lottery_id = $1 AND is_winner = FALSE AND random() < 0.6 RETURNING id"
	rows, err := tx.Query(query, lotteryID)
	if err != nil {
		log.Printf("Error selecting winners: %v", err)
		tx.Rollback()
		return err
	}
	defer rows.Close()

	var purchaseIDs []int
	for rows.Next() {
		var purchaseID int
		if err := rows.Scan(&purchaseID); err != nil {
			log.Printf("Error scanning winner ID: %v", err)
			tx.Rollback()
			return err
		}
		purchaseIDs = append(purchaseIDs, purchaseID)
	}

	// Insert winners into the winning_tickets table
	if len(purchaseIDs) > 0 {
		for _, purchaseID := range purchaseIDs {
			winningAmount := 500.0 // Example win amount
			_, err = tx.Exec("INSERT INTO winning_tickets (purchase_id, winning_amount) VALUES ($1, $2)", purchaseID, winningAmount)
			if err != nil {
				log.Printf("Error inserting winning ticket: %v", err)
				tx.Rollback()
				return err
			}
			log.Printf("Winner added: Purchase ID %d with winning amount %f", purchaseID, winningAmount)
		}
	} else {
		log.Println("No winners in this draw.")
	}

	if err := tx.Commit(); err != nil {
		log.Printf("Error committing transaction: %v", err)
		return err
	}

	return nil
}
