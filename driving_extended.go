package main

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type DrivingDocument struct {
	ID         int    `json:"id"`
	StudentID  int    `json:"student_id"`
	Category   string `json:"category"`
	Name       string `json:"name"`
	StoredName string `json:"stored_name"`
	Expiry     string `json:"expiry"`
	Status     string `json:"status"`
	UploadedAt string `json:"uploaded_at"`
	UploadedBy string `json:"uploaded_by"`
}

type DrivingPackage struct {
	ID           int     `json:"id"`
	Name         string  `json:"name"`
	TheoryHours  float64 `json:"theory_hours"`
	DrivingHours float64 `json:"driving_hours"`
	Price        float64 `json:"price"`
	Active       bool    `json:"active"`
}

type DrivingPackageSale struct {
	ID           int     `json:"id"`
	StudentID    int     `json:"student_id"`
	PackageID    int     `json:"package_id"`
	PackageName  string  `json:"package_name"`
	TheoryHours  float64 `json:"theory_hours"`
	DrivingHours float64 `json:"driving_hours"`
	Price        float64 `json:"price"`
	Date         string  `json:"date"`
	CreatedBy    string  `json:"created_by"`
}

type DrivingVehicleEvent struct {
	ID        int     `json:"id"`
	VehicleID int     `json:"vehicle_id"`
	Type      string  `json:"type"`
	Date      string  `json:"date"`
	Mileage   int     `json:"mileage"`
	Cost      float64 `json:"cost"`
	Provider  string  `json:"provider"`
	Notes     string  `json:"notes"`
	NextDue   int     `json:"next_due"`
}

type DrivingInstructorAbsence struct {
	ID           int    `json:"id"`
	InstructorID int    `json:"instructor_id"`
	Start        string `json:"start"`
	End          string `json:"end"`
	Reason       string `json:"reason"`
}

type DrivingAudit struct {
	ID     int    `json:"id"`
	Time   string `json:"time"`
	User   string `json:"user"`
	Action string `json:"action"`
	Entity string `json:"entity"`
	Detail string `json:"detail"`
}

type DrivingInvoice struct {
	ID           int     `json:"id"`
	StudentID    int     `json:"student_id"`
	InstructorID int     `json:"instructor_id"`
	Date         string  `json:"date"`
	Label        string  `json:"label"`
	Amount       float64 `json:"amount"`
	Status       string  `json:"status"`
	CreatedBy    string  `json:"created_by"`
}

type DrivingConfig struct {
	QueueOrder           []int  `json:"queue_order"`
	VisitAlertDays       int    `json:"visit_alert_days"`
	InsuranceAlertDays   int    `json:"insurance_alert_days"`
	ExamAlertDays        int    `json:"exam_alert_days"`
	MedicalAlertDays     int    `json:"medical_alert_days"`
	DefaultLessonMinutes int    `json:"default_lesson_minutes"`
	OpenTime             string `json:"open_time"`
	CloseTime            string `json:"close_time"`
	InvoiceNote          string `json:"invoice_note"`
	ReceiptNote          string `json:"receipt_note"`
	AppointmentNote      string `json:"appointment_note"`
	CertificateIntro     string `json:"certificate_intro"`
	CertificateClosing   string `json:"certificate_closing"`
}

func (s *Store) ensureDrivingExtendedDefaults() {
	if s.data.DrivingDocuments == nil {
		s.data.DrivingDocuments = []DrivingDocument{}
	}
	if s.data.DrivingPackages == nil {
		s.data.DrivingPackages = []DrivingPackage{}
	}
	if s.data.PackageSales == nil {
		s.data.PackageSales = []DrivingPackageSale{}
	}
	if s.data.VehicleEvents == nil {
		s.data.VehicleEvents = []DrivingVehicleEvent{}
	}
	if s.data.InstructorAbsences == nil {
		s.data.InstructorAbsences = []DrivingInstructorAbsence{}
	}
	if s.data.DrivingAudits == nil {
		s.data.DrivingAudits = []DrivingAudit{}
	}
	if s.data.DrivingInvoices == nil {
		s.data.DrivingInvoices = []DrivingInvoice{}
	}
	c := &s.data.DrivingConfig
	if c.VisitAlertDays <= 0 {
		c.VisitAlertDays = 30
	}
	if c.InsuranceAlertDays <= 0 {
		c.InsuranceAlertDays = 30
	}
	if c.ExamAlertDays <= 0 {
		c.ExamAlertDays = 7
	}
	if c.MedicalAlertDays <= 0 {
		c.MedicalAlertDays = 30
	}
	if c.DefaultLessonMinutes <= 0 {
		c.DefaultLessonMinutes = 60
	}
	if c.OpenTime == "" {
		c.OpenTime = "08:00"
	}
	if c.CloseTime == "" {
		c.CloseTime = "18:00"
	}
	if c.InvoiceNote == "" {
		c.InvoiceNote = "Merci pour votre confiance."
	}
	if c.ReceiptNote == "" {
		c.ReceiptNote = "Paiement reçu, sous réserve d'encaissement."
	}
	if c.AppointmentNote == "" {
		c.AppointmentNote = "Merci de vous présenter 10 minutes avant le rendez-vous."
	}
	if c.CertificateIntro == "" {
		c.CertificateIntro = "Nous certifions que le candidat :"
	}
	if c.CertificateClosing == "" {
		c.CertificateClosing = "Le présent certificat est délivré à l'intéressé pour servir et valoir ce que de droit."
	}
	if len(s.data.DrivingPackages) == 0 {
		s.data.DrivingPackages = []DrivingPackage{{ID: 1, Name: "Permis B Essentiel", TheoryHours: 20, DrivingHours: 20, Price: 900, Active: true}}
	}
}

