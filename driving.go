package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type DrivingStudent struct {
	ID             int     `json:"id"`
	CIN            string  `json:"cin"`
	Name           string  `json:"name"`
	Phone          string  `json:"phone"`
	BirthDate      string  `json:"birth_date"`
	Gender         string  `json:"gender"`
	Address        string  `json:"address"`
	Permit         string  `json:"permit"`
	EnrollmentDate string  `json:"enrollment_date"`
	Package        float64 `json:"package"`
	Status         string  `json:"status"`
	DocumentStatus string  `json:"document_status"`
	MedicalExpiry  string  `json:"medical_expiry"`
	TheoryTarget   float64 `json:"theory_target"`
	DrivingTarget  float64 `json:"driving_target"`
	Notes          string  `json:"notes"`
}
type DrivingInstructor struct {
	ID            int     `json:"id"`
	Name          string  `json:"name"`
	Phone         string  `json:"phone"`
	Permit        string  `json:"permit"`
	HireDate      string  `json:"hire_date"`
	Status        string  `json:"status"`
	HourlyRate    float64 `json:"hourly_rate"`
	MonthlySalary float64 `json:"monthly_salary"`
}
type DrivingVehicle struct {
	ID                 int     `json:"id"`
	Brand              string  `json:"brand"`
	Model              string  `json:"model"`
	Plate              string  `json:"plate"`
	Category           string  `json:"category"`
	Year               int     `json:"year"`
	Color              string  `json:"color"`
	Fuel               string  `json:"fuel"`
	Transmission       string  `json:"transmission"`
	VIN                string  `json:"vin"`
	RegistrationDate   string  `json:"registration_date"`
	RegistrationExpiry string  `json:"registration_expiry"`
	TaxExpiry          string  `json:"tax_expiry"`
	InsuranceCompany   string  `json:"insurance_company"`
	InsurancePolicy    string  `json:"insurance_policy"`
	InsuranceExpiry    string  `json:"insurance_expiry"`
	VisitExpiry        string  `json:"visit_expiry"`
	Mileage            int     `json:"mileage"`
	MaintenanceDue     int     `json:"maintenance_due"`
	OilChangeDue       int     `json:"oil_change_due"`
	PurchaseDate       string  `json:"purchase_date"`
	PurchasePrice      float64 `json:"purchase_price"`
	Status             string  `json:"status"`
	Notes              string  `json:"notes"`
}
type DrivingLesson struct {
	ID           int    `json:"id"`
	StudentID    int    `json:"student_id"`
	InstructorID int    `json:"instructor_id"`
	VehicleID    int    `json:"vehicle_id"`
	Start        string `json:"start"`
	Duration     int    `json:"duration"`
	Kind         string `json:"kind"`
	Status       string `json:"status"`
	Notes        string `json:"notes"`
	CreatedBy    string `json:"created_by"`
}
type DrivingExam struct {
	FollowUp  string `json:"follow_up,omitempty"`
	RequestID string `json:"request_id,omitempty"`
	ID        int    `json:"id"`
	StudentID int    `json:"student_id"`
	Type      string `json:"type"`
	Date      string `json:"date"`
	Result    string `json:"result"`
	Center    string `json:"center"`
	Notes     string `json:"notes"`
}
type DrivingPayment struct {
	InvoiceID  int     `json:"invoice_id,omitempty"`
	ID         int     `json:"id"`
	StudentID  int     `json:"student_id"`
	Time       string  `json:"time"`
	Amount     float64 `json:"amount"`
	Method     string  `json:"method"`
	Reference  string  `json:"reference"`
	ReceivedBy string  `json:"received_by"`
}
type DrivingExpense struct {
	ID       int     `json:"id"`
	Time     string  `json:"time"`
	Category string  `json:"category"`
	Label    string  `json:"label"`
	Amount   float64 `json:"amount"`
	User     string  `json:"user"`
}

