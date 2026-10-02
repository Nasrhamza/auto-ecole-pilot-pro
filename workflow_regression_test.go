package main

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"path/filepath"
	"testing"
)

func TestLegacyShortPasswordCanStillLogin(t *testing.T) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		t.Fatal(err)
	}
	const iterations = 120000
	password := "1234"
	auth := &AuthManager{sessions: map[string]AuthSession{}}
	auth.config.Users = []AuthUser{{
		Username:   "hamza",
		Salt:       base64.RawStdEncoding.EncodeToString(salt),
		Hash:       base64.RawStdEncoding.EncodeToString(pbkdf2SHA256([]byte(password), salt, iterations, 32)),
		Iterations: iterations,
		Role:       "Administrateur",
		Active:     true,
	}}
	if _, err := auth.login("  HAMZA  ", password); err != nil {
		t.Fatalf("legacy login rejected: %v", err)
	}
}

func TestPermanentStudentDeletionRemovesAllLinkedData(t *testing.T) {
	store, mux := coachFixture(t)
	store.path = filepath.Join(t.TempDir(), "data.json")
	store.data.Students = append(store.data.Students, DrivingStudent{ID: 2, Name: "Candidat conservé", Status: "Actif"})
	store.data.Lessons = []DrivingLesson{{ID: 1, StudentID: 1}, {ID: 2, StudentID: 2}}
	store.data.Exams = []DrivingExam{{ID: 1, StudentID: 1}, {ID: 2, StudentID: 2}}
	store.data.DrivingPayments = []DrivingPayment{{ID: 1, StudentID: 1}, {ID: 2, StudentID: 2}}
	store.data.DrivingDocuments = []DrivingDocument{{ID: 1, StudentID: 1}, {ID: 2, StudentID: 2}}
	store.data.PackageSales = []DrivingPackageSale{{ID: 1, StudentID: 1}, {ID: 2, StudentID: 2}}
	store.data.DrivingInvoices = []DrivingInvoice{{ID: 1, StudentID: 1}, {ID: 2, StudentID: 2}}
	store.data.DrivingConfig.QueueOrder = []int{1, 2}
	store.data.DrivingAudits = []DrivingAudit{{ID: 1, Detail: "Élève : dossier ouvert"}, {ID: 2, Detail: "Candidat conservé : dossier ouvert"}}

	req, err := http.NewRequest(http.MethodDelete, "http://local.test/api/driving/student?id=1&permanent=1", nil)
	if err != nil {
		t.Fatal(err)
	}
	w := &coachRecorder{headers: http.Header{}}
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("permanent deletion failed: %d %s", w.Code, w.Body.String())
	}
	if len(store.data.Students) != 1 || store.data.Students[0].ID != 2 ||
		len(store.data.Lessons) != 1 || store.data.Lessons[0].StudentID != 2 ||
		len(store.data.Exams) != 1 || store.data.Exams[0].StudentID != 2 ||
		len(store.data.DrivingPayments) != 1 || store.data.DrivingPayments[0].StudentID != 2 ||
		len(store.data.DrivingDocuments) != 1 || store.data.DrivingDocuments[0].StudentID != 2 ||
		len(store.data.PackageSales) != 1 || store.data.PackageSales[0].StudentID != 2 ||
		len(store.data.DrivingInvoices) != 1 || store.data.DrivingInvoices[0].StudentID != 2 ||
		len(store.data.DrivingConfig.QueueOrder) != 1 || store.data.DrivingConfig.QueueOrder[0] != 2 ||
		len(store.data.DrivingAudits) != 1 || store.data.DrivingAudits[0].ID != 2 {
		t.Fatalf("linked data remains after permanent deletion: %#v", store.data)
	}
}

func TestLessonDeletionOnlyRemovesSelectedLesson(t *testing.T) {
	store, mux := coachFixture(t)
	store.path = filepath.Join(t.TempDir(), "data.json")
	store.data.Lessons = []DrivingLesson{{ID: 1, StudentID: 1}, {ID: 2, StudentID: 1}}
	store.data.Exams = []DrivingExam{{ID: 1, StudentID: 1}}
	store.data.DrivingPayments = []DrivingPayment{{ID: 1, StudentID: 1}}
	store.data.DrivingDocuments = []DrivingDocument{{ID: 1, StudentID: 1}}

	req, err := http.NewRequest(http.MethodDelete, "http://local.test/api/driving/lesson?id=1", nil)
	if err != nil {
		t.Fatal(err)
	}
	w := &coachRecorder{headers: http.Header{}}
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("lesson deletion failed: %d %s", w.Code, w.Body.String())
	}
	if len(store.data.Students) != 1 || store.data.Students[0].ID != 1 {
		t.Fatalf("candidate was changed while deleting a lesson: %#v", store.data.Students)
	}
	if len(store.data.Lessons) != 1 || store.data.Lessons[0].ID != 2 {
		t.Fatalf("wrong lessons after single deletion: %#v", store.data.Lessons)
	}
	if len(store.data.Exams) != 1 || len(store.data.DrivingPayments) != 1 || len(store.data.DrivingDocuments) != 1 {
		t.Fatalf("candidate-linked data was changed while deleting a lesson: %#v", store.data)
	}
}

func TestDrivingOnlyBackupValidation(t *testing.T) {
	data := StoreData{
		Students: []DrivingStudent{{ID: 1, Name: "Candidat", Status: "Actif"}},
		Lessons:  []DrivingLesson{{ID: 1, StudentID: 1, Start: "2026-09-14T09:00", Duration: 60}},
	}
	candidate := &Store{data: data}
	candidate.ensureDefaults()
	if err := validateRestoredData(&candidate.data); err != nil {
		t.Fatalf("driving-only backup rejected: %v", err)
	}
	candidate.data.Lessons[0].StudentID = 999
	if err := validateRestoredData(&candidate.data); err == nil {
		t.Fatal("backup with an orphan lesson was accepted")
	}
}
