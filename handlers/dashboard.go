// HANDLERS - Dashboard et Classes
package handlers

import (
	"database/sql"
	"html/template"
	"net/http"
	"noteflow/database"
	"noteflow/models"
	"strconv"
	"strings"
)

func DashboardHandler(w http.ResponseWriter, r *http.Request) {
	// Récupérer les classes
	rows, err := database.DB.Query(`
		SELECT id, nom, ecole, annee_scolaire, maitre, created_at 
		FROM classes 
		ORDER BY created_at DESC`)
	if err != nil {
		http.Error(w, "Erreur chargement classes", 500)
		return
	}
	defer rows.Close()

	var classes []models.Classe
	for rows.Next() {
		var c models.Classe
		rows.Scan(&c.ID, &c.Nom, &c.Ecole, &c.AnneeScolaire, &c.Maitre, &c.CreatedAt)
		classes = append(classes, c)
	}

	tmpl := template.Must(template.ParseFiles("templates/dashboard.html"))
	tmpl.Execute(w, map[string]interface{}{
		"Classes": classes,
	})
}

func NewClasseHandler(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles("templates/classe_new.html"))
	tmpl.Execute(w, nil)
}

func CreateClasseHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
		return
	}

	r.ParseForm()
	nom := r.FormValue("nom")
	ecole := r.FormValue("ecole")
	annee := r.FormValue("annee_scolaire")
	maitre := r.FormValue("maitre")
	trimestre := r.FormValue("trimestre")

	_, err := database.DB.Exec(`
        INSERT INTO classes (nom, ecole, annee_scolaire, maitre, trimestre, user_id) 
        VALUES (?, ?, ?, ?, ?, ?)`,
		nom, ecole, annee, maitre, trimestre, 0)

	if err != nil {
		http.Error(w, "Erreur création classe", 500)
		return
	}

	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

func ClasseDetailHandler(w http.ResponseWriter, r *http.Request) {
	// Extraire l'ID de la classe depuis l'URL
	path := strings.TrimPrefix(r.URL.Path, "/classes/")
	classeID, err := strconv.Atoi(path)
	if err != nil {
		http.Error(w, "ID classe invalide", 400)
		return
	}

	// Récupérer la classe
	var classe models.Classe
	err = database.DB.QueryRow(`
		SELECT id, nom, ecole, annee_scolaire, maitre 
		FROM classes 
		WHERE id = ?`, classeID).
		Scan(&classe.ID, &classe.Nom, &classe.Ecole, &classe.AnneeScolaire, &classe.Maitre)

	if err != nil {
		http.Error(w, "Classe introuvable", 404)
		return
	}

	// Récupérer les élèves avec leurs notes et bulletins
	rows, err := database.DB.Query(`
		SELECT e.id, e.prenom, e.nom, 
			   n.r_lc, n.c_lc, n.dm, n.r_math, n.c_math, n.edd, n.arabe, n.dessin,
			   n.total, n.moyenne, n.rang,
			   b.pdf_path
		FROM eleves e
		LEFT JOIN notes n ON e.id = n.eleve_id
		LEFT JOIN bulletins b ON e.id = b.eleve_id
		WHERE e.classe_id = ?
		ORDER BY n.rang ASC`, classeID)

	if err != nil {
		http.Error(w, "Erreur chargement élèves", 500)
		return
	}
	defer rows.Close()

	type EleveDetail struct {
		models.Eleve
		models.Notes
		PDFPath string
	}

	var eleves []EleveDetail
	for rows.Next() {
		var ed EleveDetail
		var pdfPath sql.NullString
		rows.Scan(&ed.Eleve.ID, &ed.Prenom, &ed.Nom,
			&ed.RLC, &ed.CLC, &ed.DM, &ed.RMath, &ed.CMath, &ed.EDD, &ed.Arabe, &ed.Dessin,
			&ed.Total, &ed.Moyenne, &ed.Rang, &pdfPath)
		if pdfPath.Valid {
			ed.PDFPath = pdfPath.String
		}
		eleves = append(eleves, ed)
	}

	tmpl := template.Must(template.ParseFiles("templates/classe_detail.html"))
	tmpl.Execute(w, map[string]interface{}{
		"Classe": classe,
		"Eleves": eleves,
	})
}
