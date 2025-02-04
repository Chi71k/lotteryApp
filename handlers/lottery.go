package handlers

import (
	"log"
	"loto/db"
	"loto/models"
	"net/http"
	"strconv"
	"time"
)

// Authenticate the user
func authenticateUser(r *http.Request) bool {
	cookie, err := r.Cookie("username")
	if err != nil || cookie.Value == "" {
		return false
	}
	return true
}

// List available lotteries
func LotteriesHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("username")
	if err != nil || cookie.Value == "" {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	rows, err := db.DB.Query("SELECT id, name, description, price, end_date FROM lotteries")
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

	// Pass lotteries data and the username to the template
	tmpl.ExecuteTemplate(w, "lotteries.html", map[string]interface{}{
		"Lotteries": lotteries,
		"Username":  cookie.Value,
	})
}

// Handle ticket purchase
func BuyLotteryHandler(w http.ResponseWriter, r *http.Request) {
	if !authenticateUser(r) {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if r.Method == http.MethodPost {
		lotteryID := r.FormValue("lottery_id")
		cookie, err := r.Cookie("username")
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		username := cookie.Value

		_, err = db.DB.Exec("INSERT INTO purchases (username, lottery_id) VALUES ($1, $2)", username, lotteryID)
		if err != nil {
			log.Println("BuyLottery error:", err)
			http.Error(w, "Unable to complete purchase", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/lotteries", http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/lotteries", http.StatusSeeOther)
}

// Create a new lottery (only accessible by admin)
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

		// Parse the end date
		endDate, err := time.Parse("2006-01-02T15:04", endDateStr)
		if err != nil {
			http.Error(w, "Invalid end date", http.StatusBadRequest)
			return
		}

		// Insert the new lottery into the database
		_, err = db.DB.Exec("INSERT INTO lotteries (name, description, price, end_date) VALUES ($1, $2, $3, $4)",
			name, description, price, endDate)
		if err != nil {
			log.Println("Error inserting lottery:", err)
			http.Error(w, "Unable to create lottery", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/lotteries", http.StatusSeeOther)
		return
	}

	// Render the create lottery form
	tmpl.ExecuteTemplate(w, "create_lottery.html", nil)
}

// Edit an existing lottery (only accessible by admin)
// Update an existing lottery (only accessible by admin)
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

	// Fetch the lottery details
	row := db.DB.QueryRow("SELECT id, name, description, price, end_date FROM lotteries WHERE id = $1", lotteryID)

	var lottery models.Lottery
	if err := row.Scan(&lottery.ID, &lottery.Name, &lottery.Description, &lottery.Price, &lottery.EndDate); err != nil {
		http.Error(w, "Lottery not found", http.StatusNotFound)
		return
	}

	if r.Method == http.MethodPost {
		// Get form values
		name := r.FormValue("name")
		description := r.FormValue("description")
		priceStr := r.FormValue("price")
		endDateStr := r.FormValue("end_date")

		price, err := strconv.ParseFloat(priceStr, 64)
		if err != nil {
			http.Error(w, "Invalid price", http.StatusBadRequest)
			return
		}

		// Parse the end date
		endDate, err := time.Parse("2006-01-02T15:04", endDateStr)
		if err != nil {
			http.Error(w, "Invalid end date", http.StatusBadRequest)
			return
		}

		// Update the lottery in the database
		_, err = db.DB.Exec("UPDATE lotteries SET name = $1, description = $2, price = $3, end_date = $4 WHERE id = $5",
			name, description, price, endDate, lotteryID)
		if err != nil {
			http.Error(w, "Unable to update lottery", http.StatusInternalServerError)
			return
		}

		// Redirect to the lotteries page after the update
		http.Redirect(w, r, "/lotteries", http.StatusSeeOther)
		return
	}

	// Render the update lottery form with the existing values
	tmpl.ExecuteTemplate(w, "update_lottery.html", lottery)
}

// Delete a lottery (only accessible by admin)
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

	// Redirect to the lotteries page after deletion
	http.Redirect(w, r, "/lotteries", http.StatusSeeOther)
}
