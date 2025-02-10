// db/db.go
package db

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

var DB *sql.DB

const (
	dbAdminConnection = "user=loto_user password=1234 dbname=postgres sslmode=disable"
	dbConnection      = "user=loto_user password=1234 dbname=goproject sslmode=disable"
)

// Init устанавливает соединение с базой данных.
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

// createDatabaseIfNotExists создаёт базу данных, если её нет.
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

// InitializeSchema создаёт все необходимые таблицы.
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
		end_date TIMESTAMP,
		ticket_limit INT,
		tickets_table_name VARCHAR(100)
	);

	CREATE TABLE IF NOT EXISTS lottery_analytics (
		lottery_id INT PRIMARY KEY REFERENCES lotteries(id),
		tickets_sold INT DEFAULT 0,
		total_money_received NUMERIC DEFAULT 0,
		total_winnings NUMERIC DEFAULT 0,
		total_project_expenses NUMERIC DEFAULT 0,
		total_charity NUMERIC DEFAULT 0
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

	CREATE TABLE IF NOT EXISTS payment_cards (
		id SERIAL PRIMARY KEY,
		user_id INT REFERENCES users(id),
		card_number VARCHAR(16) UNIQUE NOT NULL
	);
	`

	if _, err := DB.Exec(schema); err != nil {
		return err
	}

	alterUsers := `
	ALTER TABLE users 
	ADD COLUMN IF NOT EXISTS balance NUMERIC DEFAULT 0,
	ADD COLUMN IF NOT EXISTS profile_picture BYTEA
	`
	if _, err := DB.Exec(alterUsers); err != nil {
		return err
	}

	// Если в таблице lotteries нет записей, добавляем sample-данные.
	checkQuery := "SELECT COUNT(*) FROM lotteries"
	var count int
	if err := DB.QueryRow(checkQuery).Scan(&count); err != nil {
		return err
	}
	if count == 0 {
		log.Println("Adding sample lotteries...")
		insertQuery := `
		INSERT INTO lotteries (name, description, price, end_date, ticket_limit)
		VALUES
		('Mega Jackpot', 'Win the biggest prize in our history!', 100, '2025-12-31', 1000),
		('Holiday Special', 'Celebrate the holidays with amazing prizes!', 50, '2025-11-30', 500),
		('Weekly Draw', 'Join our weekly draw for exciting rewards!', 20, '2025-10-15', 200);
		`
		if _, err := DB.Exec(insertQuery); err != nil {
			return err
		}
		log.Println("Sample lotteries added.")
	}
	return nil
}

// sanitizeTableName очищает строку для использования в имени таблицы.
func sanitizeTableName(name string) string {
	return strings.ReplaceAll(name, " ", "_")
}

// createLotteryTicketsTable создаёт динамическую таблицу для билетов конкретной лотереи.
func createLotteryTicketsTable(tableName string) error {
	schema := fmt.Sprintf(`
	CREATE TABLE IF NOT EXISTS %s (
		id SERIAL PRIMARY KEY,
		username VARCHAR(50) NOT NULL,
		purchase_time TIMESTAMP DEFAULT NOW(),
		is_winner BOOLEAN DEFAULT FALSE,
		winning_amount NUMERIC DEFAULT 0
	);
	`, tableName)
	if _, err := DB.Exec(schema); err != nil {
		log.Printf("Error creating dynamic table %s: %v", tableName, err)
		return err
	}
	return nil
}

// CreateLottery создаёт новую лотерею, динамическую таблицу для билетов и инициализирует аналитику.
func CreateLottery(name, description string, price float64, endDate time.Time, ticketLimit int) error {
	tx, err := DB.Begin()
	if err != nil {
		return err
	}

	var lotteryID int
	err = tx.QueryRow(`
		INSERT INTO lotteries (name, description, price, end_date, ticket_limit)
		VALUES ($1, $2, $3, $4, $5) RETURNING id`,
		name, description, price, endDate, ticketLimit).Scan(&lotteryID)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("error inserting lottery: %v", err)
	}

	// Формируем имя динамической таблицы для билетов.
	sanitized := sanitizeTableName(name)
	tableName := fmt.Sprintf("lottery_%d_%s", lotteryID, sanitized)

	// Обновляем запись: устанавливаем tickets_table_name.
	if _, err = tx.Exec(`UPDATE lotteries SET tickets_table_name = $1 WHERE id = $2`, tableName, lotteryID); err != nil {
		tx.Rollback()
		return fmt.Errorf("error updating lottery with table name: %v", err)
	}

	// Создаём динамическую таблицу для билетов.
	if err = createLotteryTicketsTable(tableName); err != nil {
		tx.Rollback()
		return fmt.Errorf("error creating dynamic table: %v", err)
	}

	// Инициализируем запись аналитики для лотереи.
	if _, err = tx.Exec(`INSERT INTO lottery_analytics (lottery_id) VALUES ($1)`, lotteryID); err != nil {
		tx.Rollback()
		return fmt.Errorf("error inserting lottery analytics: %v", err)
	}

	return tx.Commit()
}

// PurchaseTicket совершает покупку билета с проверкой лимита и обновляет аналитику.
// PurchaseTicket совершает покупку билета с проверкой лимита и обновляет аналитику.
func PurchaseTicket(username string, lotteryID int) error {
	// Используем COALESCE, чтобы гарантировать ненулевые значения.
	query := `
		SELECT COALESCE(tickets_table_name, '') AS tickets_table_name,
		       price,
		       COALESCE(ticket_limit, 0) AS ticket_limit
		FROM lotteries
		WHERE id = $1
	`
	var tableName string
	var price float64
	var ticketLimit int
	err := DB.QueryRow(query, lotteryID).Scan(&tableName, &price, &ticketLimit)
	if err != nil {
		return fmt.Errorf("error fetching lottery info: %v", err)
	}
	if tableName == "" {
		return fmt.Errorf("lottery has no tickets table defined")
	}
	if ticketLimit == 0 {
		return fmt.Errorf("ticket_limit is not defined for lottery %d", lotteryID)
	}

	// Попытка получить количество проданных билетов из lottery_analytics.
	var ticketsSold int
	err = DB.QueryRow("SELECT tickets_sold FROM lottery_analytics WHERE lottery_id = $1", lotteryID).Scan(&ticketsSold)
	if err == sql.ErrNoRows {
		// Если записи нет, создадим её с начальными значениями.
		if _, errInsert := DB.Exec("INSERT INTO lottery_analytics (lottery_id) VALUES ($1)", lotteryID); errInsert != nil {
			return fmt.Errorf("error inserting lottery analytics: %v", errInsert)
		}
		ticketsSold = 0
	} else if err != nil {
		return fmt.Errorf("error fetching lottery analytics: %v", err)
	}

	if ticketsSold >= ticketLimit {
		return fmt.Errorf("ticket limit reached for lottery %d", lotteryID)
	}

	tx, err := DB.Begin()
	if err != nil {
		return fmt.Errorf("error starting transaction: %v", err)
	}

	// Вставляем запись о покупке билета в динамическую таблицу.
	query = fmt.Sprintf("INSERT INTO %s (username) VALUES ($1) RETURNING id", tableName)
	var purchaseID int
	err = tx.QueryRow(query, username).Scan(&purchaseID)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("error inserting purchase: %v", err)
	}
	log.Printf("Ticket purchased in table %s: Purchase ID %d", tableName, purchaseID)

	// Определяем, выиграл ли билет (вероятность 60%).
	randQuery := fmt.Sprintf("UPDATE %s SET is_winner = TRUE WHERE id = $1 AND random() < 0.6 RETURNING id", tableName)
	var winnerID int
	err = tx.QueryRow(randQuery, purchaseID).Scan(&winnerID)
	var winningAmount float64 = 0
	if err != nil {
		if err == sql.ErrNoRows {
			log.Println("No winner for this ticket.")
		} else {
			tx.Rollback()
			return fmt.Errorf("error marking winner: %v", err)
		}
	} else {
		winningAmount = 500 // Пример фиксированной суммы выигрыша.
		updateQuery := fmt.Sprintf("UPDATE %s SET winning_amount = $1 WHERE id = $2", tableName)
		if _, err = tx.Exec(updateQuery, winningAmount, winnerID); err != nil {
			tx.Rollback()
			return fmt.Errorf("error updating winning amount: %v", err)
		}
		log.Printf("Winner registered in table %s: Purchase ID %d, Winning Amount: %f", tableName, winnerID, winningAmount)
	}

	const ProjectExpensePercentage = 0.50
	const CharityPercentage = 0.25
	projectExpense := price * ProjectExpensePercentage
	charityAmount := price * CharityPercentage

	_, err = tx.Exec(`
		UPDATE lottery_analytics
		SET tickets_sold = tickets_sold + 1,
		    total_money_received = total_money_received + $1,
		    total_winnings = total_winnings + $2,
		    total_project_expenses = total_project_expenses + $3,
		    total_charity = total_charity + $4
		WHERE lottery_id = $5
	`, price, winningAmount, projectExpense, charityAmount, lotteryID)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("error updating lottery analytics: %v", err)
	}

	return tx.Commit()
}


// DrawWinners выбирает победителей для заданной лотереи.
func DrawWinners(lotteryID int) error {
	tx, err := DB.Begin()
	if err != nil {
		return fmt.Errorf("error starting transaction: %v", err)
	}

	query := `
		UPDATE purchases
		SET is_winner = TRUE
		WHERE lottery_id = $1 AND is_winner = FALSE AND random() < 0.6
		RETURNING id
	`
	rows, err := tx.Query(query, lotteryID)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("error selecting winners: %v", err)
	}
	defer rows.Close()

	var purchaseIDs []int
	for rows.Next() {
		var purchaseID int
		if err := rows.Scan(&purchaseID); err != nil {
			tx.Rollback()
			return fmt.Errorf("error scanning winner ID: %v", err)
		}
		purchaseIDs = append(purchaseIDs, purchaseID)
	}

	for _, purchaseID := range purchaseIDs {
		if _, err = tx.Exec("INSERT INTO winning_tickets (purchase_id, winning_amount) VALUES ($1, 500)", purchaseID); err != nil {
			tx.Rollback()
			return fmt.Errorf("error inserting winning ticket: %v", err)
		}
		log.Printf("Winner added: Purchase ID %d", purchaseID)
	}

	return tx.Commit()
}

// AddPaymentCard добавляет платёжную карту для пользователя. Если карта с указанными 16 цифрами уже существует, возвращает ошибку.
func AddPaymentCard(userID int, cardNumber string) error {
	// Проверяем, что номер карты состоит ровно из 16 символов.
	if len(cardNumber) != 16 {
		return fmt.Errorf("card number must be 16 digits")
	}
	query := `INSERT INTO payment_cards (user_id, card_number) VALUES ($1, $2)`
	_, err := DB.Exec(query, userID, cardNumber)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			return fmt.Errorf("card already exists")
		}
		return err
	}
	return nil
}

// GetWinningTickets возвращает список ID выигрышных покупок.
func GetWinningTickets() ([]int, error) {
	query := `SELECT id FROM purchases WHERE is_winner = TRUE`
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
