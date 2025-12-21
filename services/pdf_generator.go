package services

// =============================================================================
// SERVICE - Génération PDF
// =============================================================================

import (
	"bytes"
	"fmt"
	"html/template"
	"noteflow/models"
	"os"
	"os/exec"
	"path/filepath"
)

func GeneratePDF(data models.BulletinData, eleveID int) (string, error) {
	// Vérifier que wkhtmltopdf est disponible
	fmt.Printf("[DEBUG] Début génération PDF pour élève %d\n", eleveID)
	_, err := exec.LookPath("wkhtmltopdf")
	if err != nil {
		fmt.Printf("[DEBUG] wkhtmltopdf non trouvé: %v\n", err)
		return "", fmt.Errorf("wkhtmltopdf n'est pas installé. Veuillez l'installer pour générer les PDFs")
	}

	// Charger le template de bulletin
	fmt.Printf("[DEBUG] Chargement du template bulletin_template.html...\n")
	tmpl, err := template.ParseFiles("templates/bulletin_template.html")
	if err != nil {
		fmt.Printf("[DEBUG] Erreur chargement template: %v\n", err)
		return "", fmt.Errorf("erreur chargement template: %v", err)
	}

	// Générer le HTML
	var htmlBuffer bytes.Buffer
	fmt.Printf("[DEBUG] Exécution du template avec les données pour élève %d...\n", eleveID)
	err = tmpl.Execute(&htmlBuffer, data)
	if err != nil {
		fmt.Printf("[DEBUG] Erreur génération HTML: %v\n", err)
		return "", fmt.Errorf("erreur génération HTML: %v", err)
	}

	// Créer un fichier HTML temporaire
	tmpHTML := filepath.Join("uploads", fmt.Sprintf("bulletin_%d.html", eleveID))
	fmt.Printf("[DEBUG] Écriture du fichier HTML temporaire: %s\n", tmpHTML)
	err = os.WriteFile(tmpHTML, htmlBuffer.Bytes(), 0644)
	if err != nil {
		fmt.Printf("[DEBUG] Erreur écriture fichier temporaire: %v\n", err)
		return "", fmt.Errorf("erreur écriture fichier temporaire: %v", err)
	}
	defer os.Remove(tmpHTML) // Nettoyer même en cas d'erreur

	// Générer le PDF avec wkhtmltopdf
	pdfPath := filepath.Join("bulletins", fmt.Sprintf("bulletin_%d.pdf", eleveID))
	fmt.Printf("[DEBUG] Création du dossier bulletins si besoin...\n")
	os.MkdirAll("bulletins", 0755)
	fmt.Printf("[DEBUG] Génération du PDF avec wkhtmltopdf : %s\n", pdfPath)
	cmd := exec.Command("wkhtmltopdf",
		"--page-size", "A4",
		"--margin-top", "10mm",
		"--margin-bottom", "10mm",
		"--margin-left", "10mm",
		"--margin-right", "10mm",
		"--encoding", "UTF-8",
		"--quiet",
		tmpHTML, pdfPath)

	// Capturer la sortie d'erreur pour le débogage
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		fmt.Printf("[DEBUG] Erreur génération PDF avec wkhtmltopdf: %v\n[stderr]: %s\n", err, stderr.String())
		return "", fmt.Errorf("erreur génération PDF: %v (stderr: %s)", err, stderr.String())
	}
	fmt.Printf("[DEBUG] PDF généré avec succès : %s\n", pdfPath)

	// Vérifier que le fichier PDF a été créé
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		fmt.Printf("[DEBUG] Le fichier PDF n'a pas été créé !\n")
		return "", fmt.Errorf("le fichier PDF n'a pas été créé")
	}

	return pdfPath, nil
}