func (s *Store) ensureDrivingDefaults() {
	if s.data.Students == nil {
		s.data.Students = []DrivingStudent{}
	}
	if s.data.Instructors == nil {
		s.data.Instructors = []DrivingInstructor{}
	}
	if len(s.data.Instructors) == 0 {
		s.data.Instructors = append(s.data.Instructors, DrivingInstructor{
			ID:       1,
			Name:     "Walid Nafougui",
			Permit:   "B",
			HireDate: time.Now().Format("2006-01-02"),
			Status:   "Actif",
		})
	}
	if s.data.Vehicles == nil {
		s.data.Vehicles = []DrivingVehicle{}
	}
	if s.data.Lessons == nil {
		s.data.Lessons = []DrivingLesson{}
	}
	if s.data.Exams == nil {
		s.data.Exams = []DrivingExam{}
	}
	if s.data.DrivingPayments == nil {
		s.data.DrivingPayments = []DrivingPayment{}
	}
	if s.data.Expenses == nil {
		s.data.Expenses = []DrivingExpense{}
	}
	s.ensureDrivingExtendedDefaults()
}

func nextDrivingID[T any](rows []T, id func(T) int) int {
	n := 0
	for _, row := range rows {
		if id(row) > n {
			n = id(row)
		}
	}
	return n + 1
}

func keepDriving[T any](rows []T, keep func(T) bool) []T {
	out := make([]T, 0, len(rows))
	for _, row := range rows {
		if keep(row) {
			out = append(out, row)
		}
	}
	return out
}

func parseLessonTime(value string) (time.Time, error) {
	for _, layout := range []string{time.RFC3339, "2006-01-02T15:04", "2006-01-02T15:04:05"} {
		if valueTime, err := time.ParseInLocation(layout, value, time.Local); err == nil {
			return valueTime, nil
		}
	}
	return time.Time{}, fmt.Errorf("date ou heure invalide")
}

func rangesOverlap(firstStart time.Time, firstDuration int, secondStart time.Time, secondDuration int) bool {
	return firstStart.Before(secondStart.Add(time.Duration(secondDuration)*time.Minute)) && secondStart.Before(firstStart.Add(time.Duration(firstDuration)*time.Minute))
}

