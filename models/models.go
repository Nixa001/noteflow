package models

import "time"

type User struct {
	ID           int
	Nom          string
	Email        string
	PasswordHash string
	CreatedAt    time.Time
}

type Classe struct {
	ID            int
	Nom           string
	Ecole         string
	AnneeScolaire string
	Effectif      int
	UserID        int
	CreatedAt     time.Time
}

type Eleve struct {
	ID       int
	Prenom   string
	Nom      string
	ClasseID int
}

type Notes struct {
	ID      int
	EleveID int
	RLC     float64
	CLC     float64
	DM      float64
	RMath   float64
	CMath   float64
	EDD     float64
	Arabe   float64
	Dessin  float64
	Total   float64
	Moyenne float64
	Rang    int
}

type Bulletin struct {
	ID        int
	EleveID   int
	ClasseID  int
	PDFPath   string
	CreatedAt time.Time
}

type BulletinData struct {
	Ecole         string
	Classe        string
	AnneeScolaire string
	Effectif      int
	Maitre        string
	Prenom        string
	Nom           string
	Notes         Notes
	Mention       string
	Date          string
	Appreciation  string
}