func (s *Store) auditLocked(user, action, entity, detail string) {
	id := nextDrivingID(s.data.DrivingAudits, func(v DrivingAudit) int { return v.ID })
	s.data.DrivingAudits = append(s.data.DrivingAudits, DrivingAudit{ID: id, Time: time.Now().Format(time.RFC3339), User: user, Action: action, Entity: entity, Detail: detail})
	if len(s.data.DrivingAudits) > 2000 {
		s.data.DrivingAudits = append([]DrivingAudit(nil), s.data.DrivingAudits[len(s.data.DrivingAudits)-2000:]...)
	}
}

func registerDrivingExtendedRoutes(mux *http.ServeMux, store *Store, auth *AuthManager) {
	mux.HandleFunc("/api/driving/users", func(w http.ResponseWriter, r *http.Request) {
		if auth.currentRole(r) != "Administrateur" {
			errJSON(w, 403, "Réservé à l’administrateur")
			return
		}
		if r.Method == "GET" {
			okJSON(w, auth.publicUsers())
			return
		}
		if r.Method == "DELETE" {
			if err := auth.deleteUser(r.URL.Query().Get("username")); err != nil {
				errJSON(w, 400, err.Error())
				return
			}
			okJSON(w, true)
			return
		}
		if r.Method != "POST" {
			errJSON(w, 405, "Méthode non autorisée")
			return
		}
		var in struct {
			Username string `json:"username"`
			Password string `json:"password"`
			Role     string `json:"role"`
		}
		if readJSON(r, &in) != nil {
			errJSON(w, 400, "Données invalides")
			return
		}
		if err := auth.addUser(in.Username, in.Password, in.Role); err != nil {
			errJSON(w, 400, err.Error())
			return
		}
		okJSON(w, true)
	})
	mux.HandleFunc("/api/driving/package", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			errJSON(w, 405, "Méthode non autorisée")
			return
		}
		var x DrivingPackage
		if readJSON(r, &x) != nil || strings.TrimSpace(x.Name) == "" || x.Price < 0 {
			errJSON(w, 400, "Forfait invalide")
			return
		}
		x.Active = true
		store.mu.Lock()
		defer store.mu.Unlock()
		before := store.drivingSnapshotLocked()
		if x.ID == 0 {
			x.ID = nextDrivingID(store.data.DrivingPackages, func(v DrivingPackage) int { return v.ID })
			x.Active = true
			store.data.DrivingPackages = append(store.data.DrivingPackages, x)
		} else {
			found := false
			for i := range store.data.DrivingPackages {
				if store.data.DrivingPackages[i].ID == x.ID {
					store.data.DrivingPackages[i] = x
					found = true
				}
			}
			if !found {
				errJSON(w, 404, "Forfait introuvable")
				return
			}
		}
		store.auditLocked(auth.currentUsername(r), "Enregistrement", "Forfait", x.Name)
		if !store.commitDrivingLocked(w, before) {
			return
		}
		okJSON(w, x)
	})
	mux.HandleFunc("/api/driving/package-sale", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			errJSON(w, 405, "Méthode non autorisée")
			return
		}
		var in struct {
			StudentID int    `json:"student_id"`
			PackageID int    `json:"package_id"`
			Mode      string `json:"mode"`
		}
		if readJSON(r, &in) != nil {
			errJSON(w, 400, "Données invalides")
			return
		}
		store.mu.Lock()
		defer store.mu.Unlock()
		before := store.drivingSnapshotLocked()
		var p *DrivingPackage
		studentIndex := -1
		for i := range store.data.DrivingPackages {
			if store.data.DrivingPackages[i].ID == in.PackageID {
				p = &store.data.DrivingPackages[i]
			}
		}
		for i := range store.data.Students {
			if store.data.Students[i].ID == in.StudentID {
				studentIndex = i
			}
		}
		if p == nil || studentIndex < 0 {
			errJSON(w, 404, "Candidat ou forfait introuvable")
			return
		}
		if in.Mode != "initial" && in.Mode != "additional" {
			errJSON(w, 400, "Choisissez forfait initial ou heures supplémentaires")
			return
		}
		if in.Mode == "initial" {
			for _, sale := range store.data.PackageSales {
				if sale.StudentID == in.StudentID {
					errJSON(w, 409, "Un forfait est déjà affecté. Choisissez des heures supplémentaires.")
					return
				}
			}
			doneTheory, doneDrive := 0.0, 0.0
			for _, lesson := range store.data.Lessons {
				if lesson.StudentID != in.StudentID || lesson.Status == "Annulée" || lesson.Status == "Absence" {
					continue
				}
				if lesson.Kind == "Code" {
					doneTheory += float64(lesson.Duration) / 60
				} else {
					doneDrive += float64(lesson.Duration) / 60
				}
			}
			if p.TheoryHours < doneTheory || p.DrivingHours < doneDrive {
				errJSON(w, 409, "Le forfait doit couvrir les heures déjà réalisées et réservées.")
				return
			}
			store.data.Students[studentIndex].Package = 0
			store.data.Students[studentIndex].TheoryTarget = 0
			store.data.Students[studentIndex].DrivingTarget = 0
		}
		x := DrivingPackageSale{ID: nextDrivingID(store.data.PackageSales, func(v DrivingPackageSale) int { return v.ID }), StudentID: in.StudentID, PackageID: p.ID, PackageName: p.Name, TheoryHours: p.TheoryHours, DrivingHours: p.DrivingHours, Price: p.Price, Date: time.Now().Format(time.RFC3339), CreatedBy: auth.currentUsername(r)}
		store.data.PackageSales = append(store.data.PackageSales, x)
		store.data.Students[studentIndex].Package += p.Price
		store.data.Students[studentIndex].TheoryTarget += p.TheoryHours
		store.data.Students[studentIndex].DrivingTarget += p.DrivingHours
		store.auditLocked(auth.currentUsername(r), "Affectation", "Forfait", p.Name)
		if !store.commitDrivingLocked(w, before) {
			return
		}
		okJSON(w, x)
	})
	mux.HandleFunc("/api/driving/document", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "DELETE" {
			id, _ := strconv.Atoi(r.URL.Query().Get("id"))
			store.mu.Lock()
			defer store.mu.Unlock()
			before := store.drivingSnapshotLocked()
			out := store.data.DrivingDocuments[:0]
			studentID := 0
			removedPath := ""
			for _, x := range store.data.DrivingDocuments {
				if x.ID == id {
					studentID = x.StudentID
					removedPath = filepath.Join(appDataDir(), "documents", x.StoredName)
					continue
				}
				out = append(out, x)
			}
			store.data.DrivingDocuments = out
			store.refreshStudentDocumentStatusLocked(studentID)
			store.auditLocked(auth.currentUsername(r), "Suppression", "Document", strconv.Itoa(id))
			if !store.commitDrivingLocked(w, before) {
				return
			}
			if removedPath != "" {
				_ = os.Remove(removedPath)
			}
			okJSON(w, true)
			return
		}
		if r.Method != "POST" {
			errJSON(w, 405, "Méthode non autorisée")
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 6<<20)
		if err := r.ParseMultipartForm(6 << 20); err != nil {
			errJSON(w, 400, "Fichier invalide ou supérieur à 5 Mo")
			return
		}
		studentID, _ := strconv.Atoi(r.FormValue("student_id"))
		replaceID, _ := strconv.Atoi(r.FormValue("replace_id"))
		category := strings.TrimSpace(r.FormValue("category"))
		expiry := r.FormValue("expiry")
		file, header, err := r.FormFile("file")
		if err != nil || studentID == 0 || category == "" {
			errJSON(w, 400, "Candidat, catégorie et fichier obligatoires")
			return
		}
		defer file.Close()
		ext := strings.ToLower(filepath.Ext(filepath.Base(header.Filename)))
		if ext != ".pdf" && ext != ".png" && ext != ".jpg" && ext != ".jpeg" {
			errJSON(w, 400, "Formats acceptés : PDF, PNG ou JPG")
			return
		}
		dir := filepath.Join(appDataDir(), "documents")
		if err = os.MkdirAll(dir, 0700); err != nil {
			errJSON(w, 500, "Stockage indisponible")
			return
		}
		store.mu.Lock()
		id := nextDrivingID(store.data.DrivingDocuments, func(v DrivingDocument) int { return v.ID })
		stored := fmt.Sprintf("%d_%d%s", studentID, id, ext)
		store.mu.Unlock()
		target := filepath.Join(dir, stored)
		dst, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err != nil {
			errJSON(w, 500, "Enregistrement impossible")
			return
		}
		_, copyErr := io.Copy(dst, file)
		closeErr := dst.Close()
		if copyErr != nil || closeErr != nil {
			_ = os.Remove(target)
			errJSON(w, 500, "Enregistrement incomplet")
			return
		}
		x := DrivingDocument{ID: id, StudentID: studentID, Category: category, Name: filepath.Base(header.Filename), StoredName: stored, Expiry: expiry, Status: "Valide", UploadedAt: time.Now().Format(time.RFC3339), UploadedBy: auth.currentUsername(r)}
		store.mu.Lock()
		before := store.drivingSnapshotLocked()
		replacedPath := ""
		if replaceID > 0 {
			kept := store.data.DrivingDocuments[:0]
			for _, document := range store.data.DrivingDocuments {
				if document.ID == replaceID && document.StudentID == studentID && document.Category == category {
					replacedPath = filepath.Join(appDataDir(), "documents", document.StoredName)
					continue
				}
				kept = append(kept, document)
			}
			store.data.DrivingDocuments = kept
		}
		store.data.DrivingDocuments = append(store.data.DrivingDocuments, x)
		store.refreshStudentDocumentStatusLocked(studentID)
		action := "Ajout"
		if replacedPath != "" {
			action = "Remplacement"
		}
		store.auditLocked(auth.currentUsername(r), action, "Document", x.Name)
		saved := store.commitDrivingLocked(w, before)
		store.mu.Unlock()
		if !saved {
			_ = os.Remove(target)
			return
		}
		if replacedPath != "" {
			_ = os.Remove(replacedPath)
		}
		okJSON(w, x)
	})
	mux.HandleFunc("/api/driving/document/download", func(w http.ResponseWriter, r *http.Request) {
		id, _ := strconv.Atoi(r.URL.Query().Get("id"))
		data := store.snapshot()
		for _, x := range data.DrivingDocuments {
			if x.ID == id {
				disposition := "attachment"
				if r.URL.Query().Get("inline") == "1" {
					disposition = "inline"
				}
				w.Header().Set("Content-Disposition", fmt.Sprintf("%s; filename=%q", disposition, x.Name))
				http.ServeFile(w, r, filepath.Join(appDataDir(), "documents", x.StoredName))
				return
			}
		}
		http.NotFound(w, r)
	})
	mux.HandleFunc("/api/driving/vehicle-event", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			errJSON(w, 405, "Méthode non autorisée")
			return
		}
		var x DrivingVehicleEvent
		if readJSON(r, &x) != nil || x.VehicleID == 0 || x.Date == "" || x.Type == "" {
			errJSON(w, 400, "Intervention invalide")
			return
		}
		store.mu.Lock()
		defer store.mu.Unlock()
		before := store.drivingSnapshotLocked()
		x.ID = nextDrivingID(store.data.VehicleEvents, func(v DrivingVehicleEvent) int { return v.ID })
		store.data.VehicleEvents = append(store.data.VehicleEvents, x)
		vehicleLabel := "Véhicule"
		for i := range store.data.Vehicles {
			if store.data.Vehicles[i].ID == x.VehicleID {
				vehicleLabel = strings.TrimSpace(store.data.Vehicles[i].Brand + " " + store.data.Vehicles[i].Model + " · " + store.data.Vehicles[i].Plate)
				if x.Mileage > store.data.Vehicles[i].Mileage {
					store.data.Vehicles[i].Mileage = x.Mileage
				}
				if x.NextDue > 0 {
					if strings.EqualFold(strings.TrimSpace(x.Type), "Vidange") {
						store.data.Vehicles[i].OilChangeDue = x.NextDue
					} else {
						store.data.Vehicles[i].MaintenanceDue = x.NextDue
					}
				}
			}
		}
		if x.Cost > 0 {
			eventTime := x.Date + "T12:00:00"
			label := strings.TrimSpace(x.Type + " · " + vehicleLabel)
			if strings.TrimSpace(x.Provider) != "" {
				label += " · " + strings.TrimSpace(x.Provider)
			}
			store.data.Expenses = append(store.data.Expenses, DrivingExpense{ID: nextDrivingID(store.data.Expenses, func(v DrivingExpense) int { return v.ID }), Time: eventTime, Category: "Véhicule · " + x.Type, Label: label, Amount: x.Cost, User: auth.currentUsername(r)})
		}
		store.auditLocked(auth.currentUsername(r), "Ajout", "Véhicule", x.Type)
		if !store.commitDrivingLocked(w, before) {
			return
		}
		okJSON(w, x)
	})
	mux.HandleFunc("/api/driving/absence", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			errJSON(w, 405, "Méthode non autorisée")
			return
		}
		var x DrivingInstructorAbsence
		if readJSON(r, &x) != nil || x.InstructorID == 0 || x.Start == "" || x.End == "" || x.End < x.Start {
			errJSON(w, 400, "Absence invalide")
			return
		}
		store.mu.Lock()
		defer store.mu.Unlock()
		before := store.drivingSnapshotLocked()
		for _, lesson := range store.data.Lessons {
			lessonDate := lesson.Start
			if len(lessonDate) > 10 {
				lessonDate = lessonDate[:10]
			}
			if lesson.InstructorID == x.InstructorID && lessonDate >= x.Start && lessonDate <= x.End && lesson.Status != "Annulée" {
				errJSON(w, 409, "Ce moniteur possède déjà des séances pendant cette période")
				return
			}
		}
		x.ID = nextDrivingID(store.data.InstructorAbsences, func(v DrivingInstructorAbsence) int { return v.ID })
		store.data.InstructorAbsences = append(store.data.InstructorAbsences, x)
		store.auditLocked(auth.currentUsername(r), "Ajout", "Absence", x.Reason)
		if !store.commitDrivingLocked(w, before) {
			return
		}
		okJSON(w, x)
	})
	mux.HandleFunc("/api/driving/config", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			errJSON(w, 405, "Méthode non autorisée")
			return
		}
		var x DrivingConfig
		if readJSON(r, &x) != nil || x.DefaultLessonMinutes < 15 || x.DefaultLessonMinutes > 480 || !validCoachHours(x.OpenTime, x.CloseTime) {
			errJSON(w, 400, "Paramètres invalides")
			return
		}
		store.mu.Lock()
		defer store.mu.Unlock()
		before := store.drivingSnapshotLocked()
		// Other settings forms must not discard the instructor's saved queue.
		x.QueueOrder = store.data.DrivingConfig.QueueOrder
		store.data.DrivingConfig = x
		store.auditLocked(auth.currentUsername(r), "Modification", "Paramètres", "Alertes et horaires")
		if !store.commitDrivingLocked(w, before) {
			return
		}
		okJSON(w, x)
	})
	mux.HandleFunc("/api/driving/invoice", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			errJSON(w, 405, "Méthode non autorisée")
			return
		}
		var x DrivingInvoice
		if readJSON(r, &x) != nil || x.StudentID == 0 || strings.TrimSpace(x.Label) == "" || x.Amount <= 0 {
			errJSON(w, 400, "Facture invalide")
			return
		}
		if x.Date == "" {
			x.Date = time.Now().Format("2006-01-02")
		}
		if x.Status != "Annulée" {
			x.Status = "À payer"
		}
		x.CreatedBy = auth.currentUsername(r)
		store.mu.Lock()
		defer store.mu.Unlock()
		before := store.drivingSnapshotLocked()
		exists := false
		for _, student := range store.data.Students {
			if student.ID == x.StudentID {
				exists = true
			}
		}
		if !exists {
			errJSON(w, 404, "Candidat introuvable")
			return
		}
		linked := 0.0
		for _, payment := range store.data.DrivingPayments {
			if x.ID != 0 && payment.InvoiceID == x.ID {
				if payment.StudentID != x.StudentID {
					errJSON(w, 409, "Cette facture possède déjà des paiements pour un autre candidat")
					return
				}
				linked += payment.Amount
			}
		}
		if linked > x.Amount+.0005 || (linked > 0 && x.Status == "Annulée") {
			errJSON(w, 409, "Cette facture possède des paiements. Réaffectez les paiements avant de la réduire ou de l’annuler.")
			return
		}
		if x.ID == 0 {
			x.ID = nextDrivingID(store.data.DrivingInvoices, func(v DrivingInvoice) int { return v.ID })
			store.data.DrivingInvoices = append(store.data.DrivingInvoices, x)
		} else {
			for i := range store.data.DrivingInvoices {
				if store.data.DrivingInvoices[i].ID == x.ID {
					store.data.DrivingInvoices[i] = x
				}
			}
		}
		store.auditLocked(auth.currentUsername(r), "Enregistrement", "Facture", x.Label)
		if !store.commitDrivingLocked(w, before) {
			return
		}
		for _, current := range store.data.DrivingInvoices {
			if current.ID == x.ID {
				x = current
				break
			}
		}
		okJSON(w, x)
	})
	mux.HandleFunc("/api/driving/invoice.pdf", func(w http.ResponseWriter, r *http.Request) {
		id, _ := strconv.Atoi(r.URL.Query().Get("id"))
		data := store.snapshot()
		for _, x := range data.DrivingInvoices {
			if x.ID == id {
				student := studentName(data.Students, x.StudentID)
				instructor := "Non renseigné"
				for _, m := range data.Instructors {
					if m.ID == x.InstructorID {
						instructor = m.Name
					}
				}
				serveDrivingPDF(w, fmt.Sprintf("Facture-%06d.pdf", x.ID), data.Settings.Business, "FACTURE CLIENT", []string{fmt.Sprintf("Facture N° %06d", x.ID), "Date : " + formatDrivingDate(x.Date), "Candidat : " + student, "Moniteur : " + instructor, "Désignation : " + x.Label, fmt.Sprintf("MONTANT TOTAL : %.3f DT", x.Amount), "Statut : " + x.Status, data.DrivingConfig.InvoiceNote})
				return
			}
		}
		http.NotFound(w, r)
	})
	mux.HandleFunc("/api/driving/appointment.pdf", func(w http.ResponseWriter, r *http.Request) {
		id, _ := strconv.Atoi(r.URL.Query().Get("id"))
		data := store.snapshot()
		for _, x := range data.Lessons {
			if x.ID == id {
				serveDrivingPDF(w, fmt.Sprintf("Rendez-vous-%06d.pdf", x.ID), data.Settings.Business, "BON DE RENDEZ-VOUS", []string{"Candidat : " + studentName(data.Students, x.StudentID), "Date et heure : " + formatDrivingDate(x.Start), "Type de séance : " + x.Kind, "Moniteur : " + instructorName(data.Instructors, x.InstructorID), "Véhicule : " + vehicleName(data.Vehicles, x.VehicleID), fmt.Sprintf("Durée : %d minutes", x.Duration), data.DrivingConfig.AppointmentNote})
				return
			}
		}
		http.NotFound(w, r)
	})
	mux.HandleFunc("/api/driving/certificate.pdf", func(w http.ResponseWriter, r *http.Request) {
		id, _ := strconv.Atoi(r.URL.Query().Get("id"))
		data := store.snapshot()
		for _, x := range data.Exams {
			if x.ID == id {
				if x.Result != "Réussi" {
					errJSON(w, 409, "Le certificat exige un résultat Réussi")
					return
				}
				serveStudentCertificate(w, data, x)
				return
			}
		}
		http.NotFound(w, r)
	})
	mux.HandleFunc("/api/driving/report.csv", func(w http.ResponseWriter, r *http.Request) {
		data := store.snapshot()
		from, to := r.URL.Query().Get("from"), r.URL.Query().Get("to")
		inside := func(value string) bool {
			key := value
			if len(key) > 10 {
				key = key[:10]
			}
			return (from == "" || key >= from) && (to == "" || key <= to)
		}
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", "attachment; filename=rapport-autoecole.csv")
		_, _ = w.Write([]byte{0xEF, 0xBB, 0xBF})
		_, _ = io.WriteString(w, "Type;Date;Candidat;Libellé;Montant DT\r\n")
		for _, x := range data.DrivingPayments {
			if !inside(x.Time) {
				continue
			}
			_, _ = fmt.Fprintf(w, "Paiement;%s;%s;%s;%.3f\r\n", csvCell(x.Time), csvCell(studentName(data.Students, x.StudentID)), csvCell(x.Method), x.Amount)
		}
		for _, x := range data.Expenses {
			vehicleExpense := strings.HasPrefix(x.Category, "Véhicule ·") || x.Category == "Carburant" || x.Category == "Entretien" || x.Category == "Assurance"
			if !inside(x.Time) || !vehicleExpense {
				continue
			}
			_, _ = fmt.Fprintf(w, "Dépense;%s;;%s;-%.3f\r\n", csvCell(x.Time), csvCell(x.Label), x.Amount)
		}
	})
}

