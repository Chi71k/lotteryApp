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

	// Вызываем инициализацию схемы после успешного подключения
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
		lottery_id INT REFERENCES lotteries(id),
		is_winner BOOLEAN DEFAULT FALSE
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
		winner VARCHAR(100),
		prize_amount NUMERIC
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
		('Weekly Draw', 'Join our weekly draw for exciting rewards!', 20, '2025-10-15');`
		
		_, err := DB.Exec(insertQuery)
		if err != nil {
			return err
		}
		log.Println("Sample lotteries added.")
	}
	return nil
}

func GetWinningTickets() ([]int, error) {
	query := "SELECT id FROM purchases WHERE is_winner = TRUE"
	rows, err := DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var winningIDs []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		winningIDs = append(winningIDs, id)
	}
	return winningIDs, nil
}

func PurchaseTicket(username string, lotteryID int) error {
	tx, err := DB.Begin()
	if err != nil {
		log.Printf("Error starting transaction: %v", err)
		return err
	}

	query := "INSERT INTO purchases (username, lottery_id) VALUES ($1, $2) RETURNING id"
	var purchaseID int
	err = tx.QueryRow(query, username, lotteryID).Scan(&purchaseID)
	if err != nil {
		log.Printf("Error inserting purchase: %v", err)
		tx.Rollback()
		return err
	}
	log.Printf("Ticket purchased: Purchase ID %d", purchaseID)

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
		_, err = tx.Exec("INSERT INTO winning_tickets (purchase_id, winning_amount) VALUES ($1, 500)", winnerID)
		if err != nil {
			log.Printf("Error inserting winning ticket: %v", err)
			tx.Rollback()
			return err
		}
		log.Printf("Winner immediately added: Purchase ID %d", winnerID)
	}

	if err := tx.Commit(); err != nil {
		log.Printf("Error committing transaction: %v", err)
		return err
	}

	return nil
}

func DrawWinners(lotteryID int) error {
	tx, err := DB.Begin()
	if err != nil {
		log.Printf("Error starting transaction: %v", err)
		return err
	}

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

	if len(purchaseIDs) > 0 {
		for _, purchaseID := range purchaseIDs {
			_, err = tx.Exec("INSERT INTO winning_tickets (purchase_id, winning_amount) VALUES ($1, 500)", purchaseID)
			if err != nil {
				log.Printf("Error inserting winning ticket: %v", err)
				tx.Rollback()
				return err
			}
			log.Printf("Winner added: Purchase ID %d", purchaseID)
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
