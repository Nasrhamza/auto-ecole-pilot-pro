package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Called with the store locked, before any mutation.
func (s *Store) drivingSnapshotLocked() []byte {
	b, _ := json.Marshal(s.data)
	return b
}

func (s *Store) commitDrivingLocked(w http.ResponseWriter, before []byte) bool {
	refreshInvoiceStatuses(&s.data)
	if err := s.saveLocked(); err != nil {
		_ = json.Unmarshal(before, &s.data)
		errJSON(w, 500, "Enregistrement impossible sur le disque. Aucun changement conservé. Réessayez.")
		return false
	}
	return true
}

func refreshInvoiceStatuses(data *StoreData) {
	for i := range data.DrivingInvoices {
		invoice := &data.DrivingInvoices[i]
		if invoice.Status == "Annulée" {
			continue
		}
		paid := 0.0
		for _, payment := range data.DrivingPayments {
			if payment.InvoiceID == invoice.ID && payment.StudentID == invoice.StudentID {
				paid += payment.Amount
			}
		}
		invoice.Status = "À payer"
		if paid > 0.0005 {
			invoice.Status = "Partiellement payée"
		}
		if paid+0.0005 >= invoice.Amount {
			invoice.Status = "Payée"
		}
	}
}

func (s *Store) validateLessonCapacityLocked(x DrivingLesson) error {
	if x.Status == "Annulée" || x.Status == "Absence" {
		return nil
	}
	var pupil *DrivingStudent
	for i := range s.data.Students {
		if s.data.Students[i].ID == x.StudentID {
			pupil = &s.data.Students[i]
			break
		}
	}
	if pupil == nil {
		return fmt.Errorf("Candidat introuvable")
	}
	used := float64(x.Duration) / 60
	drive := x.Kind == "Conduite" || x.Kind == "Créneau"
	for _, lesson := range s.data.Lessons {
		if lesson.ID == x.ID || lesson.StudentID != x.StudentID || lesson.Status == "Annulée" || lesson.Status == "Absence" {
			continue
		}
		if (lesson.Kind == "Conduite" || lesson.Kind == "Créneau") == drive {
			used += float64(lesson.Duration) / 60
		}
	}
	target := pupil.TheoryTarget
	if drive {
		target = pupil.DrivingTarget
	}
	// A zero target means that no hourly package has been configured yet. It must
	// not make the candidate impossible to schedule; positive targets remain a
	// strict cap so completed and reserved hours stay reliable.
	if target > 0 && used > target+0.0001 {
		return fmt.Errorf("Le forfait prévoit %.1f h, mais cette séance porterait le total réalisé et réservé à %.1f h. Ajoutez des heures au forfait avant de planifier.", target, used)
	}
	// Expiry is checked against the lesson date, not just the current day.
	if drive && x.Status != "Terminée" {
		if x.VehicleID == 0 {
			return fmt.Errorf("Choisissez un véhicule pour la conduite")
		}
		date := x.Start[:10]
		for _, v := range s.data.Vehicles {
			if v.ID != x.VehicleID {
				continue
			}
			if (v.InsuranceExpiry != "" && v.InsuranceExpiry < date) || (v.VisitExpiry != "" && v.VisitExpiry < date) {
				return fmt.Errorf("Assurance ou visite technique expirée à la date de la séance. Mettez la fiche véhicule à jour.")
			}
		}
	}
	return nil
}

func (s *Store) archiveStudentLocked(id int) (string, bool) {
	for i := range s.data.Students {
		if s.data.Students[i].ID != id {
			continue
		}
		s.data.Students[i].Status = "Archivé"
		for j := range s.data.Lessons {
			lesson := &s.data.Lessons[j]
			start, err := parseLessonTime(lesson.Start)
			if lesson.StudentID == id && err == nil && start.After(time.Now()) && (lesson.Status == "Planifiée" || lesson.Status == "Confirmée") {
				lesson.Status = "Annulée"
			}
		}
		s.data.DrivingConfig.QueueOrder = keepDriving(s.data.DrivingConfig.QueueOrder, func(v int) bool { return v != id })
		return s.data.Students[i].Name, true
	}
	return "", false
}
