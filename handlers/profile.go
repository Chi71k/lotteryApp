package handlers

import (
	"bytes"
	"encoding/base64"
	"io"
	"log"
	"loto/db"
	"loto/models"
	"net/http"
	"strconv"
)

func ProfileHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("username")
	if err != nil || cookie.Value == "" {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	username := cookie.Value

	if r.Method == http.MethodPost {
		file, header, err := r.FormFile("profile_picture")
		log.Println("ProfileHandler POST: trying to upload file")
		if err != nil {
			log.Println("Error retrieving file:", err)
		}
		if err == nil && header != nil {
			var buf bytes.Buffer
			_, err := io.Copy(&buf, file)
			if err != nil {
				log.Println("Error reading file:", err)
				data := map[string]interface{}{
					"Error": "Unable to read file",
				}
				tmpl.ExecuteTemplate(w, "profile.html", data)
				return
			}
			_, err = db.DB.Exec("UPDATE users SET profile_picture = $1 WHERE username = $2", buf.Bytes(), username)
			if err != nil {
				log.Println("Error updating profile picture:", err)
				data := map[string]interface{}{
					"Error": "Unable to update profile picture",
				}
				tmpl.ExecuteTemplate(w, "profile.html", data)
				return
			}
		}

		topUpStr := r.FormValue("topup_amount")
		if topUpStr != "" {
			amount, err := strconv.ParseFloat(topUpStr, 64)
			if err == nil && amount > 0 {
				_, err := db.DB.Exec("UPDATE users SET balance = balance + $1 WHERE username = $2", amount, username)
				if err != nil {
					log.Println("Error updating balance:", err)
					data := map[string]interface{}{
						"Error": "Unable to update balance",
					}
					tmpl.ExecuteTemplate(w, "profile.html", data)
					return
				}
			}
		}

		http.Redirect(w, r, "/profile", http.StatusSeeOther)
		return
	}

	row := db.DB.QueryRow("SELECT id, password, balance, profile_picture FROM users WHERE username = $1", username)

	var user models.User
	user.Username = username
	err = row.Scan(&user.ID, &user.Password, &user.Balance, &user.ProfilePicture)
	if err != nil {
		log.Println("Error fetching user:", err)
		data := map[string]interface{}{
			"Error": "Unable to fetch user",
		}
		tmpl.ExecuteTemplate(w, "profile.html", data)
		return
	}

	var base64Img string
	if len(user.ProfilePicture) > 0 {
		encoded := base64.StdEncoding.EncodeToString(user.ProfilePicture)
		base64Img = "data:image/jpeg;base64," + encoded
	}

	data := map[string]interface{}{
		"User":      user,
		"ImageData": base64Img,
	}

	tmpl.ExecuteTemplate(w, "profile.html", data)
}

