package handlers

import (
	"html/template"
	"log"
	"net/http"

	"golang.org/x/crypto/bcrypt"

	"loto/db"
	"loto/models"
)

// Для вывода ошибок при логине
type LoginData struct {
	Error string
}

// Простая "сессия" через map (демонстрационный вариант)
var session = map[string]string{}

// Подгружаем шаблоны
var tmpl = template.Must(template.ParseGlob("templates/*.html"))

// Главная страница (/)
func HomeHandler(w http.ResponseWriter, r *http.Request) {
	tmpl.ExecuteTemplate(w, "home.html", nil)
}

// Регистрация
func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		username := r.FormValue("username")
		password := r.FormValue("password")

		// Хэшируем пароль
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			http.Error(w, "Server error, unable to create your account.", http.StatusInternalServerError)
			return
		}

		// Сохраняем пользователя в БД
		_, err = db.DB.Exec("INSERT INTO users (username, password) VALUES ($1, $2)",
			username, string(hashedPassword))
		if err != nil {
			log.Println("Error inserting user:", err)
			http.Error(w, "Unable to register", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// Если GET-запрос — показать форму регистрации
	tmpl.ExecuteTemplate(w, "register.html", nil)
}

// Логин
// ...existing code...

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		username := r.FormValue("username")
		password := r.FormValue("password")

		// Ищем пользователя в базе
		row := db.DB.QueryRow("SELECT id, password FROM users WHERE username = $1", username)
		var user models.User
		err := row.Scan(&user.ID, &user.Password)
		if err != nil {
			data := LoginData{Error: "Username or password is incorrect"}
			tmpl.ExecuteTemplate(w, "login.html", data)
			return
		}

		// Сверяем пароль
		if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
			data := LoginData{Error: "Username or password is incorrect"}
			tmpl.ExecuteTemplate(w, "login.html", data)
			return
		}

		// Всё ок — ставим cookie
		http.SetCookie(w, &http.Cookie{
			Name:  "username",
			Value: username,
			Path:  "/",
			// HttpOnly: true, Secure: true и т.п. (по потребности)
		})

		// Редирект на список лотерей
		http.Redirect(w, r, "/lotteries", http.StatusSeeOther)
		return
	}

	// Если GET
	tmpl.ExecuteTemplate(w, "login.html", nil)
}

// Выход (логаут)
func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:   "username",
		Value:  "",
		Path:   "/",
		MaxAge: -1, // удалить cookie
	})
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
