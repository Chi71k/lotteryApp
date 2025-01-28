package controllers

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"lotteryapp/config"
	"lotteryapp/models"
)

func CreateDrawHandler(w http.ResponseWriter, r *http.Request) {
	_, err := config.DB.Exec(
		"INSERT INTO draws (winning_numbers, draw_date) VALUES ($1, $2)",
		"", time.Now(),
	)
	if err != nil {
		log.Println("Error creating draw:", err)
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}

	w.Write([]byte("Draw created"))
}

func GetAllDrawsHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := config.DB.Query(
		"SELECT id, winning_numbers, draw_date FROM draws ORDER BY id DESC",
	)
	if err != nil {
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var draws []models.Draw
	for rows.Next() {
		var d models.Draw
		if err := rows.Scan(&d.ID, &d.WinningNumbers, &d.DrawDate); err == nil {
			draws = append(draws, d)
		}
	}
	for _, d := range draws {
		fmt.Fprintf(w, "Draw #%d: %s (%v)\n", d.ID, d.WinningNumbers, d.DrawDate)
	}
}

func ExecuteDrawHandler(w http.ResponseWriter, r *http.Request) {
	drawIDStr := r.FormValue("draw_id")
	drawID, _ := strconv.Atoi(drawIDStr)

	winning := "1,5,9"
	_, err := config.DB.Exec(
		"UPDATE draws SET winning_numbers=$1 WHERE id=$2",
		winning, drawID,
	)
	if err != nil {
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}

	_, err = config.DB.Exec(
		"UPDATE tickets SET status='WINNER' WHERE numbers=$1",
		winning,
	)
	if err != nil {
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}

	w.Write([]byte(fmt.Sprintf("Draw %d executed. Winning: %s", drawID, winning)))
}
