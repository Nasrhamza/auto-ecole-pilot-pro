package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/phpdave11/gofpdf"
)

const pdfFontFamily = "PilotSans"

func newNativePDF(orientation string) (*gofpdf.Fpdf, error) {
	pdf := gofpdf.New(orientation, "mm", "A4", "")
	fontDir := filepath.Join(os.Getenv("WINDIR"), "Fonts")
	if fontDir == "Fonts" {
		fontDir = `C:\Windows\Fonts`
	}
	fonts := []struct{ style, name string }{{"", "segoeui.ttf"}, {"B", "segoeuib.ttf"}, {"I", "segoeuii.ttf"}}
	for _, font := range fonts {
		path := filepath.Join(fontDir, font.name)
		if _, err := os.Stat(path); err != nil {
			return nil, fmt.Errorf("police Windows introuvable: %s", font.name)
		}
		pdf.AddUTF8Font(pdfFontFamily, font.style, path)
	}
	if pdf.Error() != nil {
		return nil, pdf.Error()
	}
	pdf.SetMargins(14, 14, 14)
	pdf.SetAutoPageBreak(false, 12)
	pdf.SetTitle("AutoEcole Pilot Pro - Document PDF", true)
	pdf.SetAuthor(storeDisplayName, true)
	return pdf, nil
}

func pdfOutput(pdf *gofpdf.Fpdf) ([]byte, error) {
	var out bytes.Buffer
	if err := pdf.Output(&out); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func nativeLogo(pdf *gofpdf.Fpdf, data []byte, name string, x, y, w, h float64) error {
	if len(data) == 0 {
		return nil
	}
	kind := "PNG"
	if len(data) > 2 && data[0] == 0xff && data[1] == 0xd8 {
		kind = "JPG"
	}
	info := pdf.RegisterImageOptionsReader(name, gofpdf.ImageOptions{ImageType: kind, ReadDpi: true}, bytes.NewReader(data))
	if info == nil || pdf.Error() != nil {
		return pdf.Error()
	}
	pdf.ImageOptions(name, x, y, w, h, false, gofpdf.ImageOptions{ImageType: kind, ReadDpi: true}, 0, "")
	return pdf.Error()
}

func grayscalePNG(source []byte) ([]byte, error) {
	img, _, err := image.Decode(bytes.NewReader(source))
	if err != nil {
		return nil, err
	}
	b := img.Bounds()
	gray := image.NewGray(b)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, blue, a := img.At(x, y).RGBA()
			// Composite transparency over white before conversion. This avoids
			// black rectangles around transparent PNG logos in PDF readers.
			r += 0xffff - a
			g += 0xffff - a
			blue += 0xffff - a
			luma := uint8(((299*r + 587*g + 114*blue) / 1000) >> 8)
			gray.SetGray(x, y, color.Gray{Y: luma})
		}
	}
	var out bytes.Buffer
	if err := png.Encode(&out, gray); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func nativeCentered(pdf *gofpdf.Fpdf, y, size float64, style, text string) {
	pdf.SetFont(pdfFontFamily, style, size)
	pdf.SetXY(22, y)
	pdf.CellFormat(253, size*.42, text, "", 0, "C", false, 0, "")
}

func drawCertificateCorner(pdf *gofpdf.Fpdf, x, y, sx, sy float64) {
	pdf.SetDrawColor(25, 25, 25)
	pdf.SetLineWidth(.25)
	for i := 0; i < 4; i++ {
		o := float64(i) * 3.2
		pdf.CurveBezierCubic(x, y+sy*o, x+sx*9, y+sy*(2+o), x+sx*(22-o), y+sy*14, x+sx*(27-o), y+sy*28, "D")
	}
	for i := 0; i < 5; i++ {
		t := float64(i)
		lx := x + sx*(5+t*4.2)
		ly := y + sy*(5+t*4.4)
		pdf.Ellipse(lx, ly, 2.8, 1.25, sx*sy*(35+t*7), "D")
		pdf.Ellipse(lx+sx*3.2, ly+sy*1.2, 2.5, 1.1, sx*sy*(-25-t*5), "D")
	}
	pdf.Arc(x+sx*8, y+sy*8, 5, 5, 0, 15, 320, "D")
}

