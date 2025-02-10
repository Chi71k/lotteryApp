package handlers

import (
    "net/http"
    "loto/db"
    "loto/models"
    "log"
)

func AdminDashboardHandler(w http.ResponseWriter, r *http.Request) {
    if !isAdmin(r) {
        http.Redirect(w, r, "/", http.StatusSeeOther)
        return
    }

    rows, err := db.DB.Query("SELECT id, name, description, price, end_date FROM lotteries")
    if err != nil {
        log.Println("Error fetching lotteries:", err)
        http.Error(w, "Unable to fetch lotteries", http.StatusInternalServerError)
        return
    }
    defer rows.Close()

    var lotteries []models.Lottery
    for rows.Next() {
        var l models.Lottery
        if err := rows.Scan(&l.ID, &l.Name, &l.Description, &l.Price, &l.EndDate); err != nil {
            log.Println("Error scanning lottery:", err)
            continue
        }
        lotteries = append(lotteries, l)
    }

    rows2, err := db.DB.Query("SELECT id, lottery_id, draw_date, winner, prize_amount FROM draws")
    if err != nil {
        log.Println("Error fetching draws:", err)
        http.Error(w, "Unable to fetch draws", http.StatusInternalServerError)
        return
    }
    defer rows2.Close()

    var draws []models.Draw
    for rows2.Next() {
        var d models.Draw
        if err := rows2.Scan(&d.ID, &d.LotteryID, &d.DrawDate, &d.Winner, &d.PrizeAmount); err != nil {
            log.Println("Error scanning draw:", err)
            continue
        }
        draws = append(draws, d)
    }

    data := map[string]interface{}{
        "Lotteries": lotteries,
        "Draws":     draws,
    }

    tmpl.ExecuteTemplate(w, "admin.html", data)
}
