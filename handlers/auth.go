// =============================================================================
// HANDLERS - Auth
// =============================================================================

package handlers

import (
	"database/sql"
	"fmt"
	"html/template"
	"net/http"
	"noteflow/database"
	"noteflow/models"

	"golang.org/x/crypto/bcrypt"
)

func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		tmpl := template.Must(template.ParseFiles("templates/register.html"))
		tmpl.Execute(w, nil)
		return
	}

	if r.Method == "POST" {
		r.ParseForm()
		nom := r.FormValue("nom")
		email := r.FormValue("email")
		password := r.FormValue("password")

		// Hash du mot de passe
		hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			http.Error(w, "Erreur création compte", 500)
			return
		}

		// Insertion en base
		_, err = database.DB.Exec("INSERT INTO users (nom, email, password_hash) VALUES (?, ?, ?)",
			nom, email, string(hash))
		if err != nil {
			http.Error(w, "Email déjà utilisé", 400)
			return
		}

		http.Redirect(w, r, "/login", http.StatusSeeOther)
	}
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		tmpl := template.Must(template.ParseFiles("templates/login.html"))
		tmpl.Execute(w, nil)
		return
	}

	if r.Method == "POST" {
		r.ParseForm()
		email := r.FormValue("email")
		password := r.FormValue("password")

		var user models.User
		err := database.DB.QueryRow("SELECT id, nom, email, password_hash FROM users WHERE email = ?", email).
			Scan(&user.ID, &user.Nom, &user.Email, &user.PasswordHash)

		if err == sql.ErrNoRows {
			http.Error(w, "Email ou mot de passe incorrect", 401)
			return
		}

		// Vérifier le mot de passe
		err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
		if err != nil {
			http.Error(w, "Email ou mot de passe incorrect", 401)
			return
		}

		// Créer session
		CreateSession(w, user.ID)
		http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
	}
}

func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	DestroySession(w)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func HomeHandler(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// Sessions simples avec cookies
func CreateSession(w http.ResponseWriter, userID int) {
	cookie := &http.Cookie{
		Name:     "session",
		Value:    fmt.Sprintf("%d", userID),
		Path:     "/",
		MaxAge:   86400, // 24h
		HttpOnly: true,
	}
	http.SetCookie(w, cookie)
}

func DestroySession(w http.ResponseWriter) {
	cookie := &http.Cookie{
		Name:   "session",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	}
	http.SetCookie(w, cookie)
}
