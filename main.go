// main.go
package main

import (
	"log"
	"net/http"
	"noteflow/database"
	"noteflow/handlers"
	"os"
)

func main() {
	// Créer les dossiers nécessaires
	createDirs()

	// Initialiser la base de données
	db, err := database.InitDB()
	if err != nil {
		log.Fatal("Erreur initialisation DB:", err)
	}
	defer db.Close()

	// Routes publiques
	http.HandleFunc("/", handlers.HomeHandler)

	// Routes CSV et bulletins
	http.HandleFunc("/csv/upload", handlers.UploadCSVHandler)
	http.HandleFunc("/bulletins/download/", handlers.DownloadBulletinHandler)
	http.HandleFunc("/bulletins/zip/", handlers.DownloadZipHandler)

	// Servir les fichiers statiques
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	// Servir les bulletins PDF
	bulletinFS := http.FileServer(http.Dir("./bulletins"))
	http.Handle("/bulletins-files/", http.StripPrefix("/bulletins-files/", bulletinFS))

	log.Println("🚀 Serveur démarré sur http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func createDirs() {
	dirs := []string{"uploads", "bulletins", "static/css", "templates"}
	for _, dir := range dirs {
		os.MkdirAll(dir, 0755)
	}
}
