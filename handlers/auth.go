package handlers

import (
	"html/template"
	"log"
	"loto/db"
	"loto/models"
	"net/http"

	"golang.org/x/crypto/bcrypt"
)

type LoginData struct {
	Error string
}

type RegisterData struct {
	Error string
}

var tmpl = template.Must(template.ParseGlob("templates/*.html"))

func HomeHandler(w http.ResponseWriter, r *http.Request) {
	tmpl.ExecuteTemplate(w, "home.html", nil)
}

func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		username := r.FormValue("username")
		password := r.FormValue("password")

		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			data := RegisterData{Error: "Server error, unable to create your account."}
			tmpl.ExecuteTemplate(w, "register.html", data) // Render with error
			return
		}

		_, err = db.DB.Exec("INSERT INTO users (username, password) VALUES ($1, $2)",
			username, string(hashedPassword))
		if err != nil {
			log.Println("Error inserting user:", err)
			data := RegisterData{Error: "Unable to register. Username might be taken."} // More specific error
			tmpl.ExecuteTemplate(w, "register.html", data)                              // Render with error
			return
		}

		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	tmpl.ExecuteTemplate(w, "register.html", nil)
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		username := r.FormValue("username")
		password := r.FormValue("password")

		row := db.DB.QueryRow("SELECT id, password FROM users WHERE username = $1", username)
		var user models.User
		err := row.Scan(&user.ID, &user.Password)
		if err != nil {
			data := LoginData{Error: "Username or password is incorrect"}
			tmpl.ExecuteTemplate(w, "login.html", data)
			return
		}

		if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
			data := LoginData{Error: "Username or password is incorrect"}
			tmpl.ExecuteTemplate(w, "login.html", data)
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:  "username",
			Value: username,
			Path:  "/",
		})

		http.Redirect(w, r, "/lotteries", http.StatusSeeOther)
		return
	}

	tmpl.ExecuteTemplate(w, "login.html", nil)
}

func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:   "username",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func isAdmin(r *http.Request) bool {
	cookie, err := r.Cookie("username")
	if err != nil || cookie.Value != "admin" {
		return false
	}
	return true
}