func (s *Store) refreshStudentDocumentStatusLocked(studentID int) {
	required := map[string]bool{"CIN": false, "Photo": false, "Certificat médical": false, "Contrat": false, "Justificatif": false}
	for _, document := range s.data.DrivingDocuments {
		if document.StudentID == studentID {
			if _, ok := required[document.Category]; ok {
				if document.Expiry == "" || document.Expiry >= time.Now().Format("2006-01-02") {
					required[document.Category] = true
				}
			}
		}
	}
	complete := true
	for _, present := range required {
		complete = complete && present
	}
	for i := range s.data.Students {
		if s.data.Students[i].ID == studentID {
			if complete {
				s.data.Students[i].DocumentStatus = "Complet"
			} else {
				s.data.Students[i].DocumentStatus = "Incomplet"
			}
		}
	}
}

func csvCell(value string) string { return `"` + strings.ReplaceAll(value, `"`, `""`) + `"` }

func studentName(rows []DrivingStudent, id int) string {
	for _, x := range rows {
		if x.ID == id {
			return x.Name
		}
	}
	return ""
}
func instructorName(rows []DrivingInstructor, id int) string {
	for _, x := range rows {
		if x.ID == id {
			return x.Name
		}
	}
	return "Non renseigné"
}
func vehicleName(rows []DrivingVehicle, id int) string {
	for _, x := range rows {
		if x.ID == id {
			return strings.TrimSpace(x.Brand + " " + x.Model + " · " + x.Plate)
		}
	}
	return "Non renseigné"
}

