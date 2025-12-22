// =============================================================================
// HANDLERS - Téléchargement bulletins
// =============================================================================

package handlers

import (
	"archive/zip"
	"fmt"
	"net/http"
	"noteflow/database"
	"os"
	"strconv"
	"strings"
)

func DownloadBulletinHandler(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/bulletins/download/")
	eleveID, err := strconv.Atoi(path)
	if err != nil {
		http.Error(w, "ID invalide", 400)
		return
	}

	// Récupérer le bulletin
	var pdfPath string
	err = database.DB.QueryRow(`
		SELECT pdf_path 
		FROM bulletins 
		WHERE eleve_id = ?`, eleveID).
		Scan(&pdfPath)

	if err != nil {
		http.Error(w, "Bulletin introuvable", 404)
		return
	}

	// Vérifier que le fichier existe
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		http.Error(w, "Fichier PDF introuvable", 404)
		return
	}

	// Servir le fichier
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=bulletin_%d.pdf", eleveID))
	http.ServeFile(w, r, pdfPath)
}

func DownloadZipHandler(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/bulletins/zip/")
	classeID, err := strconv.Atoi(path)
	if err != nil {
		http.Error(w, "ID invalide", 400)
		return
	}

	// Vérifier que la classe existe
	var count int
	err = database.DB.QueryRow("SELECT COUNT(*) FROM classes WHERE id = ?", classeID).Scan(&count)
	if err != nil || count == 0 {
		http.Error(w, "Classe introuvable", 404)
		return
	}

	// Récupérer tous les bulletins de la classe
	rows, err := database.DB.Query(`
		SELECT e.prenom, e.nom, b.pdf_path 
		FROM bulletins b
		JOIN eleves e ON b.eleve_id = e.id
		WHERE b.classe_id = ?`, classeID)
	if err != nil {
		http.Error(w, "Erreur récupération bulletins", 500)
		return
	}
	defer rows.Close()

	// Créer un ZIP en mémoire
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=bulletins_classe_%d.zip", classeID))

	zipWriter := zip.NewWriter(w)
	defer zipWriter.Close()

	fileCount := 0
	for rows.Next() {
		var prenom, nom, pdfPath string
		err := rows.Scan(&prenom, &nom, &pdfPath)
		if err != nil {
			continue
		}

		// Vérifier que le fichier existe
		if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
			continue
		}

		// Lire le PDF
		pdfData, err := os.ReadFile(pdfPath)
		if err != nil {
			continue
		}

		// Ajouter au ZIP
		filename := fmt.Sprintf("%s_%s.pdf", nom, prenom)
		zipFile, err := zipWriter.Create(filename)
		if err != nil {
			continue
		}

		_, err = zipFile.Write(pdfData)
		if err != nil {
			continue
		}

		fileCount++
	}

	if fileCount == 0 {
		http.Error(w, "Aucun bulletin trouvé pour cette classe", 404)
		return
	}
}
