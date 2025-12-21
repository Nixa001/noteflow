package middleware


import (
	"net/http"
	"strconv"
)

func AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session")
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		userID, err := strconv.Atoi(cookie.Value)
		if err != nil || userID == 0 {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		next(w, r)
	}
}

func GetCurrentUserID(r *http.Request) int {
	cookie, _ := r.Cookie("session")
	userID, _ := strconv.Atoi(cookie.Value)
	return userID
}