func drawCertificateOrnaments(pdf *gofpdf.Fpdf) {
	pdf.SetDrawColor(35, 35, 35)
	pdf.SetLineWidth(.22)
	pdf.Rect(7, 7, 283, 196, "D")
	pdf.Rect(9.2, 9.2, 278.6, 191.6, "D")
	drawCertificateCorner(pdf, 10, 10, 1, 1)
	drawCertificateCorner(pdf, 287, 10, -1, 1)
	drawCertificateCorner(pdf, 10, 200, 1, -1)
	drawCertificateCorner(pdf, 287, 200, -1, -1)
	pdf.SetLineWidth(.18)
	for i := 0; i < 3; i++ {
		o := float64(i) * 4
		pdf.CurveBezierCubic(92+o, 31, 110, 18+o, 130-o, 24, 148.5, 34-o, "D")
		pdf.CurveBezierCubic(205-o, 31, 187, 18+o, 167+o, 24, 148.5, 34-o, "D")
	}
	pdf.Ellipse(148.5, 34, 2.4, 2.4, 0, "D")
	pdf.Line(112, 190, 185, 190)
}

func buildStudentCertificatePDF(data StoreData, exam DrivingExam, logoBytes []byte) ([]byte, error) {
	var pupil *DrivingStudent
	for i := range data.Students {
		if data.Students[i].ID == exam.StudentID {
			pupil = &data.Students[i]
			break
		}
	}
	if pupil == nil {
		return nil, fmt.Errorf("dossier du candidat introuvable")
	}
	if exam.Result != "Réussi" {
		return nil, fmt.Errorf("le certificat exige un résultat Réussi")
	}
	pdf, err := newNativePDF("L")
	if err != nil {
		return nil, err
	}
	pdf.AddPage()
	drawCertificateOrnaments(pdf)
	gray, err := grayscalePNG(logoBytes)
	if err == nil {
		_ = nativeLogo(pdf, gray, "certificate-logo", 137.5, 12, 22, 22)
	}
	business := data.Settings.Business
	if business.Name == "" {
		business.Name = storeDisplayName
	}
	nativeCentered(pdf, 39, 25, "B", "CERTIFICAT DE RÉUSSITE")
	nativeCentered(pdf, 51, 14, "B", business.Name)
	pdf.Line(126, 61, 171, 61)
	intro := data.DrivingConfig.CertificateIntro
	if intro == "" {
		intro = "Nous certifions que le candidat :"
	}
	nativeCentered(pdf, 65, 11, "", intro)
	nameSize := 23.0
	if len([]rune(pupil.Name)) > 40 {
		nameSize = 18
	}
	nativeCentered(pdf, 76, nameSize, "I", strings.ToUpper(pupil.Name))
	identity := []string{}
	if pupil.CIN != "" {
		identity = append(identity, "CIN : "+pupil.CIN)
	}
	if pupil.BirthDate != "" {
		identity = append(identity, "Né(e) le "+certificateDate(pupil.BirthDate))
	}
	if pupil.Permit != "" {
		identity = append(identity, "Catégorie "+pupil.Permit)
	}
	nativeCentered(pdf, 91, 10, "", strings.Join(identity, "   -   "))
	nativeCentered(pdf, 102, 13, "", "a réussi l’épreuve de "+exam.Type)
	result := "le " + certificateDate(exam.Date)
	if exam.Center != "" {
		result += ", au centre " + exam.Center
	}
	nativeCentered(pdf, 111, 12, "B", result)
	first, last, minutes := "", "", 0
	for _, lesson := range data.Lessons {
		date := strings.Split(lesson.Start, "T")[0]
		if lesson.StudentID != pupil.ID || lesson.Status != "Terminée" || date > strings.Split(exam.Date, "T")[0] {
			continue
		}
		minutes += lesson.Duration
		if first == "" || date < first {
			first = date
		}
		if last == "" || date > last {
			last = date
		}
	}
	pdf.SetFont(pdfFontFamily, "B", 10)
	pdf.SetXY(61, 127)
	pdf.CellFormat(82, 5, "PÉRIODE DE FORMATION ENREGISTRÉE", "", 0, "C", false, 0, "")
	pdf.SetXY(154, 127)
	pdf.CellFormat(82, 5, "DURÉE DES SÉANCES TERMINÉES", "", 0, "C", false, 0, "")
	pdf.SetFont(pdfFontFamily, "", 10)
	period := "Non renseignée"
	if first != "" {
		period = "Du " + certificateDate(first) + " au " + certificateDate(last)
	}
	pdf.SetXY(61, 133)
	pdf.CellFormat(82, 5, period, "", 0, "C", false, 0, "")
	hours := strings.ReplaceAll(fmt.Sprintf("%g", float64(minutes)/60), ".", ",") + " heures enregistrées"
	pdf.SetXY(154, 133)
	pdf.CellFormat(82, 5, hours, "", 0, "C", false, 0, "")
	pdf.SetFont(pdfFontFamily, "B", 9)
	pdf.SetXY(28, 148)
	pdf.CellFormat(100, 5, business.Name, "", 0, "L", false, 0, "")
	pdf.SetFont(pdfFontFamily, "", 8)
	contact := strings.Join(filterPDFValues(business.Address, business.WhatsApp, business.Email, business.TaxID), " - ")
	pdf.SetXY(28, 154)
	pdf.MultiCell(100, 4.5, contact, "", "L", false)
	pdf.SetFont(pdfFontFamily, "B", 10)
	pdf.SetXY(173, 148)
	pdf.CellFormat(96, 5, "Délivré le "+certificateDate(timeNowDate()), "", 0, "C", false, 0, "")
	pdf.Line(187, 170, 255, 170)
	pdf.SetFont(pdfFontFamily, "I", 8)
	pdf.SetXY(187, 172)
	pdf.CellFormat(68, 5, "Signature du moniteur et cachet", "", 0, "C", false, 0, "")
	closing := data.DrivingConfig.CertificateClosing
	if closing == "" {
		closing = "Ce certificat est délivré à l’intéressé pour servir et valoir ce que de droit."
	}
	pdf.SetFont(pdfFontFamily, "I", 8)
	pdf.SetXY(42, 181)
	pdf.MultiCell(213, 4, closing, "", "C", false)
	nativeCentered(pdf, 193, 8, "", fmt.Sprintf("N° CERT-%06d", exam.ID))
	return pdfOutput(pdf)
}

