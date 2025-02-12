package handlers

import (
	"log"
	"loto/db"
	"loto/models"
	"math/rand"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

// DrawsHandler handles displaying all draws
func DrawsHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("username")
	if err != nil || cookie.Value == "" {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	rows, err := db.DB.Query("SELECT id, lottery_id, draw_date, winner, prize_amount FROM draws")
	if err != nil {
		http.Error(w, "Unable to retrieve draws", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var draws []models.Draw
	for rows.Next() {
		var draw models.Draw
		if err := rows.Scan(&draw.ID, &draw.LotteryID, &draw.DrawDate, &draw.Winner, &draw.PrizeAmount); err != nil {
			http.Error(w, "Error scanning draw data", http.StatusInternalServerError)
			return
		}
		draws = append(draws, draw)
	}

	tmpl.ExecuteTemplate(w, "draws.html", map[string]interface{}{
		"Draws":    draws,
		"Username": cookie.Value,
	})
}

// PerformDraw automatically when a lottery ends
func PerformDraw(lotteryID int) {
	// Generate 6 fixed winning numbers for testing
	winningNumbers := generateWinningNumbers()
	winningNumbersStr := sortNumbers(numbersToString(winningNumbers))

	log.Println("Performing draw for lottery:", lotteryID, "Winning Numbers:", winningNumbersStr)

	// Calculate total prize amount based on ticket count
	var totalPrizeAmount float64
	err := db.DB.QueryRow("SELECT COUNT(*) FROM purchases WHERE lottery_id = $1", lotteryID).Scan(&totalPrizeAmount)
	if err != nil {
		log.Println("Error calculating total ticket count:", err)
		return
	}

	totalPrizeAmount *= getLotteryPrice(lotteryID) // Calculate prize pool

	// Get all chosen numbers for the lottery and check for winners
	rows, err := db.DB.Query("SELECT username, chosen_numbers FROM purchases WHERE lottery_id = $1", lotteryID)
	if err != nil {
		log.Println("Error fetching purchased numbers:", err)
		return
	}
	defer rows.Close()

	var winners []string

	for rows.Next() {
		var username string
		var chosenNumbers string

		if err := rows.Scan(&username, &chosenNumbers); err != nil {
			log.Println("Error scanning purchased numbers:", err)
			return
		}

		// Check if the user's numbers match the winning ones
		if chosenNumbers == winningNumbersStr {
			winners = append(winners, username)
		}
	}

	var prizePerWinner float64
	if len(winners) > 0 {
		prizePerWinner = totalPrizeAmount / float64(len(winners)) // Divide prize equally
	} else {
		winners = append(winners, "No winner")
		prizePerWinner = totalPrizeAmount
	}

	// Insert draw results into the database
	for _, winnerUsername := range winners {
		_, err = db.DB.Exec(`
			INSERT INTO draws (lottery_id, draw_date, winner, winning_numbers ,prize_amount) 
			VALUES ($1, NOW(), $2, $3, $4)`, lotteryID, winnerUsername, winningNumbersStr, prizePerWinner)

		if err != nil {
			log.Println("Error inserting draw:", err)
			return
		}
	}

	// Mark lottery as ended
	_, err = db.DB.Exec("UPDATE lotteries SET status = 'ended' WHERE id = $1", lotteryID)
	if err != nil {
		log.Println("Error updating lottery status:", err)
	}
}

// Get lottery price
func getLotteryPrice(lotteryID int) float64 {
	var price float64
	db.DB.QueryRow("SELECT price FROM lotteries WHERE id = $1", lotteryID).Scan(&price)
	return price
}

// Helper function to generate 6 random unique numbers
func generateWinningNumbers() []int {
	rand.Seed(time.Now().UnixNano())
	numbers := rand.Perm(49)[:6] // Generate numbers from 1 to 49
	sort.Ints(numbers)           // Ensure sorted order
	return numbers
}

// Convert slice of numbers to comma-separated string
func numbersToString(numbers []int) string {
	var strNumbers []string
	for _, n := range numbers {
		strNumbers = append(strNumbers, strconv.Itoa(n))
	}
	return strings.Join(strNumbers, ",")
}

// Sort numbers in a comma-separated string
func sortNumbers(numbersStr string) string {
	numbers := strings.Split(numbersStr, ",")
	intNumbers := make([]int, len(numbers))

	for i, num := range numbers {
		n, err := strconv.Atoi(strings.TrimSpace(num))
		if err != nil {
			log.Println("Error parsing number:", num)
			return ""
		}
		intNumbers[i] = n
	}

	sort.Ints(intNumbers)

	var sortedNumbers []string
	for _, n := range intNumbers {
		sortedNumbers = append(sortedNumbers, strconv.Itoa(n))
	}
	return strings.Join(sortedNumbers, ",")
}

// Check and perform draw for lotteries that have ended
func CheckForExpiredLotteries() {
	rows, err := db.DB.Query("SELECT id FROM lotteries WHERE end_date <= NOW() AND status = 'active'")
	if err != nil {
		log.Println("Error fetching expired lotteries:", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var lotteryID int
		if err := rows.Scan(&lotteryID); err != nil {
			log.Println("Error scanning expired lottery:", err)
			continue
		}
		PerformDraw(lotteryID)
	}
}

// Scheduled function to check for expired lotteries every minute
func StartDrawScheduler() {
	ticker := time.NewTicker(1 * time.Minute)
	go func() {
		for range ticker.C {
			CheckForExpiredLotteries()
		}
	}()
}

func CreateDrawHandler(w http.ResponseWriter, r *http.Request) {
	if !isAdmin(r) {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	if r.Method == http.MethodPost {
		lotteryID := r.FormValue("lottery_id")
		drawDateStr := r.FormValue("draw_date")
		winner := r.FormValue("winner")
		prizeAmountStr := r.FormValue("prize_amount")

		drawDate, err := time.Parse("2006-01-02T15:04", drawDateStr)
		if err != nil {
			http.Error(w, "Invalid draw date", http.StatusBadRequest)
			return
		}

		drawDate = time.Date(drawDate.Year(), drawDate.Month(), drawDate.Day(), 0, 0, 0, 0, time.UTC)

		prizeAmount, err := strconv.ParseFloat(prizeAmountStr, 64)
		if err != nil {
			http.Error(w, "Invalid prize amount", http.StatusBadRequest)
			return
		}

		_, err = db.DB.Exec("INSERT INTO draws (lottery_id, draw_date, winner, prize_amount) VALUES ($1, $2, $3, $4)",
			lotteryID, drawDate, winner, prizeAmount)
		if err != nil {
			log.Println("Error inserting draw:", err)
			http.Error(w, "Unable to create draw", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/draws", http.StatusSeeOther)
		return
	}

	rows, err := db.DB.Query("SELECT id, name FROM lotteries")
	if err != nil {
		http.Error(w, "Unable to fetch lotteries", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var lotteries []models.Lottery
	for rows.Next() {
		var lottery models.Lottery
		if err := rows.Scan(&lottery.ID, &lottery.Name); err != nil {
			http.Error(w, "Error reading lottery data", http.StatusInternalServerError)
			return
		}
		lotteries = append(lotteries, lottery)
	}

	tmpl.ExecuteTemplate(w, "create_draw.html", lotteries)
}

func UpdateDrawHandler(w http.ResponseWriter, r *http.Request) {
	if !isAdmin(r) {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	if r.Method == http.MethodPost {
		drawID := r.FormValue("draw_id")
		drawDateStr := r.FormValue("draw_date")
		winner := r.FormValue("winner")
		prizeAmountStr := r.FormValue("prize_amount")

		log.Printf("Received update form data: draw_id=%s, draw_date=%s, winner=%s, prize_amount=%s",
			drawID, drawDateStr, winner, prizeAmountStr)

		drawDate, err := time.Parse("2006-01-02T15:04", drawDateStr)
		if err != nil {
			log.Println("Error parsing draw date:", err)
			http.Error(w, "Invalid draw date", http.StatusBadRequest)
			return
		}

		drawDate = time.Date(drawDate.Year(), drawDate.Month(), drawDate.Day(), 0, 0, 0, 0, time.UTC)

		prizeAmount, err := strconv.ParseFloat(prizeAmountStr, 64)
		if err != nil {
			http.Error(w, "Invalid prize amount", http.StatusBadRequest)
			return
		}

		_, err = db.DB.Exec("UPDATE draws SET draw_date = $1, winner = $2, prize_amount = $3 WHERE id = $4",
			drawDate, winner, prizeAmount, drawID)
		if err != nil {
			log.Println("Error updating draw:", err)
			http.Error(w, "Unable to update draw", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/draws", http.StatusSeeOther)
		return
	}

	drawID := r.URL.Query().Get("id")
	if drawID == "" {
		http.Error(w, "Invalid draw ID", http.StatusBadRequest)
		return
	}
	row := db.DB.QueryRow("SELECT id, draw_date, winner, prize_amount FROM draws WHERE id = $1", drawID)

	var draw models.Draw
	if err := row.Scan(&draw.ID, &draw.DrawDate, &draw.Winner, &draw.PrizeAmount); err != nil {
		http.Error(w, "Draw not found", http.StatusNotFound)
		return
	}

	tmpl.ExecuteTemplate(w, "update_draw.html", draw)
}

func DeleteDrawHandler(w http.ResponseWriter, r *http.Request) {
	if !isAdmin(r) {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	drawID := r.URL.Query().Get("id")

	_, err := db.DB.Exec("DELETE FROM draws WHERE id = $1", drawID)
	if err != nil {
		log.Println("Error deleting draw:", err)
		http.Error(w, "Unable to delete draw", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/draws", http.StatusSeeOther)
}