func registerDrivingRoutes(mux *http.ServeMux, store *Store, auth *AuthManager) {
	registerCoachRoutes(mux, store, auth)
	registerDrivingExtendedRoutes(mux, store, auth)
	mux.HandleFunc("/api/driving/student", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "DELETE" {
			id, _ := strconv.Atoi(r.URL.Query().Get("id"))
			if id <= 0 {
				errJSON(w, 400, "Candidat invalide")
				return
			}
			store.mu.Lock()
			defer store.mu.Unlock()
			before := store.drivingSnapshotLocked()
			if r.URL.Query().Get("permanent") == "1" {
				name := ""
				students := store.data.Students[:0]
				for _, x := range store.data.Students {
					if x.ID == id {
						name = x.Name
						continue
					}
					students = append(students, x)
				}
				if name == "" {
					errJSON(w, 404, "Candidat introuvable")
					return
				}
				documentFiles := []string{}
				documents := store.data.DrivingDocuments[:0]
				for _, x := range store.data.DrivingDocuments {
					if x.StudentID == id {
						if x.StoredName != "" && filepath.Base(x.StoredName) == x.StoredName {
							documentFiles = append(documentFiles, filepath.Join(appDataDir(), "documents", x.StoredName))
						}
						continue
					}
					documents = append(documents, x)
				}
				lessons := store.data.Lessons[:0]
				for _, x := range store.data.Lessons {
					if x.StudentID != id {
						lessons = append(lessons, x)
					}
				}
				exams := store.data.Exams[:0]
				for _, x := range store.data.Exams {
					if x.StudentID != id {
						exams = append(exams, x)
					}
				}
				payments := store.data.DrivingPayments[:0]
				for _, x := range store.data.DrivingPayments {
					if x.StudentID != id {
						payments = append(payments, x)
					}
				}
				sales := store.data.PackageSales[:0]
				for _, x := range store.data.PackageSales {
					if x.StudentID != id {
						sales = append(sales, x)
					}
				}
				invoices := store.data.DrivingInvoices[:0]
				for _, x := range store.data.DrivingInvoices {
					if x.StudentID != id {
						invoices = append(invoices, x)
					}
				}
				queue := store.data.DrivingConfig.QueueOrder[:0]
				for _, studentID := range store.data.DrivingConfig.QueueOrder {
					if studentID != id {
						queue = append(queue, studentID)
					}
				}
				audits := store.data.DrivingAudits[:0]
				for _, x := range store.data.DrivingAudits {
					if !strings.Contains(strings.ToLower(x.Detail), strings.ToLower(name)) {
						audits = append(audits, x)
					}
				}
				store.data.Students = students
				store.data.Lessons = lessons
				store.data.Exams = exams
				store.data.DrivingPayments = payments
				store.data.DrivingDocuments = documents
				store.data.PackageSales = sales
				store.data.DrivingInvoices = invoices
				store.data.DrivingConfig.QueueOrder = queue
				store.data.DrivingAudits = audits
				if !store.commitDrivingLocked(w, before) {
					return
				}
				for _, path := range documentFiles {
					_ = os.Remove(path)
				}
				okJSON(w, true)
				return
			}
			name, found := store.archiveStudentLocked(id)
			if !found {
				errJSON(w, 404, "Candidat introuvable")
				return
			}
			store.auditLocked(auth.currentUsername(r), "Archivage", "Candidat", name+" : historique conservé")
			if !store.commitDrivingLocked(w, before) {
				return
			}
			okJSON(w, true)
			return
		}
		if r.Method != "POST" {
			errJSON(w, 405, "Méthode non autorisée")
			return
		}
		var x DrivingStudent
		if readJSON(r, &x) != nil || strings.TrimSpace(x.Name) == "" || strings.TrimSpace(x.Phone) == "" {
			errJSON(w, 400, "Nom et téléphone obligatoires")
			return
		}
		if x.Permit == "" {
			x.Permit = "B"
		}
		if x.Gender == "" {
			x.Gender = "Homme"
		}
		if x.Gender != "Homme" && x.Gender != "Femme" {
			errJSON(w, 400, "Genre invalide")
			return
		}
		if x.Status == "" {
			x.Status = "Actif"
		}
		if x.EnrollmentDate == "" {
			x.EnrollmentDate = time.Now().Format("2006-01-02")
		}
		if x.DocumentStatus == "" {
			x.DocumentStatus = "Incomplet"
		}
		if x.TheoryTarget < 0 || x.DrivingTarget < 0 || x.Package < 0 {
			errJSON(w, 400, "Les heures et le montant du forfait ne peuvent pas être négatifs")
			return
		}
		store.mu.Lock()
		defer store.mu.Unlock()
		before := store.drivingSnapshotLocked()
		for _, current := range store.data.Students {
			if current.ID != x.ID && x.CIN != "" && strings.EqualFold(strings.TrimSpace(current.CIN), strings.TrimSpace(x.CIN)) {
				errJSON(w, 409, "Ce CIN existe déjà dans un autre dossier")
				return
			}
		}
		theoryUsed, drivingUsed := 0.0, 0.0
		for _, lesson := range store.data.Lessons {
			if lesson.StudentID != x.ID || x.ID == 0 || lesson.Status == "Annulée" || lesson.Status == "Absence" {
				continue
			}
			if lesson.Kind == "Code" {
				theoryUsed += float64(lesson.Duration) / 60
			} else {
				drivingUsed += float64(lesson.Duration) / 60
			}
		}
		if x.TheoryTarget+.0001 < theoryUsed || x.DrivingTarget+.0001 < drivingUsed {
			errJSON(w, 409, "Les objectifs ne peuvent pas être inférieurs aux heures réalisées et réservées. Annulez les réservations concernées avant de réduire le forfait.")
			return
		}
		if x.ID == 0 {
			x.ID = nextDrivingID(store.data.Students, func(v DrivingStudent) int { return v.ID })
			store.data.Students = append(store.data.Students, x)
		} else {
			found := false
			for i := range store.data.Students {
				if store.data.Students[i].ID == x.ID {
					store.data.Students[i] = x
					found = true
				}
			}
			if !found {
				errJSON(w, 404, "Candidat introuvable")
				return
			}
		}
		if x.Status == "Archivé" {
			store.archiveStudentLocked(x.ID)
		}
		store.auditLocked(auth.currentUsername(r), "Enregistrement", "Candidat", x.Name)
		if !store.commitDrivingLocked(w, before) {
			return
		}
		okJSON(w, x)
	})
	mux.HandleFunc("/api/driving/instructor", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "DELETE" {
			id, _ := strconv.Atoi(r.URL.Query().Get("id"))
			store.mu.Lock()
			defer store.mu.Unlock()
			before := store.drivingSnapshotLocked()
			out := store.data.Instructors[:0]
			for _, x := range store.data.Instructors {
				if x.ID != id {
					out = append(out, x)
				}
			}
			store.data.Instructors = out
			if !store.commitDrivingLocked(w, before) {
				return
			}
			okJSON(w, true)
			return
		}
		if r.Method != "POST" {
			errJSON(w, 405, "Méthode non autorisée")
			return
		}
		var x DrivingInstructor
		if readJSON(r, &x) != nil || strings.TrimSpace(x.Name) == "" {
			errJSON(w, 400, "Nom du moniteur obligatoire")
			return
		}
		if x.Status == "" {
			x.Status = "Actif"
		}
		store.mu.Lock()
		defer store.mu.Unlock()
		before := store.drivingSnapshotLocked()
		for _, current := range store.data.Instructors {
			if current.ID != x.ID && x.Phone != "" && strings.TrimSpace(current.Phone) == strings.TrimSpace(x.Phone) {
				errJSON(w, 409, "Ce téléphone est déjà utilisé par un moniteur")
				return
			}
		}
		if x.ID == 0 {
			x.ID = nextDrivingID(store.data.Instructors, func(v DrivingInstructor) int { return v.ID })
			store.data.Instructors = append(store.data.Instructors, x)
		} else {
			for i := range store.data.Instructors {
				if store.data.Instructors[i].ID == x.ID {
					store.data.Instructors[i] = x
				}
			}
		}
		store.auditLocked(auth.currentUsername(r), "Enregistrement", "Moniteur", x.Name)
		if !store.commitDrivingLocked(w, before) {
			return
		}
		okJSON(w, x)
	})
	mux.HandleFunc("/api/driving/vehicle", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "DELETE" {
			id, _ := strconv.Atoi(r.URL.Query().Get("id"))
			store.mu.Lock()
			defer store.mu.Unlock()
			before := store.drivingSnapshotLocked()
			out := store.data.Vehicles[:0]
			for _, x := range store.data.Vehicles {
				if x.ID != id {
					out = append(out, x)
				}
			}
			store.data.Vehicles = out
			if !store.commitDrivingLocked(w, before) {
				return
			}
			okJSON(w, true)
			return
		}
		if r.Method != "POST" {
			errJSON(w, 405, "Méthode non autorisée")
			return
		}
		var x DrivingVehicle
		if readJSON(r, &x) != nil || strings.TrimSpace(x.Plate) == "" {
			errJSON(w, 400, "Matricule obligatoire")
			return
		}
		if x.Status == "" {
			x.Status = "Disponible"
		}
		store.mu.Lock()
		defer store.mu.Unlock()
		before := store.drivingSnapshotLocked()
		for _, current := range store.data.Vehicles {
			if current.ID != x.ID && strings.EqualFold(strings.TrimSpace(current.Plate), strings.TrimSpace(x.Plate)) {
				errJSON(w, 409, "Ce matricule existe déjà")
				return
			}
		}
		if x.ID == 0 {
			x.ID = nextDrivingID(store.data.Vehicles, func(v DrivingVehicle) int { return v.ID })
			store.data.Vehicles = append(store.data.Vehicles, x)
		} else {
			for i := range store.data.Vehicles {
				if store.data.Vehicles[i].ID == x.ID {
					store.data.Vehicles[i] = x
				}
			}
		}
		store.auditLocked(auth.currentUsername(r), "Enregistrement", "Véhicule", x.Plate)
		if !store.commitDrivingLocked(w, before) {
			return
		}
		okJSON(w, x)
	})
	mux.HandleFunc("/api/driving/lesson", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "DELETE" {
			id, _ := strconv.Atoi(r.URL.Query().Get("id"))
			store.mu.Lock()
			defer store.mu.Unlock()
			before := store.drivingSnapshotLocked()
			deleted := false
			out := store.data.Lessons[:0]
			for _, x := range store.data.Lessons {
				if x.ID != id {
					out = append(out, x)
				} else {
					deleted = true
				}
			}
			if !deleted {
				errJSON(w, 404, "Séance introuvable")
				return
			}
			store.data.Lessons = out
			store.auditLocked(auth.currentUsername(r), "Suppression", "Séance", strconv.Itoa(id))
			if !store.commitDrivingLocked(w, before) {
				return
			}
			okJSON(w, true)
			return
		}
		if r.Method != "POST" {
			errJSON(w, 405, "Méthode non autorisée")
			return
		}
		var x DrivingLesson
		if readJSON(r, &x) != nil || x.StudentID == 0 || x.Start == "" {
			errJSON(w, 400, "Candidat et date obligatoires")
			return
		}
		if x.Duration <= 0 {
			x.Duration = 60
		}
		if x.Duration < 15 || x.Duration > 480 {
			errJSON(w, 400, "La durée doit être comprise entre 15 et 480 minutes")
			return
		}
		start, err := parseLessonTime(x.Start)
		if err != nil {
			errJSON(w, 400, err.Error())
			return
		}
		defaults := store.snapshot()
		if x.InstructorID == 0 {
			for _, instructor := range defaults.Instructors {
				if instructor.Status == "Actif" {
					x.InstructorID = instructor.ID
					break
				}
			}
		}
		if x.VehicleID == 0 && (x.Kind == "Conduite" || x.Kind == "Créneau") {
			for _, vehicle := range defaults.Vehicles {
				if vehicle.Status == "Disponible" {
					x.VehicleID = vehicle.ID
					break
				}
			}
		}
		if x.InstructorID == 0 {
			errJSON(w, 400, "Configurez d'abord le moniteur principal")
			return
		}
		if x.Status == "" {
			x.Status = "Planifiée"
		}
		x.CreatedBy = auth.currentUsername(r)
		store.mu.Lock()
		defer store.mu.Unlock()
		before := store.drivingSnapshotLocked()
		studentFound, instructorFound, vehicleFound := false, false, x.VehicleID == 0
		for _, current := range store.data.Students {
			if current.ID == x.StudentID && current.Status == "Actif" {
				studentFound = true
			}
		}
		for _, current := range store.data.Instructors {
			if current.ID == x.InstructorID && current.Status == "Actif" {
				instructorFound = true
			}
		}
		for _, current := range store.data.Vehicles {
			if current.ID == x.VehicleID && current.Status == "Disponible" {
				vehicleFound = true
			}
		}
		if !studentFound || !instructorFound || !vehicleFound {
			errJSON(w, 409, "Candidat, moniteur ou véhicule indisponible")
			return
		}
		if err := store.validateLessonCapacityLocked(x); err != nil {
			errJSON(w, 409, err.Error())
			return
		}
		lessonDay, lessonClock := start.Format("2006-01-02"), start.Format("15:04")
		finish := start.Add(time.Duration(x.Duration) * time.Minute)
		if x.Status != "Annulée" && ((store.data.DrivingConfig.OpenTime != "" && lessonClock < store.data.DrivingConfig.OpenTime) || (store.data.DrivingConfig.CloseTime != "" && finish.Format("15:04") > store.data.DrivingConfig.CloseTime) || finish.Format("2006-01-02") != lessonDay) {
			errJSON(w, 409, "Cette heure est en dehors des horaires de l’auto-école")
			return
		}
		for _, absence := range store.data.InstructorAbsences {
			if absence.InstructorID == x.InstructorID && lessonDay >= absence.Start && lessonDay <= absence.End {
				errJSON(w, 409, "Ce moniteur est absent à cette date")
				return
			}
		}
		for _, current := range store.data.Lessons {
			if current.ID == x.ID || current.Status == "Annulée" || x.Status == "Annulée" {
				continue
			}
			currentStart, parseErr := parseLessonTime(current.Start)
			if parseErr != nil || !rangesOverlap(start, x.Duration, currentStart, current.Duration) {
				continue
			}
			if current.StudentID == x.StudentID {
				errJSON(w, 409, "Ce candidat a déjà une séance à cette heure")
				return
			}
			// This edition manages a single instructor, including legacy records.
			errJSON(w, 409, "Vous avez déjà une séance à cette heure. Choisissez un créneau libre.")
			return
		}
		if x.ID == 0 {
			x.ID = nextDrivingID(store.data.Lessons, func(v DrivingLesson) int { return v.ID })
			store.data.Lessons = append(store.data.Lessons, x)
		} else {
			for i := range store.data.Lessons {
				if store.data.Lessons[i].ID == x.ID {
					store.data.Lessons[i] = x
				}
			}
		}
		store.auditLocked(auth.currentUsername(r), "Enregistrement", "Séance", x.Kind)
		if !store.commitDrivingLocked(w, before) {
			return
		}
		okJSON(w, x)
	})
	mux.HandleFunc("/api/driving/exam", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			errJSON(w, 405, "Méthode non autorisée")
			return
		}
		var x DrivingExam
		if readJSON(r, &x) != nil || x.StudentID == 0 || x.Date == "" {
			errJSON(w, 400, "Candidat et date obligatoires")
			return
		}
		if x.Result == "" {
			x.Result = "Prévu"
		}
		store.mu.Lock()
		defer store.mu.Unlock()
		before := store.drivingSnapshotLocked()
		studentFound := false
		for _, current := range store.data.Students {
			if current.ID == x.StudentID {
				studentFound = true
			}
		}
		if !studentFound {
			errJSON(w, 404, "Candidat introuvable")
			return
		}

		if x.ID == 0 {
			x.ID = nextDrivingID(store.data.Exams, func(v DrivingExam) int { return v.ID })
			store.data.Exams = append(store.data.Exams, x)
		} else {
			for i := range store.data.Exams {
				if store.data.Exams[i].ID == x.ID {
					previous := store.data.Exams[i]
					if previous.FollowUp != "" && (previous.StudentID != x.StudentID || previous.Result != x.Result || previous.Type != x.Type) {
						errJSON(w, 409, "Cet examen a déjà un suivi enregistré. Modifiez le nouveau rendez-vous ou le forfait depuis le dossier.")
						return
					}
					x.FollowUp, x.RequestID = previous.FollowUp, previous.RequestID
					store.data.Exams[i] = x
				}
			}
		}
		store.auditLocked(auth.currentUsername(r), "Enregistrement", "Examen", x.Type+" · "+x.Result)
		if !store.commitDrivingLocked(w, before) {
			return
		}
		okJSON(w, x)
	})
	mux.HandleFunc("/api/driving/payment", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			errJSON(w, 405, "Méthode non autorisée")
			return
		}
		var x DrivingPayment
		if readJSON(r, &x) != nil || x.StudentID == 0 || x.Amount <= 0 {
			errJSON(w, 400, "Paiement invalide")
			return
		}
		if x.Time == "" {
			x.Time = time.Now().Format(time.RFC3339)
		}
		if x.Method == "" {
			x.Method = "Espèces"
		}
		x.ReceivedBy = auth.currentUsername(r)
		store.mu.Lock()
		defer store.mu.Unlock()
		before := store.drivingSnapshotLocked()
		studentFound := false
		for _, current := range store.data.Students {
			if current.ID == x.StudentID {
				studentFound = true
			}
		}
		if !studentFound {
			errJSON(w, 404, "Candidat introuvable")
			return
		}
		if x.InvoiceID != 0 {
			found := false
			for _, invoice := range store.data.DrivingInvoices {
				if invoice.ID != x.InvoiceID {
					continue
				}
				if invoice.StudentID != x.StudentID || invoice.Status == "Annulée" {
					errJSON(w, 409, "Cette facture ne peut pas recevoir ce paiement")
					return
				}
				total := x.Amount
				for _, payment := range store.data.DrivingPayments {
					if payment.ID != x.ID && payment.InvoiceID == x.InvoiceID {
						total += payment.Amount
					}
				}
				if total > invoice.Amount+.0005 {
					errJSON(w, 409, "Le paiement dépasse le reste de cette facture. Enregistrez l’avance séparément, sans facture.")
					return
				}
				found = true
			}
			if !found {
				errJSON(w, 404, "Facture introuvable")
				return
			}
		}
		if x.ID != 0 {
			found := false
			for _, payment := range store.data.DrivingPayments {
				if payment.ID == x.ID {
					found = true
				}
			}
			if !found {
				errJSON(w, 404, "Paiement introuvable")
				return
			}
		}
		if x.ID == 0 {
			x.ID = nextDrivingID(store.data.DrivingPayments, func(v DrivingPayment) int { return v.ID })
			store.data.DrivingPayments = append(store.data.DrivingPayments, x)
		} else {
			for i := range store.data.DrivingPayments {
				if store.data.DrivingPayments[i].ID == x.ID {
					store.data.DrivingPayments[i] = x
				}
			}
		}
		store.auditLocked(auth.currentUsername(r), "Enregistrement", "Paiement", fmt.Sprintf("%.3f DT", x.Amount))
		if !store.commitDrivingLocked(w, before) {
			return
		}
		okJSON(w, x)
	})
	mux.HandleFunc("/api/driving/expense", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			errJSON(w, 405, "Méthode non autorisée")
			return
		}
		var x DrivingExpense
		if readJSON(r, &x) != nil || x.Amount <= 0 || strings.TrimSpace(x.Label) == "" {
			errJSON(w, 400, "Dépense invalide")
			return
		}
		if x.Time == "" {
			x.Time = time.Now().Format(time.RFC3339)
		}
		x.User = auth.currentUsername(r)
		store.mu.Lock()
		defer store.mu.Unlock()
		before := store.drivingSnapshotLocked()
		x.ID = nextDrivingID(store.data.Expenses, func(v DrivingExpense) int { return v.ID })
		store.data.Expenses = append(store.data.Expenses, x)
		store.auditLocked(auth.currentUsername(r), "Sortie", "Dépense", x.Label)
		if !store.commitDrivingLocked(w, before) {
			return
		}
		okJSON(w, x)
	})
	mux.HandleFunc("/api/driving/payment-receipt.pdf", func(w http.ResponseWriter, r *http.Request) {
		id, _ := strconv.Atoi(r.URL.Query().Get("id"))
		data := store.snapshot()
		var payment *DrivingPayment
		for i := range data.DrivingPayments {
			if data.DrivingPayments[i].ID == id {
				payment = &data.DrivingPayments[i]
			}
		}
		if payment == nil {
			http.NotFound(w, r)
			return
		}
		student := "Candidat"
		for _, x := range data.Students {
			if x.ID == payment.StudentID {
				student = x.Name
			}
		}
		serveDrivingPDF(w, fmt.Sprintf("Recu-%06d.pdf", payment.ID), data.Settings.Business, "BON DE RÉCEPTION / REÇU", []string{fmt.Sprintf("Reçu N° %06d", payment.ID), "Candidat : " + student, "Date : " + formatDrivingDate(payment.Time), "Mode de paiement : " + payment.Method, "Référence : " + payment.Reference, fmt.Sprintf("MONTANT REÇU : %.3f DT", payment.Amount), "Encaissement : " + payment.ReceivedBy, data.DrivingConfig.ReceiptNote})
	})
}

func formatDrivingDate(value string) string {
	if t, err := time.Parse(time.RFC3339, value); err == nil {
		return t.Format("02/01/2006 15:04")
	}
	return value
}
