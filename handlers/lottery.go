// lottery.go
package handlers

import (
	"html/template"
	"loto/db"
	"loto/models"
	"net/http"
	"strconv"
	"time"
)

var lotteryTmpl = template.Must(template.ParseGlob("templates/*.html"))

// authenticateUser проверяет, что пользователь аутентифицирован (наличие cookie "username").
func authenticateUser(r *http.Request) bool {
	cookie, err := r.Cookie("username")
	if err != nil || cookie.Value == "" {
		return false
	}
	return true
}

// LotteriesHandler отображает список лотерей.
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

	lotteryTmpl.ExecuteTemplate(w, "lotteries.html", map[string]interface{}{
		"Lotteries": lotteries,
		"Username":  cookie.Value,
	})
}

// BuyLotteryHandler обрабатывает покупку билета пользователем.
func BuyLotteryHandler(w http.ResponseWriter, r *http.Request) {
	if !authenticateUser(r) {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if r.Method == http.MethodPost {
		lotteryIDStr := r.FormValue("lottery_id")
		cookie, err := r.Cookie("username")
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		username := cookie.Value

		lotteryID, err := strconv.Atoi(lotteryIDStr)
		if err != nil {
			http.Error(w, "Invalid lottery ID", http.StatusBadRequest)
			return
		}

		err = db.PurchaseTicket(username, lotteryID)
		if err != nil {
			http.Error(w, "Error purchasing ticket: "+err.Error(), http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/lotteries", http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/lotteries", http.StatusSeeOther)
}

// CreateLotteryHandler позволяет администратору создать новую лотерею.
func CreateLotteryHandler(w http.ResponseWriter, r *http.Request) {
	if !isAdmin(r) {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	if r.Method == http.MethodPost {
		name := r.FormValue("name")
		description := r.FormValue("description")
		priceStr := r.FormValue("price")
		price, err := strconv.ParseFloat(priceStr, 64)
		if err != nil {
			http.Error(w, "Invalid price", http.StatusBadRequest)
			return
		}
		endDateStr := r.FormValue("end_date")
		endDate, err := time.Parse("2006-01-02T15:04", endDateStr)
		if err != nil {
			http.Error(w, "Invalid end date", http.StatusBadRequest)
			return
		}
		ticketLimitStr := r.FormValue("ticket_limit")
		ticketLimit, err := strconv.Atoi(ticketLimitStr)
		if err != nil {
			http.Error(w, "Invalid ticket limit", http.StatusBadRequest)
			return
		}

		err = db.CreateLottery(name, description, price, endDate, ticketLimit)
		if err != nil {
			http.Error(w, "Error creating lottery: "+err.Error(), http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/lotteries", http.StatusSeeOther)
		return
	}

	lotteryTmpl.ExecuteTemplate(w, "create_lottery.html", nil)
}

// UpdateLotteryHandler позволяет обновить данные лотереи.
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
		price, err := strconv.ParseFloat(priceStr, 64)
		if err != nil {
			http.Error(w, "Invalid price", http.StatusBadRequest)
			return
		}
		endDateStr := r.FormValue("end_date")
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
		return
	}

	lotteryTmpl.ExecuteTemplate(w, "update_lottery.html", lottery)
}

// DeleteLotteryHandler удаляет лотерею.
func DeleteLotteryHandler(w http.ResponseWriter, r *http.Request) {
	if !isAdmin(r) {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	lotteryID := r.URL.Query().Get("id")
	_, err := db.DB.Exec("DELETE FROM lotteries WHERE id = $1", lotteryID)
	if err != nil {
		http.Error(w, "Unable to delete lottery", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/lotteries", http.StatusSeeOther)
}
func AddPaymentCardHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		userIDStr := r.FormValue("user_id")
		cardNumber := r.FormValue("card_number")
		userID, err := strconv.Atoi(userIDStr)
		if err != nil {
			http.Error(w, "Invalid user ID", http.StatusBadRequest)
			return
		}
		err = db.AddPaymentCard(userID, cardNumber)
		if err != nil {
			http.Error(w, "Error adding payment card: "+err.Error(), http.StatusBadRequest)
			return
		}
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	tmpl.ExecuteTemplate(w, "add_payment_card.html", nil)
}