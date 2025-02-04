package handlers

import (
	"log"
	"loto/db"
	"loto/models"
	"net/http"
	"strconv"
	"time"
)

// DrawsHandler - Fetches and displays all draws, only accessible by logged-in users
func DrawsHandler(w http.ResponseWriter, r *http.Request) {
	// Check if the user is logged in
	cookie, err := r.Cookie("username")
	if err != nil || cookie.Value == "" {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// Fetch draws from the database
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

	// Pass the logged-in user's username and the draws to the template
	tmpl.ExecuteTemplate(w, "draws.html", map[string]interface{}{
		"Draws":    draws,
		"Username": cookie.Value, // Pass the username to the template
	})
}

// Displays form for creating a new draw (only for admins)
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

		// Parse form data with yyyy-mm-ddTHH:mm format (e.g., 2025-02-04T14:30)
		drawDate, err := time.Parse("2006-01-02T15:04", drawDateStr)
		if err != nil {
			http.Error(w, "Invalid draw date", http.StatusBadRequest)
			return
		}

		// Set the time part to midnight (00:00:00) if necessary
		drawDate = time.Date(drawDate.Year(), drawDate.Month(), drawDate.Day(), 0, 0, 0, 0, time.UTC)

		prizeAmount, err := strconv.ParseFloat(prizeAmountStr, 64)
		if err != nil {
			http.Error(w, "Invalid prize amount", http.StatusBadRequest)
			return
		}

		// Insert the new draw into the database
		_, err = db.DB.Exec("INSERT INTO draws (lottery_id, draw_date, winner, prize_amount) VALUES ($1, $2, $3, $4)",
			lotteryID, drawDate, winner, prizeAmount)
		if err != nil {
			log.Println("Error inserting draw:", err)
			http.Error(w, "Unable to create draw", http.StatusInternalServerError)
			return
		}

		// Redirect back to the draws list
		http.Redirect(w, r, "/draws", http.StatusSeeOther)
		return
	}

	// Fetch all lotteries to show in the form
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

	// Render the create draw form with the lotteries data
	tmpl.ExecuteTemplate(w, "create_draw.html", lotteries)
}

// Update a specific draw (only for admins)
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

		// Log incoming form values to debug
		log.Printf("Received update form data: draw_id=%s, draw_date=%s, winner=%s, prize_amount=%s",
			drawID, drawDateStr, winner, prizeAmountStr)

		// Parse form data with yyyy-mm-ddTHH:mm format (e.g., 2025-02-04T14:30)
		drawDate, err := time.Parse("2006-01-02T15:04", drawDateStr)
		if err != nil {
			log.Println("Error parsing draw date:", err) // Log the error
			http.Error(w, "Invalid draw date", http.StatusBadRequest)
			return
		}

		// Set the time part to midnight (00:00:00) if necessary
		drawDate = time.Date(drawDate.Year(), drawDate.Month(), drawDate.Day(), 0, 0, 0, 0, time.UTC)

		prizeAmount, err := strconv.ParseFloat(prizeAmountStr, 64)
		if err != nil {
			http.Error(w, "Invalid prize amount", http.StatusBadRequest)
			return
		}

		// Update the draw in the database
		_, err = db.DB.Exec("UPDATE draws SET draw_date = $1, winner = $2, prize_amount = $3 WHERE id = $4",
			drawDate, winner, prizeAmount, drawID)
		if err != nil {
			log.Println("Error updating draw:", err)
			http.Error(w, "Unable to update draw", http.StatusInternalServerError)
			return
		}

		// Redirect back to the draws list
		http.Redirect(w, r, "/draws", http.StatusSeeOther)
		return
	}

	// Fetch the draw ID from the URL
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

	// Render the update draw form with the existing values
	tmpl.ExecuteTemplate(w, "update_draw.html", draw)
}

// Delete a specific draw (only for admins)
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

	// Redirect back to the draws list
	http.Redirect(w, r, "/draws", http.StatusSeeOther)
}
