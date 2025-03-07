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
	"encoding/base64"

)

func authenticateUser(r *http.Request) bool {
	cookie, err := r.Cookie("username")
	return err == nil && cookie.Value != ""
}

func LotteriesHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("username")
	if err != nil || cookie.Value == "" {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	rows, err := db.DB.Query("SELECT id, name, description, price, end_date FROM lotteries WHERE status = 'active'")
if err != nil {
    log.Printf("Error fetching lotteries: %v", err)
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

func normalizeNumbers(numbers string) string {
	numArr := strings.Split(numbers, ",")
	nums := make([]int, len(numArr))

	for i, numStr := range numArr {
		num, err := strconv.Atoi(strings.TrimSpace(numStr))
		if err != nil {
			return "" // Return empty string if conversion fails (invalid input)
		}
		nums[i] = num
	}

	sort.Ints(nums)

	strArr := make([]string, len(nums))
	for i, num := range nums {
		strArr[i] = strconv.Itoa(num)
	}

	return strings.Join(strArr, ",") // Return sorted numbers as a single string
}

func BuyLotteryHandler(w http.ResponseWriter, r *http.Request) {
	if !authenticateUser(r) {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	cookie, err := r.Cookie("username")
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	username := cookie.Value

	if r.Method == http.MethodPost {
		lotteryID := r.FormValue("lottery_id")
		chosenNumbers := r.FormValue("chosen_numbers")

		var balance float64
		err = db.DB.QueryRow("SELECT balance FROM users WHERE username = $1", username).Scan(&balance)
		if err != nil {
			http.Error(w, "Unable to fetch balance", http.StatusInternalServerError)
			return
		}

		var lotteryPrice float64
		err = db.DB.QueryRow("SELECT price FROM lotteries WHERE id = $1", lotteryID).Scan(&lotteryPrice)
		if err != nil {
			http.Error(w, "Unable to fetch lottery price", http.StatusInternalServerError)
			return
		}

		if balance < lotteryPrice {
			http.Redirect(w, r, "/add-card", http.StatusSeeOther)
			return
		}

		if !validateLotteryNumbers(chosenNumbers) {
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

			data := map[string]interface{}{
				"Error":     "Invalid numbers. Choose 6 unique numbers between 1 and 49.",
				"Lotteries": lotteries,
				"Username":  username,
			}
			tmpl.ExecuteTemplate(w, "lotteries.html", data)
			return
		}

		normalizedNumbers := normalizeNumbers(chosenNumbers)

		var count int
		err = db.DB.QueryRow(`
			SELECT COUNT(*) 
			FROM purchases 
			WHERE username = $1 
			  AND lottery_id = $2 
			  AND chosen_numbers = $3
		`, username, lotteryID, normalizedNumbers).Scan(&count)
		if err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		if count > 0 {
			data := map[string]interface{}{
				"Error":    "You already purchased these exact numbers for this lottery!",
				"Username": username,
			}
			tmpl.ExecuteTemplate(w, "lotteries.html", data)
			return
		}

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

		_, err = tx.Exec("UPDATE users SET balance = balance - $1 WHERE username = $2", lotteryPrice, username)
		if err != nil {
			tx.Rollback()
			log.Println("Error updating balance:", err)
			data := map[string]interface{}{
				"Error": "Unable to update balance",
			}
			tmpl.ExecuteTemplate(w, "lotteries.html", data)
			return
		}

		if err := tx.Commit(); err != nil {
			log.Println("Error committing transaction:", err)
			data := map[string]interface{}{
				"Error": "Transaction commit failed",
			}
			tmpl.ExecuteTemplate(w, "lotteries.html", data)
			return
		}

		http.Redirect(w, r, "/lotteries", http.StatusSeeOther)
	}
}


	


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

func RemoveExpiredLotteries() {
	for {
		time.Sleep(1 * time.Minute)

		_, err := db.DB.Exec("UPDATE lotteries SET status = 'ended' WHERE end_date < NOW() AND status = 'active'")
		if err != nil {
			log.Println("Error updating expired lotteries:", err)
		}
	}
}

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


func AddCardHandler(w http.ResponseWriter, r *http.Request) {
    cookie, err := r.Cookie("username")
    if err != nil || cookie.Value == "" {
        http.Redirect(w, r, "/login", http.StatusSeeOther)
        return
    }
    username := cookie.Value

    if r.Method == http.MethodPost {
        cardNumber := r.FormValue("card_number")
        expiry := r.FormValue("expiry")
        cvv := r.FormValue("cvv")
        amountStr := r.FormValue("amount")

        amount, err := strconv.ParseFloat(amountStr, 64)
        if err != nil || amount <= 0 {
            data := map[string]interface{}{
                "ErrorMsg":         "Invalid amount. Please enter a valid number > 0.",
                "ShowPaymentModal": true,
            }
            fillProfileData(username, data)
            tmpl.ExecuteTemplate(w, "profile.html", data)
            return
        }

		cardNumber = r.FormValue("card_number")
		cardNumber = strings.ReplaceAll(cardNumber, " ", "")

        _, err = db.DB.Exec(
            "INSERT INTO payment_cards (username, card_number, expiry, cvv, amount) VALUES ($1, $2, $3, $4, $5)",
            username, cardNumber, expiry, cvv, amount)
        	if err != nil {
				log.Println("Error inserting payment card:", err)
				data := map[string]interface{}{
					"ErrorMsg":         "Unable to add card. Please try again.",
					"ShowPaymentModal": true,
				}
				fillProfileData(username, data)
				tmpl.ExecuteTemplate(w, "profile.html", data)
				return
        	}

        _, err = db.DB.Exec("UPDATE users SET balance = balance + $1 WHERE username = $2", amount, username)
        if err != nil {
            log.Println("Error updating balance:", err)
            data := map[string]interface{}{
                "ErrorMsg":         "Unable to update balance.",
                "ShowPaymentModal": true,
            }
            fillProfileData(username, data)
            tmpl.ExecuteTemplate(w, "profile.html", data)
            return
        }

        http.Redirect(w, r, "/profile", http.StatusSeeOther)
        return
    }

    http.Redirect(w, r, "/profile", http.StatusSeeOther)
}

func fillProfileData(username string, data map[string]interface{}) {
    row := db.DB.QueryRow("SELECT id, password, balance, profile_picture FROM users WHERE username=$1", username)
    var user models.User
    user.Username = username
    err := row.Scan(&user.ID, &user.Password, &user.Balance, &user.ProfilePicture)
    if err != nil {
        log.Println("Error fetching user in fillProfileData:", err)
        return
    }

    var base64Img string
    if len(user.ProfilePicture) > 0 {
        encoded := base64.StdEncoding.EncodeToString(user.ProfilePicture)
        base64Img = "data:image/jpeg;base64," + encoded
    }

    data["User"] = user
    data["ImageData"] = base64Img
}

