package controllers

import (
	"log"
	"net/http"
	"strconv"

	"lotteryapp/config"
	"lotteryapp/models"
)

func CreateTicketHandler(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromCookie(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	numbers := r.FormValue("numbers")

	_, err = config.DB.Exec(
		"INSERT INTO tickets (user_id, numbers, status) VALUES ($1, $2, $3)",
		userID, numbers, "ACTIVE",
	)
	if err != nil {
		log.Println("Error inserting ticket:", err)
		http.Error(w, "Unable to create ticket", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/tickets", http.StatusSeeOther)
}

func GetAllTicketsHandler(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromCookie(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	rows, err := config.DB.Query(
		"SELECT id, user_id, numbers, status FROM tickets WHERE user_id = $1",
		userID,
	)
	if err != nil {
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var tickets []models.Ticket
	for rows.Next() {
		var t models.Ticket
		if err := rows.Scan(&t.ID, &t.UserID, &t.Numbers, &t.Status); err == nil {
			tickets = append(tickets, t)
		}
	}

	RenderTemplate(w, "dashboard.html", tickets)
}

func UpdateTicketHandler(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromCookie(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	tidStr := r.URL.Query().Get("id")
	tid, _ := strconv.Atoi(tidStr)
	newNumbers := r.FormValue("numbers")

	var owner int
	err = config.DB.QueryRow(
		"SELECT user_id FROM tickets WHERE id = $1",
		tid,
	).Scan(&owner)
	if err != nil || owner != userID {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	_, err = config.DB.Exec(
		"UPDATE tickets SET numbers=$1 WHERE id=$2 AND status='ACTIVE'",
		newNumbers, tid,
	)
	if err != nil {
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/tickets", http.StatusSeeOther)
}

func DeleteTicketHandler(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromCookie(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	tidStr := r.URL.Query().Get("id")
	tid, _ := strconv.Atoi(tidStr)

	var owner int
	err = config.DB.QueryRow(
		"SELECT user_id FROM tickets WHERE id = $1",
		tid,
	).Scan(&owner)
	if err != nil || owner != userID {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	_, err = config.DB.Exec(
		"DELETE FROM tickets WHERE id=$1 AND status='ACTIVE'",
		tid,
	)
	if err != nil {
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/tickets", http.StatusSeeOther)
}

func getUserIDFromCookie(r *http.Request) (int, error) {
	cookie, err := r.Cookie("user_id")
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(cookie.Value)
}