func serveDrivingPDF(w http.ResponseWriter, filename string, business BusinessProfile, title string, lines []string) {
	if pdf, err := buildDrivingUnicodePDF(business, title, lines); err == nil && len(pdf) > 0 {
		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
		_, _ = w.Write(pdf)
		return
	}
	var content strings.Builder
	logo, _ := newPDFImage(currentStoreLogo())
	primary := pdfRGB(business.InvoicePrimary, "0.05 0.12 0.18")
	accent := pdfRGB(business.InvoiceAccent, "0.93 0.38 0.12")
	content.WriteString(primary + " rg 36 724 523 88 re f\n")
	content.WriteString("1 1 1 rg 43 733 116 70 re f\n")
	if logo != nil {
		content.WriteString("q 104 0 0 66 49 735 cm /Logo Do Q\n")
	}
	pdfText(&content, 180, 776, 19, strings.ToUpper(business.Name), "1 1 1")
	pdfText(&content, 180, 752, 9, pdfShort(business.Address+" · "+business.WhatsApp+" · "+business.Email, 62), "0.84 0.89 0.93")
	pdfText(&content, 42, 684, 18, title, accent)
	content.WriteString(accent + " rg 42 668 94 4 re f\n")
	y := 628
	for i, line := range lines {
		size := 11
		color := primary
		if i == len(lines)-1 || strings.Contains(line, "TOTAL") {
			size = 14
			color = accent
		}
		pdfText(&content, 42, float64(y), size, line, color)
		y -= 34
	}
	content.WriteString(primary + " rg 36 32 523 38 re f\n")
	pdfText(&content, 160, 47, 9, "DOCUMENT OFFICIEL · SIGNATURE ET CACHET", "1 1 1")
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	_, _ = w.Write(buildPDFFromStreamsWithImage([]string{content.String()}, logo))
}