func timeNowDate() string { return time.Now().Format("2006-01-02") }

func filterPDFValues(values ...string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			out = append(out, value)
		}
	}
	return out
}

func buildDrivingUnicodePDF(business BusinessProfile, title string, lines []string) ([]byte, error) {
	pdf, err := newNativePDF("P")
	if err != nil {
		return nil, err
	}
	pdf.AddPage()
	primaryR, primaryG, primaryB := 16, 42, 58
	pdf.SetFillColor(primaryR, primaryG, primaryB)
	pdf.Rect(0, 0, 210, 48, "F")
	_ = nativeLogo(pdf, currentStoreLogo(), "document-logo", 16, 10, 30, 25)
	if business.Name == "" {
		business.Name = storeDisplayName
	}
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont(pdfFontFamily, "B", 18)
	pdf.SetXY(52, 13)
	pdf.CellFormat(142, 8, strings.ToUpper(business.Name), "", 1, "C", false, 0, "")
	pdf.SetFont(pdfFontFamily, "I", 8)
	pdf.SetXY(52, 24)
	pdf.MultiCell(142, 4, strings.Join(filterPDFValues(business.Address, business.WhatsApp, business.Email, business.TaxID), " - "), "", "C", false)
	pdf.SetTextColor(16, 42, 58)
	pdf.SetFont(pdfFontFamily, "B", 20)
	pdf.SetXY(16, 60)
	pdf.CellFormat(178, 9, title, "B", 1, "C", false, 0, "")
	y := 82.0
	for i, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		style, size := "", 11.0
		upper := strings.ToUpper(line)
		if i == 0 {
			style = "B"
		}
		if strings.Contains(upper, "TOTAL") || strings.Contains(upper, "MONTANT REÇU") {
			style = "B"
			size = 15
		}
		if i == len(lines)-1 {
			style = "I"
			size = 9
		}
		pdf.SetFont(pdfFontFamily, style, size)
		pdf.SetXY(24, y)
		pdf.MultiCell(162, 7, line, "B", "C", false)
		y = pdf.GetY() + 4
	}
	pdf.SetFont(pdfFontFamily, "I", 8)
	pdf.SetXY(118, 260)
	pdf.CellFormat(66, 5, "Signature et cachet", "T", 0, "C", false, 0, "")
	pdf.SetFillColor(primaryR, primaryG, primaryB)
	pdf.Rect(0, 280, 210, 17, "F")
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont(pdfFontFamily, "B", 8)
	pdf.SetXY(20, 285)
	pdf.CellFormat(170, 5, "DOCUMENT OFFICIEL - "+business.Name, "", 0, "C", false, 0, "")
	return pdfOutput(pdf)
}
