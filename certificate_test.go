package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCertificateTemplate(t *testing.T) {
	s, _ := coachFixture(t)
	s.data.Settings.Business = BusinessProfile{Name: "Auto-École Walid Nafougui"}
	s.data.Students[0].Name = "Élève Démonstration"
	s.data.Students[0].CIN = "EXEMPLE"
	s.data.Students[0].Permit = "B"
	s.data.Students[0].BirthDate = "2000-01-15"
	s.data.DrivingConfig.CertificateClosing = "Le présent certificat est délivré à l'intéressé pour servir et valoir ce que de droit."
	s.data.Lessons = []DrivingLesson{{StudentID: 1, Start: "2026-08-01T09:00", Duration: 750, Status: "Terminée"}, {StudentID: 1, Start: "2026-08-02T09:00", Duration: 60, Status: "Absence"}}
	exam := DrivingExam{ID: 1, StudentID: 1, Type: "Conduite", Date: "2026-09-12", Result: "Réussi", Center: "Centre d’examen"}
	html, err := certificateHTML(s.data, exam, storeLogoPNG)
	if err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{"Élève Démonstration", "12,5 heures", "A4 landscape", "grayscale(1)", "CERT-000001"} {
		if !strings.Contains(html, value) {
			t.Fatal("missing", value)
		}
	}
	if strings.Contains(html, "{{") {
		t.Fatal("unresolved template")
	}
	if target := os.Getenv("PILOT_CERT_PREVIEW"); target != "" {
		started := time.Now()
		pdf, e := buildStudentCertificatePDF(s.data, exam, storeLogoPNG)
		if e != nil {
			t.Fatal(e)
		}
		if e = os.MkdirAll(filepath.Dir(target), 0755); e != nil {
			t.Fatal(e)
		}
		if e = os.WriteFile(target, pdf, 0644); e != nil {
			t.Fatal(e)
		}
		if elapsed := time.Since(started); elapsed > 2*time.Second {
			t.Fatalf("native certificate too slow: %v", elapsed)
		}
	}
	s.data.Students[0].Name = "<script> & {{.School}}"
	html, err = certificateHTML(s.data, exam, storeLogoPNG)
	if err != nil || !strings.Contains(html, "&lt;script&gt; &amp; {{.School}}") {
		t.Fatal("unsafe substitution", err)
	}
	exam.Result = "Échec"
	if _, err = certificateHTML(s.data, exam, storeLogoPNG); err == nil {
		t.Fatal("failed exam certificate allowed")
	}
}

func TestNativeAdministrativePDF(t *testing.T) {
	started := time.Now()
	pdf, err := buildDrivingUnicodePDF(BusinessProfile{Name: "Auto-École Démonstration"}, "FACTURE CLIENT", []string{"Candidat : Élève Démonstration", "MONTANT TOTAL : 150.000 DT", "Document généré localement."})
	if err != nil || len(pdf) < 1000 || string(pdf[:4]) != "%PDF" {
		t.Fatalf("native PDF failed: bytes=%d err=%v", len(pdf), err)
	}
	if time.Since(started) > 2*time.Second {
		t.Fatal("native PDF generation is too slow")
	}
}
