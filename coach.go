package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

func validCoachHours(open, close string) bool {
	a, e := time.Parse("15:04", open)
	b, f := time.Parse("15:04", close)
	return e == nil && f == nil && a.Before(b)
}

// Queue and exam follow-up are persisted, not stored in a browser tied to a port.
func registerCoachRoutes(mux *http.ServeMux, store *Store, auth *AuthManager) {
	mux.HandleFunc("/api/driving/queue", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			errJSON(w, 405, "Méthode non autorisée")
			return
		}
		var ids []int
		if readJSON(r, &ids) != nil {
			errJSON(w, 400, "Ordre invalide")
			return
		}
		store.mu.Lock()
		defer store.mu.Unlock()
		seen := map[int]bool{}
		for _, id := range ids {
			found := false
			for _, s := range store.data.Students {
				if s.ID == id {
					found = true
					break
				}
			}
			if !found || seen[id] {
				errJSON(w, 400, "Élève inconnu ou en double")
				return
			}
			seen[id] = true
		}
		previous := store.data.DrivingConfig.QueueOrder
		store.data.DrivingConfig.QueueOrder = ids
		if err := store.saveLocked(); err != nil {
			store.data.DrivingConfig.QueueOrder = previous
			errJSON(w, 500, "Ordre non enregistré")
			return
		}
		okJSON(w, ids)
	})
	mux.HandleFunc("/api/driving/exam-followup", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			errJSON(w, 405, "Méthode non autorisée")
			return
		}
		var req struct {
			Exam     DrivingExam `json:"exam"`
			Action   string      `json:"action"`
			Hours    float64     `json:"hours"`
			Price    float64     `json:"price"`
			NextDate string      `json:"next_date"`
		}
		if readJSON(r, &req) != nil {
			errJSON(w, 400, "Demande invalide")
			return
		}
		x := req.Exam
		if _, err := time.Parse("2006-01-02", x.Date); err != nil || x.StudentID == 0 || x.Result != "Échec" {
			errJSON(w, 400, "Examen invalide")
			return
		}
		if req.Action != "record" && req.Action != "rattrapage" && req.Action != "reexam" {
			errJSON(w, 400, "Choisissez une décision")
			return
		}
		if req.Action == "rattrapage" && (req.Hours <= 0 || req.Hours > 200 || req.Price < 0 || (x.Type != "Conduite" && x.Type != "Créneau")) {
			errJSON(w, 400, "Heures ou montant de rattrapage invalides")
			return
		}
		if req.Action == "reexam" {
			if _, err := time.Parse("2006-01-02", req.NextDate); err != nil || req.NextDate <= x.Date {
				errJSON(w, 400, "Le nouvel examen doit être après la date de l’échec")
				return
			}
		}
		store.mu.Lock()
		defer store.mu.Unlock()
		si, ei := -1, -1
		for i, s := range store.data.Students {
			if s.ID == x.StudentID {
				si = i
			}
		}
		for i, e := range store.data.Exams {
			if x.RequestID != "" && e.RequestID == x.RequestID {
				okJSON(w, e)
				return
			}
			if e.ID == x.ID {
				ei = i
			}
		}
		if si < 0 || (x.ID != 0 && ei < 0) {
			errJSON(w, 404, "Dossier ou examen introuvable")
			return
		}
		if ei >= 0 && store.data.Exams[ei].FollowUp != "" {
			errJSON(w, 409, "Une suite a déjà été enregistrée pour cet échec. Consultez le dossier avant une autre modification.")
			return
		}
		// Deep rollback snapshot: failed disk writes must not leave half a follow-up in memory.
		backup, _ := json.Marshal(store.data)
		x.FollowUp = ""
		if req.Action != "record" {
			x.FollowUp = req.Action
		}
		if x.ID == 0 {
			x.ID = nextDrivingID(store.data.Exams, func(e DrivingExam) int { return e.ID })
			store.data.Exams = append(store.data.Exams, x)
		} else {
			store.data.Exams[ei] = x
		}
		if req.Action == "rattrapage" {
			done := 0.0
			for _, l := range store.data.Lessons {
				if l.StudentID == x.StudentID && l.Status == "Terminée" && (l.Kind == "Conduite" || l.Kind == "Créneau") {
					done += float64(l.Duration) / 60
				}
			}
			s := &store.data.Students[si]
			if s.DrivingTarget < done {
				s.DrivingTarget = done
			}
			s.DrivingTarget += req.Hours
			s.Package += req.Price
			s.Status = "Actif"
			s.Notes += fmt.Sprintf("\nRattrapage après examen #%d : +%g h, +%.3f DT", x.ID, req.Hours, req.Price)
		} else if req.Action == "reexam" {
			next := DrivingExam{ID: nextDrivingID(store.data.Exams, func(e DrivingExam) int { return e.ID }), StudentID: x.StudentID, Type: x.Type, Date: req.NextDate, Result: "Prévu", Center: x.Center, Notes: fmt.Sprintf("Nouvelle tentative après examen #%d", x.ID)}
			store.data.Exams = append(store.data.Exams, next)
		}
		store.auditLocked(auth.currentUsername(r), "Résultat et suivi", "Examen", fmt.Sprintf("#%d · %s", x.ID, req.Action))
		if err := store.saveLocked(); err != nil {
			_ = json.Unmarshal(backup, &store.data)
			errJSON(w, 500, "Enregistrement impossible. Aucun changement appliqué.")
			return
		}
		okJSON(w, x)
	})
}
