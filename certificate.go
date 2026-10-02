package main

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

type studentCertificate struct {
	School, Address, Phone, Email, TaxID                     string
	Name, CIN, BirthDate, Permit, ExamType, ExamDate, Center string
	Intro, Closing, Period, Hours, Number, Issued, NameClass string
	Logo                                                     string
}

func certificateDate(value string) string {
	date, err := time.Parse("2006-01-02", strings.Split(value, "T")[0])
	if err != nil {
		return value
	}
	months := []string{"janvier", "février", "mars", "avril", "mai", "juin", "juillet", "août", "septembre", "octobre", "novembre", "décembre"}
	return fmt.Sprintf("%d %s %d", date.Day(), months[int(date.Month())-1], date.Year())
}

func certificateHTML(data StoreData, exam DrivingExam, logo []byte) (string, error) {
	var pupil *DrivingStudent
	for i := range data.Students {
		if data.Students[i].ID == exam.StudentID {
			pupil = &data.Students[i]
			break
		}
	}
	if pupil == nil {
		return "", fmt.Errorf("dossier du candidat introuvable")
	}
	if exam.Result != "Réussi" {
		return "", fmt.Errorf("le certificat exige un résultat Réussi")
	}
	b := data.Settings.Business
	c := studentCertificate{School: b.Name, Address: b.Address, Phone: b.WhatsApp, Email: b.Email, TaxID: b.TaxID,
		Name: pupil.Name, CIN: pupil.CIN, Permit: pupil.Permit, ExamType: exam.Type, ExamDate: certificateDate(exam.Date), Center: exam.Center,
		Intro: data.DrivingConfig.CertificateIntro, Closing: data.DrivingConfig.CertificateClosing,
		Number: fmt.Sprintf("CERT-%06d", exam.ID), Issued: certificateDate(time.Now().Format("2006-01-02"))}
	if c.School == "" {
		c.School = storeDisplayName
	}
	if c.Intro == "" {
		c.Intro = "Nous certifions que le candidat :"
	}
	if c.Closing == "" {
		c.Closing = "Ce certificat est délivré à l’intéressé pour servir et valoir ce que de droit."
	}
	if pupil.BirthDate != "" {
		c.BirthDate = certificateDate(pupil.BirthDate)
	}
	if utf8.RuneCountInString(c.Name) > 45 {
		c.NameClass = "long-name"
	}
	first, last, minutes := "", "", 0
	for _, l := range data.Lessons {
		date := strings.Split(l.Start, "T")[0]
		if l.StudentID != pupil.ID || l.Status != "Terminée" || date > strings.Split(exam.Date, "T")[0] {
			continue
		}
		minutes += l.Duration
		if first == "" || date < first {
			first = date
		}
		if date > last {
			last = date
		}
	}
	if first != "" {
		c.Period = "Du " + certificateDate(first) + " au " + certificateDate(last)
	}
	c.Hours = strings.ReplaceAll(fmt.Sprintf("%g", float64(minutes)/60), ".", ",") + " heures enregistrées"
	mime := "image/png"
	if len(logo) > 2 && logo[0] == 0xff && logo[1] == 0xd8 {
		mime = "image/jpeg"
	}
	c.Logo = "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(logo)
	source, err := webFS.ReadFile("web/certificate.html")
	if err != nil {
		return "", err
	}
	values := map[string]string{"School": c.School, "Address": c.Address, "Phone": c.Phone, "Email": c.Email, "TaxID": c.TaxID, "Name": c.Name, "CIN": c.CIN, "BirthDate": c.BirthDate, "Permit": c.Permit, "ExamType": c.ExamType, "ExamDate": c.ExamDate, "Center": c.Center, "Intro": c.Intro, "Closing": c.Closing, "Period": c.Period, "Hours": c.Hours, "Number": c.Number, "Issued": c.Issued, "NameClass": c.NameClass, "Logo": c.Logo}
	optional := regexp.MustCompile(`(?s)\{\{if \.([A-Za-z]+)\}\}(.*?)\{\{end\}\}`)
	markup := optional.ReplaceAllStringFunc(string(source), func(block string) string {
		m := optional.FindStringSubmatch(block)
		if values[m[1]] == "" {
			return ""
		}
		return m[2]
	})
	escape := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", "\"", "&quot;", "'", "&#39;")
	variables := regexp.MustCompile(`\{\{\.([A-Za-z]+)\}\}`)
	return variables.ReplaceAllStringFunc(markup, func(token string) string { return escape.Replace(values[variables.FindStringSubmatch(token)[1]]) }), nil
}

func serveStudentCertificate(w http.ResponseWriter, data StoreData, exam DrivingExam) {
	pdf, err := buildStudentCertificatePDF(data, exam, currentStoreLogo())
	if err != nil {
		errJSON(w, 400, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"Certificat-Reussite-%06d.pdf\"", exam.ID))
	_, _ = w.Write(pdf)
}
