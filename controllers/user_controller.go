package controllers

import (
	"log"
	"net/http"
	"strconv"

	"lotteryapp/config"
	"lotteryapp/models"
)

func GetUserHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	id, _ := strconv.Atoi(idStr)

	var user models.User
	err := config.DB.QueryRow(
		"SELECT id, name, email, password_hash FROM users WHERE id = $1",
		id,
	).Scan(&user.ID, &user.Name, &user.Email, &user.PasswordHash)

	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	w.Write([]byte("User found: " + user.Name + " / " + user.Email))
}

func UpdateUserHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	id, _ := strconv.Atoi(idStr)

	newName := r.FormValue("name")
	newEmail := r.FormValue("email")

	_, err := config.DB.Exec(
		"UPDATE users SET name=$1, email=$2 WHERE id=$3",
		newName, newEmail, id,
	)
	if err != nil {
		log.Println("Error updating user:", err)
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

func DeleteUserHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	id, _ := strconv.Atoi(idStr)

	_, err := config.DB.Exec("DELETE FROM users WHERE id=$1", id)
	if err != nil {
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}

	w.Write([]byte("User deleted"))
}
