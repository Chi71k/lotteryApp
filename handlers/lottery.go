package handlers

import (
	"log"
	"net/http"

	"loto/db"
	"loto/models"
)

// var session = make(map[string]string)

func authenticateUser(username, password string) bool {
	// Dummy authentication logic for example purposes
	return username == "admin" && password == "password"
}

// Список лотерей
func LotteriesHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("username")
	if err != nil || cookie.Value == "" {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")

	if authenticateUser(username, password) {
		session["user"] = username
		http.Redirect(w, r, "/lotteries", http.StatusSeeOther)
		return
	}

	if authenticateUser(username, password) {
		session["user"] = username
		http.Redirect(w, r, "/lotteries", http.StatusSeeOther)
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

	tmpl.ExecuteTemplate(w, "lotteries.html", lotteries)
}

// Покупка лотереи
func BuyLotteryHandler(w http.ResponseWriter, r *http.Request) {
	if session["user"] == "" {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if r.Method == http.MethodPost {
		lotteryID := r.FormValue("lottery_id")
		user := session["user"]

		_, err := db.DB.Exec("INSERT INTO purchases (username, lottery_id) VALUES ($1, $2)",
			user, lotteryID)
		if err != nil {
			log.Println("BuyLottery error:", err)
			http.Error(w, "Unable to complete purchase", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/lotteries", http.StatusSeeOther)
		return
	}

	// Если GET — отправить обратно к списку лотерей
	http.Redirect(w, r, "/lotteries", http.StatusSeeOther)
}
