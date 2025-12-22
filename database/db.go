package database

import (
	"database/sql"

	_ "modernc.org/sqlite"
)

var DB *sql.DB

func InitDB() (*sql.DB, error) {
	var err error
	DB, err = sql.Open("sqlite", "./bulletins.db")
	if err != nil {
		return nil, err
	}

	// Créer les tables
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		nom TEXT NOT NULL,
		email TEXT UNIQUE NOT NULL,
		password_hash TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS classes (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		nom TEXT NOT NULL,
		ecole TEXT NOT NULL,
		annee_scolaire TEXT NOT NULL,
		maitre TEXT NOT NULL,
		trimestre TEXT NOT NULL,
		user_id INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS eleves (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		prenom TEXT NOT NULL,
		nom TEXT NOT NULL,
		classe_id INTEGER NOT NULL,
		FOREIGN KEY (classe_id) REFERENCES classes(id)
	);

	CREATE TABLE IF NOT EXISTS notes (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		eleve_id INTEGER NOT NULL,
		r_lc REAL,
		c_lc REAL,
		dm REAL,
		r_math REAL,
		c_math REAL,
		edd REAL,
		arabe REAL,
		dessin REAL,
		total REAL,
		moyenne REAL,
		rang INTEGER,
		FOREIGN KEY (eleve_id) REFERENCES eleves(id)
	);

	CREATE TABLE IF NOT EXISTS bulletins (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		eleve_id INTEGER NOT NULL,
		classe_id INTEGER NOT NULL,
		pdf_path TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (eleve_id) REFERENCES eleves(id),
		FOREIGN KEY (classe_id) REFERENCES classes(id)
	);
	`

	_, err = DB.Exec(schema)
	return DB, err
}

func GetDB() *sql.DB {
	return DB
}
