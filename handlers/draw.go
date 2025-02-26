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

func DrawsHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("username")
	if err != nil || cookie.Value == "" {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	rows, err := db.DB.Query("SELECT id, lottery_id, draw_date, winner, prize_amount FROM draws")
	if err != nil {
		data := map[string]interface{}{
			"Error": "Unable to retrieve draws",
		}
		tmpl.ExecuteTemplate(w, "draws.html", data)
		return
	}
	defer rows.Close()

	var draws []models.Draw
	for rows.Next() {
		var draw models.Draw
		if err := rows.Scan(&draw.ID, &draw.LotteryID, &draw.DrawDate, &draw.Winner, &draw.PrizeAmount); err != nil {
			data := map[string]interface{}{
				"Error": "Error scanning draw data",
			}
			tmpl.ExecuteTemplate(w, "draws.html", data)
			return
		}
		draws = append(draws, draw)
	}

	tmpl.ExecuteTemplate(w, "draws.html", map[string]interface{}{
		"Draws":    draws,
		"Username": cookie.Value,
	})
}

func UpdateLotteryAnalysis(lotteryID int) error {
    var totalSales int
    err := db.DB.QueryRow("SELECT COUNT(*) FROM purchases WHERE lottery_id = $1", lotteryID).Scan(&totalSales)
    if err != nil {
        log.Println("Error calculating total sales:", err)
        return err
    }

    var remainingTickets int
    err = db.DB.QueryRow("SELECT COUNT(*) FROM purchases WHERE lottery_id = $1 AND is_winner = FALSE", lotteryID).Scan(&remainingTickets)
    if err != nil {
        log.Println("Error calculating remaining tickets:", err)
        return err
    }

    var winnersCount int
    err = db.DB.QueryRow(`
        SELECT COUNT(*) 
        FROM winning_tickets 
        WHERE purchase_id IN (
            SELECT id FROM purchases WHERE lottery_id = $1
        )`, lotteryID).Scan(&winnersCount)
    if err != nil {
        log.Println("Error calculating winners count:", err)
        return err
    }

    var totalRevenue float64
    err = db.DB.QueryRow("SELECT price FROM lotteries WHERE id = $1", lotteryID).Scan(&totalRevenue)
    if err != nil {
        log.Println("Error calculating total revenue:", err)
        return err
    }

    totalRevenue *= float64(totalSales)
    var sponsorShare, charityShare, winnerShare float64
    if winnersCount == 0 {
        sponsorShare = 0.75 * totalRevenue
        charityShare = 0.25 * totalRevenue
        winnerShare = 0
    } else {
        sponsorShare = 0.25 * totalRevenue
        charityShare = 0.25 * totalRevenue
        winnerShare = 0.50 * totalRevenue
    }

    _, err = db.DB.Exec(`
        INSERT INTO lottery_analysis (
            lottery_id, 
            total_sales, 
            remaining_tickets, 
            winners_count, 
            total_revenue, 
            sponsor_share, 
            charity_share, 
            winner_share
        )
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
        ON CONFLICT (lottery_id) DO UPDATE SET
            total_sales = EXCLUDED.total_sales,
            remaining_tickets = EXCLUDED.remaining_tickets,
            winners_count = EXCLUDED.winners_count,
            total_revenue = EXCLUDED.total_revenue,
            sponsor_share = EXCLUDED.sponsor_share,
            charity_share = EXCLUDED.charity_share,
            winner_share = EXCLUDED.winner_share
    `,
        lotteryID, totalSales, remainingTickets, winnersCount,
        totalRevenue, sponsorShare, charityShare, winnerShare,
    )
    if err != nil {
        log.Println("Error updating lottery analysis:", err)
        return err
    }

    log.Printf("Updated analysis for lottery %d", lotteryID)
    return nil
}



func PerformDraw(lotteryID int) {
	winningNumbers := generateWinningNumbers()
    // winningNumbers := []int{1, 2, 3, 4, 5, 6}
    winningNumbersStr := sortNumbers(numbersToString(winningNumbers))

    log.Println("Performing draw for lottery:", lotteryID, "Winning Numbers:", winningNumbersStr)

    var totalTicketCount float64
    err := db.DB.QueryRow("SELECT COUNT(*) FROM purchases WHERE lottery_id = $1", lotteryID).Scan(&totalTicketCount)
    if err != nil {
        log.Println("Error calculating total ticket count:", err)
        return
    }

    ticketPrice := getLotteryPrice(lotteryID)

    totalPrizeAmount := totalTicketCount * ticketPrice

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

        if chosenNumbers == winningNumbersStr {
            winners = append(winners, username)
        }
    }

    var prizePerWinner float64
    if len(winners) > 0 {
        prizePerWinner = 0.5 * totalPrizeAmount / float64(len(winners))
    } else {
        winners = append(winners, "No winner")
        prizePerWinner = totalPrizeAmount
    }

    for _, username := range winners {
        _, err = db.DB.Exec(`
            INSERT INTO draws (lottery_id, draw_date, winner, winning_numbers, prize_amount)
            VALUES ($1, NOW(), $2, $3, $4)`,
            lotteryID, username, winningNumbersStr, prizePerWinner)
        if err != nil {
            log.Println("Error inserting draw:", err)
            return
        }
    }

    for _, username := range winners {
        if username == "No winner" {
            continue
        }
        _, err = db.DB.Exec(`
            INSERT INTO winning_tickets (purchase_id, winning_amount)
            SELECT p.id, $1
            FROM purchases p
            WHERE p.username = $2
              AND p.lottery_id = $3
              AND p.chosen_numbers = $4
        `, prizePerWinner, username, lotteryID, winningNumbersStr)
        if err != nil {
            log.Println("Error inserting winning ticket:", err)
            return
        }
        _, err = db.DB.Exec(`
            UPDATE users
            SET balance = balance + $1
            WHERE username = $2
        `, prizePerWinner, username)
        if err != nil {
            log.Println("Error updating winner's balance:", err)
            return
        }
    }

    _, err = db.DB.Exec("UPDATE lotteries SET status = 'ended' WHERE id = $1", lotteryID)
    if err != nil {
        log.Println("Error updating lottery status:", err)
    }

    err = UpdateLotteryAnalysis(lotteryID)
    if err != nil {
        log.Println("Error updating lottery analysis:", err)
    }
}




func getLotteryPrice(lotteryID int) float64 {
	var price float64
	db.DB.QueryRow("SELECT price FROM lotteries WHERE id = $1", lotteryID).Scan(&price)
	return price
}

func generateWinningNumbers() []int {
	rand.Seed(time.Now().UnixNano())
	numbers := rand.Perm(49)[:6] // Generate numbers from 1 to 49
	sort.Ints(numbers)           // Ensure sorted order
	return numbers
}

func numbersToString(numbers []int) string {
	var strNumbers []string
	for _, n := range numbers {
		strNumbers = append(strNumbers, strconv.Itoa(n))
	}
	return strings.Join(strNumbers, ",")
}

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

func StartDrawScheduler() {
	ticker := time.NewTicker(1 * time.Minute)
	go func() {
		for range ticker.C {
			CheckForExpiredLotteries()
		}
	}()
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