func buildDrivingUnicodePDFWithEdge(business BusinessProfile, title string, lines []string) ([]byte, error) {
	logoBytes := currentStoreLogo()
	mime := "image/png"
	if len(logoBytes) > 2 && logoBytes[0] == 0xff && logoBytes[1] == 0xd8 {
		mime = "image/jpeg"
	}
	logo := "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(logoBytes)
	primary := business.InvoicePrimary
	if !validThemeColor(primary) {
		primary = "#101d2c"
	}
	accent := business.InvoiceAccent
	if !validThemeColor(accent) {
		accent = "#ff6b35"
	}
	escape := func(value string) string {
		return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;", "'", "&#39;").Replace(value)
	}
	var rows strings.Builder
	for i, line := range lines {
		class := "row"
		if strings.Contains(strings.ToUpper(line), "TOTAL") || strings.Contains(strings.ToUpper(line), "MONTANT REÇU") {
			class += " total"
		}
		if i == len(lines)-1 {
			class += " note"
		}
		fmt.Fprintf(&rows, `<div class="%s">%s</div>`, class, escape(line))
	}
	html := fmt.Sprintf(`<!doctype html><html lang="fr"><head><meta charset="utf-8"><style>@page{size:A4;margin:0}*{box-sizing:border-box}body{margin:0;font-family:"Segoe UI Variable","Segoe UI",Arial,sans-serif;color:#172431;background:#fff}.page{width:210mm;min-height:297mm;padding:13mm 16mm 19mm;position:relative}.head{min-height:43mm;background:linear-gradient(128deg,%s,#244b63);border-radius:5mm;display:flex;align-items:center;padding:7mm;color:#fff;box-shadow:0 3mm 9mm #17364b20}.logo{width:34mm;height:28mm;object-fit:contain;background:#fff;border-radius:4mm;padding:2mm}.identity{flex:1;text-align:center;padding:0 34mm 0 8mm}.identity h1{font-size:20pt;font-weight:900;margin:0 0 2mm;text-transform:uppercase;letter-spacing:.4pt}.identity p{font-size:9pt;font-style:italic;font-weight:600;margin:1.2mm 0;color:#e7f0f4}.title{text-align:center;color:%s;font-size:23pt;font-weight:900;font-style:italic;margin:13mm 0 3mm}.rule{width:34mm;height:1.4mm;background:%s;margin:0 auto 9mm;border-radius:2mm}.sheet{border:1px solid #d9e2e7;border-radius:4mm;padding:5mm 8mm;box-shadow:0 2mm 8mm #18364a12}.row{padding:3.6mm 2mm;border-bottom:1px solid #e7edf0;font-size:12pt;font-weight:650;text-align:center;line-height:1.45}.row:nth-child(even){background:#f8fafb}.row:last-child{border:0}.total{font-size:17pt;font-weight:900;color:%s;text-align:center;background:#fff4ed;border:1px solid #ffd9c8;border-radius:3mm;margin:3mm 0}.note{text-align:center;font-style:italic;font-weight:600;color:#667784;border:0;background:#fff!important;font-size:10pt}.sign{position:absolute;bottom:32mm;right:19mm;width:58mm;text-align:center;color:#667784;font-style:italic;font-weight:700;border-top:1px solid #8c989f;padding-top:2mm}.foot{position:absolute;bottom:0;left:0;right:0;height:15mm;background:%s;color:#fff;display:flex;align-items:center;justify-content:center;font-size:9pt;font-weight:700;font-style:italic;letter-spacing:.5px}</style></head><body><main class="page"><header class="head"><img class="logo" src="%s"><div class="identity"><h1>%s</h1><p>%s</p><p>%s · %s · MF: %s</p></div></header><h2 class="title">%s</h2><div class="rule"></div><section class="sheet">%s</section><div class="sign">Signature et cachet</div><footer class="foot">Document officiel - %s</footer></main></body></html>`, primary, accent, accent, accent, primary, logo, escape(business.Name), escape(business.Address), escape(business.WhatsApp), escape(business.Email), escape(business.TaxID), escape(title), rows.String(), escape(business.Name))
	return renderDrivingHTMLPDF(html)
}

