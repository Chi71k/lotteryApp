package handlers

import (
	"log"
	"loto/db"
	"loto/models"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Authenticate the user via session cookie
func authenticateUser(r *http.Request) bool {
	cookie, err := r.Cookie("username")
	return err == nil && cookie.Value != ""
}

// Fetch only active lotteries
func LotteriesHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("username")
	if err != nil || cookie.Value == "" {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	rows, err := db.DB.Query("SELECT id, name, description, price, end_date FROM lotteries WHERE status = 'active'")
	if err != nil {
		http.Error(w, "Unable to fetch lotteries", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var lotteries []models.Lottery
	for rows.Next() {
		var l models.Lottery
		if err := rows.Scan(&l.ID, &l.Name, &l.Description, &l.Price, &l.EndDate); err != nil {
			http.Error(w, "Error reading lotteries", http.StatusInternalServerError)
			return
		}
		lotteries = append(lotteries, l)
	}

	tmpl.ExecuteTemplate(w, "lotteries.html", map[string]interface{}{
		"Lotteries": lotteries,
		"Username":  cookie.Value,
	})
}

// Validate lottery numbers (6 unique numbers between 1 and 49)
func validateLotteryNumbers(numbers string) bool {
	numSet := make(map[int]bool)
	numArr := strings.Split(numbers, ",")

	if len(numArr) != 6 {
		return false
	}

	for _, numStr := range numArr {
		num, err := strconv.Atoi(strings.TrimSpace(numStr))
		if err != nil || num < 1 || num > 49 {
			return false
		}

		if numSet[num] {
			return false // Duplicate number detected
		}
		numSet[num] = true
	}

	return true
}

// Helper function to normalize (sort) chosen numbers
func normalizeNumbers(numbers string) string {
	numArr := strings.Split(numbers, ",")
	nums := make([]int, len(numArr))

	// Convert to integers
	for i, numStr := range numArr {
		num, err := strconv.Atoi(strings.TrimSpace(numStr))
		if err != nil {
			return "" // Return empty string if conversion fails (invalid input)
		}
		nums[i] = num
	}

	// Sort numbers
	sort.Ints(nums)

	// Convert back to string
	strArr := make([]string, len(nums))
	for i, num := range nums {
		strArr[i] = strconv.Itoa(num)
	}

	return strings.Join(strArr, ",") // Return sorted numbers as a single string
}

// Handle ticket purchases
func BuyLotteryHandler(w http.ResponseWriter, r *http.Request) {
	if !authenticateUser(r) {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if r.Method == http.MethodPost {
		lotteryID := r.FormValue("lottery_id")
		chosenNumbers := r.FormValue("chosen_numbers")
		cookie, err := r.Cookie("username")
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		username := cookie.Value

		if !validateLotteryNumbers(chosenNumbers) {
			// Получаем список активных лотерей
			rows, err := db.DB.Query("SELECT id, name, description, price, end_date FROM lotteries WHERE status = 'active'")
			if err != nil {
				http.Error(w, "Unable to fetch lotteries", http.StatusInternalServerError)
				return
			}
			defer rows.Close()

			var lotteries []models.Lottery
			for rows.Next() {
				var l models.Lottery
				if err := rows.Scan(&l.ID, &l.Name, &l.Description, &l.Price, &l.EndDate); err != nil {
					http.Error(w, "Error reading lotteries", http.StatusInternalServerError)
					return
				}
				lotteries = append(lotteries, l)
			}

			// Отправляем ошибку и лотереи обратно в шаблон
			data := map[string]interface{}{
				"Error":     "Invalid numbers. Choose 6 unique numbers between 1 and 49.",
				"Lotteries": lotteries,
				"Username":  username,
			}
			tmpl.ExecuteTemplate(w, "lotteries.html", data)
			return
		}

		// Normalize (sort) the chosen numbers
		normalizedNumbers := normalizeNumbers(chosenNumbers)

		// Check if the user already has chosen the same (sorted) numbers for this lottery
		var count int
		err = db.DB.QueryRow(
			"SELECT COUNT(*) FROM purchases WHERE username = $1 AND lottery_id = $2 AND chosen_numbers = $3",
			username, lotteryID, normalizedNumbers).Scan(&count)

		if err != nil {
			log.Println("Database error:", err)
			data := map[string]interface{}{
				"Error": "Internal server error",
			}
			tmpl.ExecuteTemplate(w, "lotteries.html", data)
			return
		}

		if count > 0 {
			data := map[string]interface{}{
				"Error": "You have already selected these numbers for this lottery. Please choose different numbers.",
			}
			tmpl.ExecuteTemplate(w, "lotteries.html", data)
			return
		}

		// Insert purchase and update ticket count
		tx, err := db.DB.Begin()
		if err != nil {
			data := map[string]interface{}{
				"Error": "Transaction error",
			}
			tmpl.ExecuteTemplate(w, "lotteries.html", data)
			return
		}

		_, err = tx.Exec(
			"INSERT INTO purchases (username, lottery_id, chosen_numbers, purchase_time) VALUES ($1, $2, $3, NOW())",
			username, lotteryID, normalizedNumbers)

		if err != nil {
			tx.Rollback()
			log.Println("BuyLottery error:", err)
			data := map[string]interface{}{
				"Error": "Unable to complete purchase",
			}
			tmpl.ExecuteTemplate(w, "lotteries.html", data)
			return
		}

		tx.Commit()
		http.Redirect(w, r, "/lotteries", http.StatusSeeOther)
	}
}



// Create a new lottery
func CreateLotteryHandler(w http.ResponseWriter, r *http.Request) {
	if !isAdmin(r) {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	if r.Method == http.MethodPost {
		name := r.FormValue("name")
		description := r.FormValue("description")
		priceStr := r.FormValue("price")
		endDateStr := r.FormValue("end_date")

		price, err := strconv.ParseFloat(priceStr, 64)
		if err != nil {
			http.Error(w, "Invalid price", http.StatusBadRequest)
			return
		}

		endDate, err := time.Parse("2006-01-02T15:04", endDateStr)
		if err != nil {
			http.Error(w, "Invalid end date", http.StatusBadRequest)
			return
		}

		_, err = db.DB.Exec(
			"INSERT INTO lotteries (name, description, price, end_date) VALUES ($1, $2, $3, $4)",
			name, description, price, endDate)
		if err != nil {
			log.Println("Error inserting lottery:", err)
			http.Error(w, "Unable to create lottery", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/lotteries", http.StatusSeeOther)
	}

	tmpl.ExecuteTemplate(w, "create_lottery.html", nil)
}

// Update lottery details
func UpdateLotteryHandler(w http.ResponseWriter, r *http.Request) {
	if !isAdmin(r) {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	lotteryID := r.URL.Query().Get("id")
	if lotteryID == "" {
		http.Error(w, "Invalid lottery ID", http.StatusBadRequest)
		return
	}

	row := db.DB.QueryRow("SELECT id, name, description, price, end_date FROM lotteries WHERE id = $1", lotteryID)
	var lottery models.Lottery
	if err := row.Scan(&lottery.ID, &lottery.Name, &lottery.Description, &lottery.Price, &lottery.EndDate); err != nil {
		http.Error(w, "Lottery not found", http.StatusNotFound)
		return
	}

	if r.Method == http.MethodPost {
		name := r.FormValue("name")
		description := r.FormValue("description")
		priceStr := r.FormValue("price")
		endDateStr := r.FormValue("end_date")

		price, err := strconv.ParseFloat(priceStr, 64)
		if err != nil {
			http.Error(w, "Invalid price", http.StatusBadRequest)
			return
		}

		endDate, err := time.Parse("2006-01-02T15:04", endDateStr)
		if err != nil {
			http.Error(w, "Invalid end date", http.StatusBadRequest)
			return
		}

		_, err = db.DB.Exec("UPDATE lotteries SET name = $1, description = $2, price = $3, end_date = $4 WHERE id = $5",
			name, description, price, endDate, lotteryID)
		if err != nil {
			http.Error(w, "Unable to update lottery", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/lotteries", http.StatusSeeOther)
	}
	tmpl.ExecuteTemplate(w, "update_lottery.html", nil)
}

// Mark expired lotteries as "ended"
func RemoveExpiredLotteries() {
	for {
		time.Sleep(1 * time.Minute)

		_, err := db.DB.Exec("UPDATE lotteries SET status = 'ended' WHERE end_date < NOW() AND status = 'active'")
		if err != nil {
			log.Println("Error updating expired lotteries:", err)
		}
	}
}

// Delete a lottery
func DeleteLotteryHandler(w http.ResponseWriter, r *http.Request) {
	if !isAdmin(r) {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	lotteryID := r.URL.Query().Get("id")
	_, err := db.DB.Exec("DELETE FROM lotteries WHERE id = $1", lotteryID)
	if err != nil {
		log.Println("Error deleting lottery:", err)
		http.Error(w, "Unable to delete lottery", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/lotteries", http.StatusSeeOther)
}
