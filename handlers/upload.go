// =============================================================================
// HANDLERS - Upload CSV et génération bulletins
// =============================================================================

package handlers

import (
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"noteflow/database"
	"noteflow/middleware"
	"noteflow/models"
	"noteflow/services"
	"strconv"
	"strings"
	"time"
)

func UploadCSVHandler(w http.ResponseWriter, r *http.Request) {
	// Slice pour stocker les bulletins à générer
	type bulletinGen struct {
		data    models.BulletinData
		eleveID int
	}
	var bulletinsToGenerate []bulletinGen
	if r.Method != "POST" {
		http.Error(w, "Méthode non autorisée", 405)
		return
	}

	// Récupérer l'ID de la classe
	classeID, err := strconv.Atoi(r.FormValue("classe_id"))
	if err != nil {
		http.Error(w, "ID classe invalide", 400)
		return
	}

	// Vérifier la propriété de la classe et récupérer le nom du maître
	userID := middleware.GetCurrentUserID(r)
	var count int
	var maitreNom string
	err = database.DB.QueryRow("SELECT COUNT(*), (SELECT nom FROM users WHERE id = ?) FROM classes WHERE id = ? AND user_id = ?", userID, classeID, userID).Scan(&count, &maitreNom)
	if err != nil || count == 0 {
		http.Error(w, "Accès refusé", 403)
		return
	}

	// Parser le fichier CSV
	file, _, err := r.FormFile("csv_file")
	if err != nil {
		http.Error(w, "Erreur lecture fichier: "+err.Error(), 400)
		return
	}
	defer file.Close()

	// Lire la première ligne pour détecter le séparateur
	firstLineBytes := make([]byte, 1024)
	n, err := file.Read(firstLineBytes)
	if err != nil && err != io.EOF {
		http.Error(w, "Erreur lecture CSV", 400)
		return
	}
	firstLine := string(firstLineBytes[:n])
	separator := detectCSVSeparator(firstLine)

	// Réinitialiser le fichier pour le parser
	file.Seek(0, 0)

	// Lire et parser le CSV
	reader := csv.NewReader(file)
	reader.Comma = separator
	reader.LazyQuotes = true
	reader.TrimLeadingSpace = true

	// Lire l'en-tête
	headers, err := reader.Read()
	if err != nil {
		http.Error(w, "Erreur lecture CSV: "+err.Error(), 400)
		return
	}

	// Normaliser les en-têtes (supprimer espaces, convertir en minuscules)
	normalizedHeaders := make([]string, len(headers))
	for i, h := range headers {
		normalizedHeaders[i] = strings.TrimSpace(strings.ToLower(h))
	}

	// Créer un map flexible pour trouver l'index de chaque colonne
	headerMap := make(map[string]int)

	// Patterns de recherche pour chaque colonne
	columnPatterns := map[string][]string{
		"prenom":       {"prénom", "prenom", "prénoms"},
		"nom":          {"nom"},
		"r_lc":         {"r. lc", "r_lc", "r lc", "r.lc", "rlc"},
		"c_lc":         {"c. lc", "c_lc", "c lc", "c.lc", "clc"},
		"dm":           {"dm", "dictée et mathématiques"},
		"r_math":       {"r.math", "r_math", "r math", "rmath", "raisonnement mathématiques"},
		"c_math":       {"c. math", "c_math", "c math", "cmath", "calcul mathématiques"},
		"edd":          {"edd", "éveil et découverte"},
		"arabe":        {"arabe"},
		"dessin":       {"dessin", "arts plastiques"},
		"total":        {"total", "total des points"},
		"moyenne":      {"moyenne", "moyenne générale"},
		"rang":         {"rang", "rang dans la classe"},
		"appreciation": {"appréciation", "appreciation", "bulletin"},
	}

	// D'abord chercher "prenom" pour éviter les conflits avec "nom"
	for i, header := range normalizedHeaders {
		if strings.Contains(header, "prénom") || strings.Contains(header, "prenom") {
			headerMap["prenom"] = i
			break
		}
	}

	// Ensuite chercher "nom" en excluant la colonne prénom déjà trouvée
	prenomIdx := -1
	if idx, ok := headerMap["prenom"]; ok {
		prenomIdx = idx
	}
	for i, header := range normalizedHeaders {
		// Chercher "nom" mais pas dans la colonne prénom et pas "prénom"/"prenom"
		if i != prenomIdx && header == "nom" {
			headerMap["nom"] = i
			break
		}
	}

	// Chercher les autres colonnes
	for colName, patterns := range columnPatterns {
		// Skip prenom et nom car déjà traités
		if colName == "prenom" || colName == "nom" {
			continue
		}

		found := false
		for i, header := range normalizedHeaders {
			for _, pattern := range patterns {
				if strings.Contains(header, pattern) {
					headerMap[colName] = i
					found = true
					break
				}
			}
			if found {
				break
			}
		}
	}

	// Debug: afficher les colonnes détectées
	fmt.Printf("DEBUG - En-têtes normalisés: %v\n", normalizedHeaders)
	fmt.Printf("DEBUG - Colonnes détectées: %v\n", headerMap)

	// Vérifier qu'on a trouvé les colonnes essentielles
	essentialCols := []string{"nom", "r_lc", "c_lc", "dm", "r_math", "c_math", "edd", "arabe", "dessin", "total", "moyenne", "rang"}
	missingCols := []string{}
	for _, col := range essentialCols {
		if _, ok := headerMap[col]; !ok {
			missingCols = append(missingCols, col)
		}
	}
	if len(missingCols) > 0 {
		http.Error(w, fmt.Sprintf("Colonnes manquantes dans le CSV: %v. En-têtes reçus: %v. Colonnes détectées: %v", missingCols, normalizedHeaders, headerMap), 400)
		return
	}

	// Vérifier qu'on a au moins prénom OU nom détecté
	if _, prenomOk := headerMap["prenom"]; !prenomOk {
		if _, nomOk := headerMap["nom"]; nomOk {
			nomIdx := headerMap["nom"]
			if nomIdx > 0 {
				headerMap["prenom"] = nomIdx - 1
			}
		} else {
			headerMap["prenom"] = 0
			headerMap["nom"] = 1
		}
	}

	// Récupérer les infos de la classe pour les bulletins
	var classe models.Classe
	err = database.DB.QueryRow("SELECT nom, ecole, annee_scolaire FROM classes WHERE id = ?", classeID).
		Scan(&classe.Nom, &classe.Ecole, &classe.AnneeScolaire)
	if err != nil {
		http.Error(w, "Erreur récupération classe", 500)
		return
	}

	successCount := 0
	errorCount := 0
	effectif := 0
	maxRang := 0

	// Traiter chaque élève
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			errorCount++
			continue
		}

		// Ignorer les lignes vides
		isEmpty := true
		for _, field := range record {
			if strings.TrimSpace(field) != "" {
				isEmpty = false
				break
			}
		}
		if isEmpty {
			continue
		}

		// Ignorer les lignes de statistiques (G:, F:, T:)
		firstField := strings.TrimSpace(record[0])
		if strings.HasPrefix(firstField, "G:") || strings.HasPrefix(firstField, "F:") ||
			strings.HasPrefix(firstField, "T:") || strings.HasPrefix(firstField, "g:") ||
			strings.HasPrefix(firstField, "f:") || strings.HasPrefix(firstField, "t:") {
			continue
		}

		// Extraire les valeurs en utilisant le map
		getValue := func(key string) string {
			if idx, ok := headerMap[key]; ok && idx < len(record) {
				return strings.TrimSpace(record[idx])
			}
			return ""
		}

		// Extraction prénom et nom
		prenom := ""
		nom := ""

		// Extraire le prénom de la colonne prenom
		if idx, ok := headerMap["prenom"]; ok && idx < len(record) {
			prenom = strings.TrimSpace(record[idx])
		}

		// Extraire le nom de la colonne nom
		if idx, ok := headerMap["nom"]; ok && idx < len(record) {
			nom = strings.TrimSpace(record[idx])
		}

		// Si nom est vide ET qu'on a un prénom avec plusieurs mots
		// (ex: colonne unique "Prénoms et Nom" avec "Paul L. Balossa")
		if nom == "" && prenom != "" {
			parts := strings.Fields(prenom)
			if len(parts) >= 2 {
				nom = parts[len(parts)-1]
				prenom = strings.Join(parts[:len(parts)-1], " ")
			} else {
				// Si un seul mot dans prenom et pas de nom, ignorer
				errorCount++
				continue
			}
		}

		// Si après tout ça le nom est toujours vide, ignorer la ligne
		if nom == "" || prenom == "" {
			errorCount++
			continue
		}

		// CORRECTION: Extraire l'appréciation du CSV
		appreciation := getValue("appreciation")

		// Si pas d'appréciation dans le CSV, utiliser la mention par défaut
		if appreciation == "" {
			moyenne := parseFloatSafe(getValue("moyenne"))
			appreciation = getMention(moyenne)
		}

		// Insérer l'élève
		result, err := database.DB.Exec("INSERT INTO eleves (prenom, nom, classe_id) VALUES (?, ?, ?)",
			prenom, nom, classeID)
		if err != nil {
			errorCount++
			continue
		}

		eleveID, _ := result.LastInsertId()

		// Parser les notes avec gestion d'erreurs
		rLC := parseFloatSafe(getValue("r_lc"))
		cLC := parseFloatSafe(getValue("c_lc"))
		dm := parseFloatSafe(getValue("dm"))
		rMath := parseFloatSafe(getValue("r_math"))
		cMath := parseFloatSafe(getValue("c_math"))
		edd := parseFloatSafe(getValue("edd"))
		arabe := parseFloatSafe(getValue("arabe"))
		dessin := parseFloatSafe(getValue("dessin"))
		total := parseFloatSafe(getValue("total"))
		moyenne := parseFloatSafe(getValue("moyenne"))
		rang := parseIntSafe(getValue("rang"))

		// Calcul effectif et max rang
		effectif++
		if rang > maxRang {
			maxRang = rang
		}

		// Insérer les notes
		_, err = database.DB.Exec(`INSERT INTO notes (eleve_id, r_lc, c_lc, dm, r_math, c_math, edd, arabe, dessin, total, moyenne, rang)
			       VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			eleveID, rLC, cLC, dm, rMath, cMath, edd, arabe, dessin, total, moyenne, rang)
		if err != nil {
			errorCount++
			continue
		}

		// CORRECTION: Utiliser l'appréciation extraite du CSV dans le bulletin
		// On ne génère pas le PDF ici, on stocke les bulletins à générer
		bulletinsToGenerate = append(bulletinsToGenerate, struct {
			data    models.BulletinData
			eleveID int
		}{
			data: models.BulletinData{
				Ecole:         classe.Ecole,
				Classe:        classe.Nom,
				AnneeScolaire: classe.AnneeScolaire,
				Effectif:      0, // sera corrigé après
				Maitre:        maitreNom,
				Prenom:        prenom,
				Nom:           nom,
				Notes: models.Notes{
					RLC:     rLC,
					CLC:     cLC,
					DM:      dm,
					RMath:   rMath,
					CMath:   cMath,
					EDD:     edd,
					Arabe:   arabe,
					Dessin:  dessin,
					Total:   total,
					Moyenne: moyenne,
					Rang:    rang,
				},
				Mention:      appreciation,
				Appreciation: appreciation,
				Date:         time.Now().Format("02/01/2006"),
				Appreciations: map[string]string{
					"RLC":    services.Appreciation(rLC, 40),
					"CLC":    services.Appreciation(cLC, 60),
					"EDD":    services.Appreciation(edd, 40),
					"RMath":  services.Appreciation(rMath, 40),
					"CMath":  services.Appreciation(cMath, 40),
					"Dessin": services.Appreciation(dessin, 10),
					"Arabe":  services.Appreciation(arabe, 10),
				},
			},
			eleveID: int(eleveID),
		})

		// successCount++ déplacé après la génération du PDF
	}

	// Mettre à jour l'effectif et le rang dans chaque bulletin, puis générer les PDFs
	effectifFinal := effectif
	if maxRang > effectifFinal {
		effectifFinal = maxRang
	}
	fmt.Printf("DEBUG - Effectif calculé: %d\n", effectifFinal)
	for _, b := range bulletinsToGenerate {
		b.data.Effectif = effectifFinal
		// Calculer l'appréciation globale de la moyenne
		b.data.AppreciationGlobale = services.AppreciationGlobale(b.data.Notes.Moyenne)
		pdfPath, err := services.GeneratePDF(b.data, b.eleveID)
		if err != nil {
			errorCount++
			continue
		}
		// Enregistrer le bulletin
		_, err = database.DB.Exec("INSERT INTO bulletins (eleve_id, classe_id, pdf_path) VALUES (?, ?, ?)",
			b.eleveID, classeID, pdfPath)
		if err != nil {
			errorCount++
			continue
		}
	}
	// Rediriger vers la page de la classe avec un message de succès
	redirectURL := fmt.Sprintf("/classes/%d?effectif=%d", classeID, effectifFinal)
	queryParams := []string{}
	if successCount > 0 {
		queryParams = append(queryParams, fmt.Sprintf("success=%d", successCount))
	}
	if errorCount > 0 {
		queryParams = append(queryParams, fmt.Sprintf("errors=%d", errorCount))
	}
	if len(queryParams) > 0 {
		redirectURL += "&" + strings.Join(queryParams, "&")
	}
	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
}

func detectCSVSeparator(firstLine string) rune {
	commaCount := strings.Count(firstLine, ",")
	semicolonCount := strings.Count(firstLine, ";")
	if semicolonCount > commaCount {
		return ';'
	}
	return ','
}

func validateHeaders(actual, expected []string) bool {
	if len(actual) < len(expected) {
		return false
	}
	for i, exp := range expected {
		if i >= len(actual) || actual[i] != exp {
			return false
		}
	}
	return true
}

func getMention(moyenne float64) string {
	if moyenne >= 5.0 {
		return "Réussi"
	}
	return "Non Réussi"
}

func parseFloatSafe(s string) float64 {
	val, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		return 0.0
	}
	return val
}

func parseIntSafe(s string) int {
	val, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return 0
	}
	return val
}

func isNumeric(s string) bool {
	_, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	return err == nil
}