func renderDrivingHTMLPDF(html string) ([]byte, error) {
	edge := ""
	for _, candidate := range []string{
		filepath.Join(os.Getenv("ProgramFiles(x86)"), "Microsoft", "Edge", "Application", "msedge.exe"),
		filepath.Join(os.Getenv("ProgramFiles"), "Microsoft", "Edge", "Application", "msedge.exe"),
		filepath.Join(os.Getenv("LOCALAPPDATA"), "Microsoft", "Edge", "Application", "msedge.exe"),
	} {
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			edge = candidate
			break
		}
	}
	if edge == "" {
		return nil, fmt.Errorf("moteur PDF Unicode indisponible")
	}
	tmp, err := os.MkdirTemp("", "pilot-pro-pdf-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmp)
	htmlPath, pdfPath := filepath.Join(tmp, "document.html"), filepath.Join(tmp, "document.pdf")
	if err = os.WriteFile(htmlPath, []byte(html), 0600); err != nil {
		return nil, err
	}
	fileURL := (&url.URL{Scheme: "file", Path: "/" + filepath.ToSlash(htmlPath)}).String()
	cmd := exec.Command(edge, "--headless=new", "--disable-gpu", "--no-pdf-header-footer", "--user-data-dir="+filepath.Join(tmp, "profile"), "--print-to-pdf="+pdfPath, fileURL)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if output, runErr := cmd.CombinedOutput(); runErr != nil {
		return nil, fmt.Errorf("création PDF: %v (%s)", runErr, strings.TrimSpace(string(output)))
	}
	return os.ReadFile(pdfPath)
}
