package main

import (
	"bytes"
	"compress/zlib"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"embed"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"log"
	"net"
	"net/http"
	neturl "net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

//go:embed web/index.html web/icons/*.png web/coach.css web/coach.js web/certificate.html
var webFS embed.FS

//go:embed assets/logo.png
var storeLogoPNG []byte

const (
	storeDisplayName = "Auto-École Walid Nafougui"
	storeWhatsApp    = ""
	storeEmail       = ""
	storeTaxID       = ""
)

type Product struct {
	ID            int     `json:"id"`
	Name          string  `json:"name"`
	Barcode       string  `json:"barcode"`
	Category      string  `json:"category"`
	Kind          string  `json:"kind"` // stock|service
	PurchasePrice float64 `json:"purchase_price"`
	SalePrice     float64 `json:"sale_price"`
	Stock         int     `json:"stock"`
	MinStock      int     `json:"min_stock"`
	Pinned        bool    `json:"pinned"`
	Unit          string  `json:"unit"`
	SupplierID    int     `json:"supplier_id,omitempty"`
	ExpiryDate    string  `json:"expiry_date,omitempty"`
	PromoPrice    float64 `json:"promo_price,omitempty"`
	PromoUntil    string  `json:"promo_until,omitempty"`
}

type Client struct {
	ID    int     `json:"id"`
	Name  string  `json:"name"`
	Phone string  `json:"phone"`
	Debt  float64 `json:"debt"`
}

type SaleItem struct {
	ProductID int     `json:"product_id"`
	Name      string  `json:"name"`
	Qty       int     `json:"qty"`
	UnitPrice float64 `json:"unit_price"`
	Total     float64 `json:"total"`
	UnitCost  float64 `json:"unit_cost,omitempty"`
}

type Sale struct {
	ID          int        `json:"id"`
	Time        string     `json:"time"`
	Items       []SaleItem `json:"items"`
	Total       float64    `json:"total"`
	Payment     string     `json:"payment"` // cash|credit
	ClientID    int        `json:"client_id,omitempty"`
	Client      string     `json:"client,omitempty"`
	Phone       string     `json:"phone,omitempty"`
	Received    float64    `json:"received,omitempty"`
	Change      float64    `json:"change,omitempty"`
	Subtotal    float64    `json:"subtotal,omitempty"`
	Discount    float64    `json:"discount,omitempty"`
	Seller      string     `json:"seller,omitempty"`
	Status      string     `json:"status,omitempty"`
	CancelledAt string     `json:"cancelled_at,omitempty"`
	CancelledBy string     `json:"cancelled_by,omitempty"`
}

type Supplier struct {
	ID      int     `json:"id"`
	Name    string  `json:"name"`
	Phone   string  `json:"phone"`
	Address string  `json:"address"`
	Balance float64 `json:"balance"`
}

type PurchaseItem struct {
	ProductID int     `json:"product_id"`
	Name      string  `json:"name"`
	Qty       int     `json:"qty"`
	UnitCost  float64 `json:"unit_cost"`
	Total     float64 `json:"total"`
}

type Purchase struct {
	ID           int            `json:"id"`
	Time         string         `json:"time"`
	SupplierID   int            `json:"supplier_id"`
	SupplierName string         `json:"supplier_name"`
	Reference    string         `json:"reference"`
	Items        []PurchaseItem `json:"items"`
	Total        float64        `json:"total"`
	Paid         float64        `json:"paid"`
	Balance      float64        `json:"balance"`
	CreatedBy    string         `json:"created_by"`
}

type StockMovement struct {
	ID          int    `json:"id"`
	Time        string `json:"time"`
	ProductID   int    `json:"product_id"`
	ProductName string `json:"product_name"`
	Type        string `json:"type"`
	Qty         int    `json:"qty"`
	Before      int    `json:"before"`
	After       int    `json:"after"`
	Reference   string `json:"reference"`
	User        string `json:"user"`
}

type CashMovement struct {
	ID     int     `json:"id"`
	Time   string  `json:"time"`
	Type   string  `json:"type"`
	Amount float64 `json:"amount"`
	Reason string  `json:"reason"`
	User   string  `json:"user"`
}

type SIM struct {
	ID       int    `json:"id"`
	Label    string `json:"label"`
	Number   string `json:"number"`
	Operator string `json:"operator"`
	QuotaMB  int    `json:"quota_mb"`
	Active   bool   `json:"active"`
}

type Transfer struct {
	ID         int     `json:"id"`
	Time       string  `json:"time"`
	SIMID      int     `json:"sim_id"`
	SIMLabel   string  `json:"sim_label"`
	ClientName string  `json:"client_name,omitempty"`
	DestPhone  string  `json:"dest_phone,omitempty"`
	SoldBy     string  `json:"sold_by,omitempty"`
	PackageMB  int     `json:"package_mb"`
	Price      float64 `json:"price"`
	Cost       float64 `json:"cost"`
	Status     string  `json:"status"`
}

type DigitalService struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	URL         string `json:"url"`
	Description string `json:"description"`
	Category    string `json:"category"`
	Icon        string `json:"icon"`
}

type CartItem struct {
	ProductID  int     `json:"product_id"`
	Qty        int     `json:"qty"`
	LineID     string  `json:"line_id,omitempty"`
	CustomName string  `json:"custom_name,omitempty"`
	UnitPrice  float64 `json:"unit_price,omitempty"`
}

type PendingCart struct {
	Slot  int        `json:"slot"`
	Items []CartItem `json:"items"`
}

type PriceHistory struct {
	Time string  `json:"time"`
	Type string  `json:"type"`
	Item string  `json:"item"`
	Old  float64 `json:"old"`
	New  float64 `json:"new"`
}

type Settings struct {
	Conn500Price  float64         `json:"conn_500_price"`
	Conn500Cost   float64         `json:"conn_500_cost"`
	Conn1000Price float64         `json:"conn_1000_price"`
	Conn1000Cost  float64         `json:"conn_1000_cost"`
	Theme         string          `json:"theme"`
	Business      BusinessProfile `json:"business"`
}

type BusinessProfile struct {
	Name           string `json:"name"`
	Address        string `json:"address"`
	WhatsApp       string `json:"whatsapp"`
	Email          string `json:"email"`
	TaxID          string `json:"tax_id"`
	InvoicePrimary string `json:"invoice_primary"`
	InvoiceAccent  string `json:"invoice_accent"`
	LogoVersion    int64  `json:"logo_version"`
}

type StoreData struct {
	Products           []Product                  `json:"products"`
	Clients            []Client                   `json:"clients"`
	Sales              []Sale                     `json:"sales"`
	SIMs               []SIM                      `json:"sims"`
	Transfers          []Transfer                 `json:"transfers"`
	DigitalServices    []DigitalService           `json:"digital_services"`
	Carts              []PendingCart              `json:"carts"`
	PriceHistory       []PriceHistory             `json:"price_history"`
	Suppliers          []Supplier                 `json:"suppliers"`
	Purchases          []Purchase                 `json:"purchases"`
	StockMovements     []StockMovement            `json:"stock_movements"`
	CashMovements      []CashMovement             `json:"cash_movements"`
	Settings           Settings                   `json:"settings"`
	Students           []DrivingStudent           `json:"students"`
	Instructors        []DrivingInstructor        `json:"instructors"`
	Vehicles           []DrivingVehicle           `json:"vehicles"`
	Lessons            []DrivingLesson            `json:"lessons"`
	Exams              []DrivingExam              `json:"exams"`
	DrivingPayments    []DrivingPayment           `json:"driving_payments"`
	Expenses           []DrivingExpense           `json:"expenses"`
	DrivingDocuments   []DrivingDocument          `json:"driving_documents"`
	DrivingPackages    []DrivingPackage           `json:"driving_packages"`
	PackageSales       []DrivingPackageSale       `json:"package_sales"`
	VehicleEvents      []DrivingVehicleEvent      `json:"vehicle_events"`
	InstructorAbsences []DrivingInstructorAbsence `json:"instructor_absences"`
	DrivingAudits      []DrivingAudit             `json:"driving_audits"`
	DrivingInvoices    []DrivingInvoice           `json:"driving_invoices"`
	DrivingConfig      DrivingConfig              `json:"driving_config"`
}

type BackupBundle struct {
	Format        string            `json:"format"`
	CreatedAt     string            `json:"created_at"`
	Data          StoreData         `json:"data"`
	HasCustomLogo bool              `json:"has_custom_logo"`
	LogoBase64    string            `json:"logo_base64,omitempty"`
	DocumentFiles map[string]string `json:"document_files,omitempty"`
}

type Store struct {
	mu   sync.Mutex
	path string
	data StoreData
}

type AuthConfig struct {
	Username   string     `json:"username"`
	Salt       string     `json:"salt"`
	Hash       string     `json:"hash"`
	Iterations int        `json:"iterations"`
	Users      []AuthUser `json:"users,omitempty"`
}

type AuthUser struct {
	Username   string `json:"username"`
	Salt       string `json:"salt,omitempty"`
	Hash       string `json:"hash,omitempty"`
	Iterations int    `json:"iterations,omitempty"`
	Role       string `json:"role"`
	Active     bool   `json:"active"`
}

type AuthSession struct {
	Username string
	Role     string
	Expires  time.Time
}

type AuthManager struct {
	mu       sync.Mutex
	path     string
	config   AuthConfig
	sessions map[string]AuthSession
}

const authCookieName = "autoecole_pilot_session"

func authPath() string {
	return filepath.Join(appDataDir(), "auth.json")
}

func newAuthManager() *AuthManager {
	a := &AuthManager{path: authPath(), sessions: map[string]AuthSession{}}
	if b, err := os.ReadFile(a.path); err == nil {
		_ = json.Unmarshal(b, &a.config)
	}
	if len(a.config.Users) == 0 && a.config.Username != "" {
		a.config.Users = []AuthUser{{Username: a.config.Username, Salt: a.config.Salt, Hash: a.config.Hash, Iterations: a.config.Iterations, Role: "Administrateur", Active: true}}
		_ = a.saveLocked()
	}
	return a
}

func (a *AuthManager) configured() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return len(a.config.Users) > 0 || (a.config.Username != "" && a.config.Salt != "" && a.config.Hash != "")
}

func (a *AuthManager) setup(username, password string) error {
	username = strings.TrimSpace(username)
	if len(username) < 3 || len(password) < 8 {
		return fmt.Errorf("Nom administrateur (3 caractères) et mot de passe (8 caractères) obligatoires")
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.config.Username != "" {
		return fmt.Errorf("Le compte administrateur existe déjà")
	}
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return err
	}
	const iterations = 120000
	hash := pbkdf2SHA256([]byte(password), salt, iterations, 32)
	a.config = AuthConfig{
		Username: username, Salt: base64.RawStdEncoding.EncodeToString(salt),
		Hash: base64.RawStdEncoding.EncodeToString(hash), Iterations: iterations,
	}
	a.config.Users = []AuthUser{{Username: username, Salt: a.config.Salt, Hash: a.config.Hash, Iterations: iterations, Role: "Administrateur", Active: true}}
	return a.saveLocked()
}

func (a *AuthManager) login(username, password string) (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if len(a.config.Users) == 0 && a.config.Username == "" {
		return "", fmt.Errorf("Compte administrateur non configuré")
	}
	var selected *AuthUser
	for i := range a.config.Users {
		if strings.EqualFold(strings.TrimSpace(username), a.config.Users[i].Username) {
			selected = &a.config.Users[i]
			break
		}
	}
	if selected == nil || !selected.Active {
		return "", fmt.Errorf("Identifiants incorrects ou compte inactif")
	}
	salt, err := base64.RawStdEncoding.DecodeString(selected.Salt)
	if err != nil {
		return "", fmt.Errorf("Configuration administrateur invalide")
	}
	want, err := base64.RawStdEncoding.DecodeString(selected.Hash)
	if err != nil {
		return "", fmt.Errorf("Configuration administrateur invalide")
	}
	iterations := selected.Iterations
	if iterations < 10000 {
		iterations = 120000
	}
	got := pbkdf2SHA256([]byte(password), salt, iterations, len(want))
	if subtle.ConstantTimeCompare(got, want) != 1 {
		return "", fmt.Errorf("Identifiants incorrects")
	}
	rawToken := make([]byte, 32)
	if _, err := rand.Read(rawToken); err != nil {
		return "", err
	}
	token := base64.RawURLEncoding.EncodeToString(rawToken)
	a.sessions[token] = AuthSession{Username: selected.Username, Role: selected.Role, Expires: time.Now().Add(12 * time.Hour)}
	return token, nil
}

func (a *AuthManager) username() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.config.Username
}

func (a *AuthManager) authenticated(r *http.Request) bool {
	cookie, err := r.Cookie(authCookieName)
	if err != nil || cookie.Value == "" {
		return false
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	session, ok := a.sessions[cookie.Value]
	if !ok || time.Now().After(session.Expires) {
		delete(a.sessions, cookie.Value)
		return false
	}
	session.Expires = time.Now().Add(12 * time.Hour)
	a.sessions[cookie.Value] = session
	return true
}

func (a *AuthManager) currentUsername(r *http.Request) string {
	cookie, err := r.Cookie(authCookieName)
	if err != nil {
		return a.username()
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if s, ok := a.sessions[cookie.Value]; ok {
		return s.Username
	}
	return a.config.Username
}
func (a *AuthManager) currentRole(r *http.Request) string {
	cookie, err := r.Cookie(authCookieName)
	if err != nil {
		return ""
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if s, ok := a.sessions[cookie.Value]; ok {
		return s.Role
	}
	return ""
}

func (a *AuthManager) addUser(username, password, role string) error {
	username = strings.TrimSpace(username)
	role = strings.TrimSpace(role)
	if len(username) < 3 || len(password) < 8 {
		return fmt.Errorf("Utilisateur (3 caractères) et mot de passe (8 caractères) obligatoires")
	}
	if role != "Administrateur" && role != "Secrétaire" && role != "Moniteur" {
		return fmt.Errorf("Rôle invalide")
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	for _, u := range a.config.Users {
		if strings.EqualFold(u.Username, username) {
			return fmt.Errorf("Cet utilisateur existe déjà")
		}
	}
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return err
	}
	iterations := 120000
	hash := pbkdf2SHA256([]byte(password), salt, iterations, 32)
	a.config.Users = append(a.config.Users, AuthUser{Username: username, Salt: base64.RawStdEncoding.EncodeToString(salt), Hash: base64.RawStdEncoding.EncodeToString(hash), Iterations: iterations, Role: role, Active: true})
	return a.saveLocked()
}

func (a *AuthManager) publicUsers() []AuthUser {
	a.mu.Lock()
	defer a.mu.Unlock()
	out := make([]AuthUser, len(a.config.Users))
	for i, u := range a.config.Users {
		out[i] = AuthUser{Username: u.Username, Role: u.Role, Active: u.Active}
	}
	return out
}
func (a *AuthManager) roleFor(username string) string {
	a.mu.Lock()
	defer a.mu.Unlock()
	for _, u := range a.config.Users {
		if strings.EqualFold(u.Username, username) {
			return u.Role
		}
	}
	return ""
}

func (a *AuthManager) deleteUser(username string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if strings.EqualFold(username, a.config.Username) {
		return fmt.Errorf("Le compte administrateur principal ne peut pas être supprimé")
	}
	out := a.config.Users[:0]
	for _, u := range a.config.Users {
		if !strings.EqualFold(u.Username, username) {
			out = append(out, u)
		}
	}
	a.config.Users = out
	return a.saveLocked()
}

func (a *AuthManager) logout(r *http.Request) {
	cookie, err := r.Cookie(authCookieName)
	if err != nil {
		return
	}
	a.mu.Lock()
	delete(a.sessions, cookie.Value)
	a.mu.Unlock()
}

func (a *AuthManager) saveLocked() error {
	b, _ := json.MarshalIndent(a.config, "", "  ")
	tmp := a.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0600); err != nil {
		return err
	}
	return os.Rename(tmp, a.path)
}

func (a *AuthManager) protect(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		protectedAPI := strings.HasPrefix(r.URL.Path, "/api/") &&
			!strings.HasPrefix(r.URL.Path, "/api/auth/") && !strings.HasPrefix(r.URL.Path, "/api/license/") &&
			!strings.HasPrefix(r.URL.Path, "/api/public/") && !strings.HasPrefix(r.URL.Path, "/api/branding/") && r.URL.Path != "/api/health"
		if protectedAPI && !a.authenticated(r) {
			errJSON(w, http.StatusUnauthorized, "Session administrateur requise")
			return
		}
		if protectedAPI {
			role := a.currentRole(r)
			adminOnly := strings.HasPrefix(r.URL.Path, "/api/settings/") || r.URL.Path == "/api/restore" || r.URL.Path == "/api/driving/config" || r.URL.Path == "/api/driving/users"
			if adminOnly && role != "Administrateur" {
				errJSON(w, http.StatusForbidden, "Action réservée à l’administrateur")
				return
			}
			if role == "Moniteur" && r.Method != "GET" {
				allowed := r.URL.Path == "/api/driving/lesson" || r.URL.Path == "/api/driving/exam" || r.URL.Path == "/api/driving/invoice" || r.URL.Path == "/api/driving/payment"
				if !allowed {
					errJSON(w, http.StatusForbidden, "Action non autorisée pour le rôle Moniteur")
					return
				}
			}
		}
		next.ServeHTTP(w, r)
	})
}

func pbkdf2SHA256(password, salt []byte, iterations, keyLen int) []byte {
	const hashLen = 32
	blocks := (keyLen + hashLen - 1) / hashLen
	out := make([]byte, 0, blocks*hashLen)
	for block := 1; block <= blocks; block++ {
		counter := make([]byte, 4)
		binary.BigEndian.PutUint32(counter, uint32(block))
		mac := hmac.New(sha256.New, password)
		_, _ = mac.Write(salt)
		_, _ = mac.Write(counter)
		u := mac.Sum(nil)
		t := append([]byte(nil), u...)
		for i := 1; i < iterations; i++ {
			mac = hmac.New(sha256.New, password)
			_, _ = mac.Write(u)
			u = mac.Sum(nil)
			for j := range t {
				t[j] ^= u[j]
			}
		}
		out = append(out, t...)
	}
	return out[:keyLen]
}

func appDataDir() string {
	base := os.Getenv("LOCALAPPDATA")
	if base == "" {
		if cfg, err := os.UserConfigDir(); err == nil {
			base = cfg
		}
	}
	if base == "" {
		base = "."
	}
	dir := filepath.Join(base, "AutoEcolePilotProV1")
	_ = os.MkdirAll(dir, 0755)
	return dir
}

func copyLegacyFile(source, target string, perm os.FileMode) {
	if _, err := os.Stat(target); err == nil || !os.IsNotExist(err) {
		return
	}
	b, err := os.ReadFile(source)
	if err != nil {
		return
	}
	_ = os.WriteFile(target, b, perm)
}

func dataPath() string {
	return filepath.Join(appDataDir(), "data.json")
}

func customLogoPath() string {
	return filepath.Join(appDataDir(), "business-logo.img")
}

func currentStoreLogo() []byte {
	if logo, err := os.ReadFile(customLogoPath()); err == nil && len(logo) > 0 {
		return logo
	}
	return storeLogoPNG
}

func newStore() *Store {
	s := &Store{path: dataPath()}
	if b, err := os.ReadFile(s.path); err == nil {
		if json.Unmarshal(b, &s.data) == nil && strings.TrimSpace(s.data.Settings.Business.Name) != "" {
			s.ensureDefaults()
			_ = s.saveLocked()
			return s
		}
	}
	s.data = demoData()
	s.ensureDefaults()
	_ = s.saveLocked()
	return s
}

func defaultBusinessProfile() BusinessProfile {
	return BusinessProfile{
		Name: storeDisplayName, WhatsApp: storeWhatsApp, Email: storeEmail, TaxID: storeTaxID,
		InvoicePrimary: "#101d2c", InvoiceAccent: "#ff6b35",
	}
}

func defaultDigitalServices() []DigitalService {
	return []DigitalService{
		{1, "Inscription scolaire", "https://inscription.education.tn/", "Inscription en ligne des élèves.", "Éducation", "ED"},
		{2, "Inscription universitaire", "https://www.inscription.tn/", "Inscription et paiement des frais étudiants.", "Université", "UN"},
		{3, "Sajalni", "https://sajalni.tn/", "Enregistrement et vérification des téléphones.", "Téléphonie", "SJ"},
		{4, "Bulletin N°3", "https://b3.interieur.gov.tn/fr/demande", "Demande en ligne du bulletin N°3.", "Administration", "B3"},
		{5, "Visite technique", "https://www.attt.com.tn/DEV_WEB/prendreunrendezvous.php?code_menu=73", "Rendez-vous ATTT pour le contrôle technique.", "Transport", "VT"},
		{6, "Facture Ooredoo", "https://www.ooredoo.tn/Personal/fr/", "Consulter et payer une facture Ooredoo.", "Paiement", "OO"},
		{7, "Facture Tunisie Telecom", "https://mytt.tunisietelecom.tn/anonymous/paiement-facture", "Payer une facture Tunisie Telecom.", "Paiement", "TT"},
		{8, "Facture Orange", "https://www.orange.tn/paiement-de-factures", "Rechercher et payer une facture Orange.", "Paiement", "OR"},
	}
}

func demoData() StoreData {
	return StoreData{
		Products: []Product{
			{ID: 1, Name: "Lait demi-écrémé 1L", Barcode: "619000100001", Category: "Produits laitiers", Kind: "stock", PurchasePrice: 1.35, SalePrice: 1.65, Stock: 48, MinStock: 12, Pinned: true, Unit: "pièce"},
			{ID: 2, Name: "Eau minérale 1.5L", Barcode: "619000100002", Category: "Boissons", Kind: "stock", PurchasePrice: 0.65, SalePrice: 0.85, Stock: 96, MinStock: 24, Pinned: true, Unit: "bouteille"},
			{ID: 3, Name: "Pain baguette", Barcode: "619000100003", Category: "Boulangerie", Kind: "stock", PurchasePrice: 0.20, SalePrice: 0.25, Stock: 35, MinStock: 10, Pinned: true, Unit: "pièce"},
			{ID: 4, Name: "Huile végétale 1L", Barcode: "619000100004", Category: "Épicerie", Kind: "stock", PurchasePrice: 4.20, SalePrice: 4.75, Stock: 24, MinStock: 6, Unit: "bouteille"},
			{ID: 5, Name: "Pâtes 500g", Barcode: "619000100005", Category: "Épicerie", Kind: "stock", PurchasePrice: 0.85, SalePrice: 1.15, Stock: 40, MinStock: 10, Unit: "paquet"},
			{ID: 6, Name: "Biscuit chocolat", Barcode: "619000100006", Category: "Biscuits & snacks", Kind: "stock", PurchasePrice: 0.70, SalePrice: 1.00, Stock: 30, MinStock: 8, Unit: "paquet"},
			{ID: 7, Name: "Soda Cola 1L", Barcode: "619000100007", Category: "Boissons", Kind: "stock", PurchasePrice: 2.10, SalePrice: 2.50, Stock: 36, MinStock: 8, Unit: "bouteille"},
			{ID: 8, Name: "Thon 160g", Barcode: "619000100008", Category: "Conserves", Kind: "stock", PurchasePrice: 3.20, SalePrice: 3.75, Stock: 18, MinStock: 5, Unit: "boîte"},
		},
		Clients: []Client{}, Sales: []Sale{},
		SIMs: []SIM{}, Transfers: []Transfer{}, DigitalServices: []DigitalService{},
		Suppliers: []Supplier{}, Purchases: []Purchase{}, StockMovements: []StockMovement{}, CashMovements: []CashMovement{},
		Settings: Settings{
			Theme: "#ff6b35", Business: defaultBusinessProfile(),
		},
	}
}

func (s *Store) ensureDefaults() {
	s.ensureDrivingDefaults()
	s.ensureCarts()
	defaults := defaultBusinessProfile()
	profile := &s.data.Settings.Business
	if strings.TrimSpace(profile.Name) == "" {
		logoVersion := profile.LogoVersion
		*profile = defaults
		profile.LogoVersion = logoVersion
	}
	if !validThemeColor(profile.InvoicePrimary) {
		profile.InvoicePrimary = defaults.InvoicePrimary
	}
	if !validThemeColor(profile.InvoiceAccent) {
		profile.InvoiceAccent = defaults.InvoiceAccent
	}
	if !validThemeColor(s.data.Settings.Theme) {
		s.data.Settings.Theme = "#ff6b35"
	}
	for i := range s.data.Products {
		if strings.TrimSpace(s.data.Products[i].Unit) == "" {
			s.data.Products[i].Unit = "pièce"
		}
	}
}

func (s *Store) ensureCarts() {
	m := map[int]PendingCart{}
	for _, c := range s.data.Carts {
		m[c.Slot] = c
	}
	s.data.Carts = nil
	for i := 1; i <= 6; i++ {
		if c, ok := m[i]; ok {
			s.data.Carts = append(s.data.Carts, c)
		} else {
			s.data.Carts = append(s.data.Carts, PendingCart{Slot: i, Items: []CartItem{}})
		}
	}
}

func (s *Store) saveLocked() error {
	b, marshalErr := json.MarshalIndent(s.data, "", "  ")
	if marshalErr != nil {
		return marshalErr
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

func (s *Store) snapshot() StoreData {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, _ := json.Marshal(s.data)
	var out StoreData
	_ = json.Unmarshal(b, &out)
	refreshInvoiceStatuses(&out)
	return out
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func readJSON(r *http.Request, v any) error {
	return readJSONLimit(r, v, 1<<20)
}

func readJSONLimit(r *http.Request, v any, limit int64) error {
	defer r.Body.Close()
	return json.NewDecoder(io.LimitReader(r.Body, limit)).Decode(v)
}
func errJSON(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]any{"ok": false, "error": msg})
}
func okJSON(w http.ResponseWriter, v any) { writeJSON(w, 200, map[string]any{"ok": true, "data": v}) }

func main() {
	store := newStore()
	auth := newAuthManager()
	license, err := newLicenseManager()
	if err != nil {
		showDesktopError("Impossible de préparer la licence", err.Error())
		return
	}
	mux := http.NewServeMux()
	for _, asset := range []string{"coach.css", "coach.js"} {
		name := asset
		mux.HandleFunc("/"+name, func(w http.ResponseWriter, r *http.Request) {
			b, err := webFS.ReadFile("web/" + name)
			if err != nil {
				http.NotFound(w, r)
				return
			}
			contentType := "text/css; charset=utf-8"
			if strings.HasSuffix(name, ".js") {
				contentType = "text/javascript; charset=utf-8"
			}
			w.Header().Set("Content-Type", contentType)
			w.Header().Set("Cache-Control", "no-cache")
			_, _ = w.Write(b)
		})
	}
	mux.HandleFunc("/icons/", func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(r.URL.Path, "/icons/")
		if name == "" || strings.Contains(name, "..") || strings.ContainsAny(name, `/\\`) || !strings.HasSuffix(strings.ToLower(name), ".png") {
			http.NotFound(w, r)
			return
		}
		b, err := webFS.ReadFile("web/icons/" + name)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write(b)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		b, err := webFS.ReadFile("web/index.html")
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(b)
	})
	mux.HandleFunc("/api/public/branding", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			errJSON(w, 405, "Méthode non autorisée")
			return
		}
		data := store.snapshot()
		okJSON(w, map[string]any{
			"business": data.Settings.Business, "theme": data.Settings.Theme,
			"logo_url": fmt.Sprintf("/api/branding/logo?v=%d", data.Settings.Business.LogoVersion),
		})
	})
	mux.HandleFunc("/api/branding/logo", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			http.Error(w, "Méthode non autorisée", 405)
			return
		}
		logo := currentStoreLogo()
		contentType := http.DetectContentType(logo)
		if contentType != "image/png" && contentType != "image/jpeg" {
			contentType = "image/png"
		}
		w.Header().Set("Content-Type", contentType)
		w.Header().Set("Cache-Control", "no-cache")
		_, _ = w.Write(logo)
	})
	mux.HandleFunc("/api/auth/status", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			errJSON(w, 405, "Méthode non autorisée")
			return
		}
		isAuthenticated := auth.authenticated(r)
		current := auth.username()
		if isAuthenticated {
			current = auth.currentUsername(r)
		}
		role := ""
		if isAuthenticated {
			role = auth.currentRole(r)
		}
		okJSON(w, map[string]any{"configured": auth.configured(), "authenticated": isAuthenticated, "username": current, "role": role})
	})
	mux.HandleFunc("/api/license/status", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			errJSON(w, 405, "Méthode non autorisée")
			return
		}
		okJSON(w, license.status())
	})
	mux.HandleFunc("/api/license/activate", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			errJSON(w, 405, "Méthode non autorisée")
			return
		}
		var in struct {
			Key string `json:"key"`
		}
		if readJSON(r, &in) != nil {
			errJSON(w, 400, "Clé produit invalide")
			return
		}
		status, err := license.activate(in.Key)
		if err != nil {
			errJSON(w, 400, err.Error())
			return
		}
		okJSON(w, status)
	})
	mux.HandleFunc("/api/auth/setup", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			errJSON(w, 405, "Méthode non autorisée")
			return
		}
		var in struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if readJSON(r, &in) != nil {
			errJSON(w, 400, "Données invalides")
			return
		}
		if err := auth.setup(in.Username, in.Password); err != nil {
			errJSON(w, 400, err.Error())
			return
		}
		token, err := auth.login(in.Username, in.Password)
		if err != nil {
			errJSON(w, 500, "Compte créé, mais connexion impossible")
			return
		}
		setAuthCookie(w, token)
		okJSON(w, map[string]string{"username": strings.TrimSpace(in.Username), "role": "Administrateur"})
	})
	mux.HandleFunc("/api/auth/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			errJSON(w, 405, "Méthode non autorisée")
			return
		}
		var in struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if readJSON(r, &in) != nil {
			errJSON(w, 400, "Données invalides")
			return
		}
		token, err := auth.login(in.Username, in.Password)
		if err != nil {
			errJSON(w, http.StatusUnauthorized, err.Error())
			return
		}
		setAuthCookie(w, token)
		okJSON(w, map[string]string{"username": strings.TrimSpace(in.Username), "role": auth.roleFor(in.Username)})
	})
	mux.HandleFunc("/api/auth/logout", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			errJSON(w, 405, "Méthode non autorisée")
			return
		}
		auth.logout(r)
		http.SetCookie(w, &http.Cookie{Name: authCookieName, Value: "", Path: "/", MaxAge: -1, HttpOnly: true, SameSite: http.SameSiteStrictMode})
		okJSON(w, true)
	})
	mux.HandleFunc("/api/data", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			errJSON(w, 405, "Méthode non autorisée")
			return
		}
		okJSON(w, store.snapshot())
	})
	registerDrivingRoutes(mux, store, auth)
	mux.HandleFunc("/api/cart", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			errJSON(w, 405, "Méthode non autorisée")
			return
		}
		var in PendingCart
		if readJSON(r, &in) != nil || in.Slot < 1 || in.Slot > 6 {
			errJSON(w, 400, "Panier invalide")
			return
		}
		store.mu.Lock()
		defer store.mu.Unlock()
		for i := range store.data.Carts {
			if store.data.Carts[i].Slot == in.Slot {
				store.data.Carts[i].Items = in.Items
			}
		}
		_ = store.saveLocked()
		okJSON(w, in)
	})
	mux.HandleFunc("/api/product", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "DELETE" {
			id, _ := strconv.Atoi(r.URL.Query().Get("id"))
			if id <= 0 {
				errJSON(w, 400, "Produit invalide")
				return
			}
			store.mu.Lock()
			defer store.mu.Unlock()
			for i, product := range store.data.Products {
				if product.ID == id {
					store.data.Products = append(store.data.Products[:i], store.data.Products[i+1:]...)
					for c := range store.data.Carts {
						filtered := store.data.Carts[c].Items[:0]
						for _, item := range store.data.Carts[c].Items {
							if item.ProductID != id {
								filtered = append(filtered, item)
							}
						}
						store.data.Carts[c].Items = filtered
					}
					_ = store.saveLocked()
					okJSON(w, product)
					return
				}
			}
			errJSON(w, 404, "Produit introuvable")
			return
		}
		if r.Method != "POST" {
			errJSON(w, 405, "Méthode non autorisée")
			return
		}
		var p Product
		if readJSON(r, &p) != nil {
			errJSON(w, 400, "Données invalides")
			return
		}
		if err := normalizeProduct(&p); err != nil {
			errJSON(w, 400, err.Error())
			return
		}
		store.mu.Lock()
		defer store.mu.Unlock()
		if p.SupplierID > 0 {
			foundSupplier := false
			for _, supplier := range store.data.Suppliers {
				if supplier.ID == p.SupplierID {
					foundSupplier = true
					break
				}
			}
			if !foundSupplier {
				errJSON(w, 400, "Fournisseur introuvable")
				return
			}
		}
		for _, x := range store.data.Products {
			if x.Barcode == p.Barcode && x.ID != p.ID {
				errJSON(w, 409, "Ce code-barres existe déjà")
				return
			}
		}
		if p.ID == 0 {
			max := 0
			for _, x := range store.data.Products {
				if x.ID > max {
					max = x.ID
				}
			}
			p.ID = max + 1
			store.data.Products = append(store.data.Products, p)
			if p.Stock > 0 {
				store.addStockMovementLocked(p, "stock_initial", p.Stock, 0, p.Stock, "Création produit", auth.username())
			}
		} else {
			found := false
			for i, x := range store.data.Products {
				if x.ID == p.ID {
					found = true
					if x.SalePrice != p.SalePrice {
						store.data.PriceHistory = append(store.data.PriceHistory, PriceHistory{time.Now().Format(time.RFC3339), "produit", p.Name, x.SalePrice, p.SalePrice})
					}
					if x.Stock != p.Stock {
						store.addStockMovementLocked(p, "ajustement", p.Stock-x.Stock, x.Stock, p.Stock, "Fiche produit", auth.username())
					}
					store.data.Products[i] = p
					break
				}
			}
			if !found {
				errJSON(w, 404, "Produit introuvable")
				return
			}
		}
		_ = store.saveLocked()
		okJSON(w, p)
	})
	mux.HandleFunc("/api/stock", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			errJSON(w, 405, "Méthode non autorisée")
			return
		}
		var in struct {
			ProductID int    `json:"product_id"`
			Qty       int    `json:"qty"`
			Reason    string `json:"reason"`
		}
		if readJSON(r, &in) != nil || in.Qty == 0 {
			errJSON(w, 400, "Quantité invalide")
			return
		}
		store.mu.Lock()
		defer store.mu.Unlock()
		for i := range store.data.Products {
			if store.data.Products[i].ID == in.ProductID {
				if store.data.Products[i].Kind == "service" {
					errJSON(w, 400, "Un service n'a pas de stock")
					return
				}
				if store.data.Products[i].Stock+in.Qty < 0 {
					errJSON(w, 400, "Stock insuffisant")
					return
				}
				before := store.data.Products[i].Stock
				store.data.Products[i].Stock += in.Qty
				movementType := "entrée"
				if in.Qty < 0 {
					movementType = "sortie"
				}
				reason := strings.TrimSpace(in.Reason)
				if reason == "" {
					reason = "Ajustement manuel"
				}
				store.addStockMovementLocked(store.data.Products[i], movementType, in.Qty, before, store.data.Products[i].Stock, reason, auth.username())
				_ = store.saveLocked()
				okJSON(w, store.data.Products[i])
				return
			}
		}
		errJSON(w, 404, "Produit introuvable")
	})
	mux.HandleFunc("/api/sale", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			errJSON(w, 405, "Méthode non autorisée")
			return
		}
		var in struct {
			Slot        int        `json:"slot"`
			Items       []CartItem `json:"items"`
			Payment     string     `json:"payment"`
			Received    float64    `json:"received"`
			Discount    float64    `json:"discount"`
			ClientName  string     `json:"client_name"`
			ClientPhone string     `json:"client_phone"`
		}
		if readJSON(r, &in) != nil || len(in.Items) == 0 {
			errJSON(w, 400, "Panier vide ou invalide")
			return
		}
		if in.Payment != "cash" && in.Payment != "card" && in.Payment != "credit" {
			errJSON(w, 400, "Paiement invalide")
			return
		}
		in.ClientName = strings.TrimSpace(in.ClientName)
		in.ClientPhone = digits(in.ClientPhone)
		if in.Payment == "credit" && (in.ClientName == "" || len(in.ClientPhone) < 8) {
			errJSON(w, 400, "Nom et téléphone client obligatoires pour le crédit")
			return
		}
		store.mu.Lock()
		defer store.mu.Unlock()
		items := []SaleItem{}
		total := 0.0
		for _, ci := range in.Items {
			if ci.Qty <= 0 {
				continue
			}
			if ci.ProductID == 0 {
				name := strings.TrimSpace(ci.CustomName)
				if name == "" || ci.UnitPrice <= 0 || ci.UnitPrice > 100000 {
					errJSON(w, 400, "Autre frais invalide")
					return
				}
				line := float64(ci.Qty) * ci.UnitPrice
				items = append(items, SaleItem{ProductID: 0, Name: name, Qty: ci.Qty, UnitPrice: ci.UnitPrice, Total: line})
				total += line
				continue
			}
			found := -1
			for i, p := range store.data.Products {
				if p.ID == ci.ProductID {
					found = i
					break
				}
			}
			if found < 0 {
				errJSON(w, 404, "Produit introuvable")
				return
			}
			p := store.data.Products[found]
			if p.Kind == "stock" && p.Stock < ci.Qty {
				errJSON(w, 400, "Stock insuffisant pour "+p.Name)
				return
			}
			unitPrice := effectiveProductPrice(p, time.Now())
			line := float64(ci.Qty) * unitPrice
			items = append(items, SaleItem{ProductID: p.ID, Name: p.Name, Qty: ci.Qty, UnitPrice: unitPrice, Total: line, UnitCost: p.PurchasePrice})
			total += line
		}
		if len(items) == 0 {
			errJSON(w, 400, "Panier vide")
			return
		}
		subtotal := total
		if in.Discount < 0 || in.Discount > subtotal {
			errJSON(w, 400, "Remise invalide")
			return
		}
		total -= in.Discount
		if in.Payment == "cash" && in.Received < total {
			errJSON(w, 400, "Montant reçu insuffisant")
			return
		}
		// decrement stock only after all validations
		for _, it := range items {
			for i := range store.data.Products {
				if store.data.Products[i].ID == it.ProductID && store.data.Products[i].Kind == "stock" {
					before := store.data.Products[i].Stock
					store.data.Products[i].Stock -= it.Qty
					store.addStockMovementLocked(store.data.Products[i], "vente", -it.Qty, before, store.data.Products[i].Stock, fmt.Sprintf("Vente #%d", len(store.data.Sales)+1), auth.username())
				}
			}
		}
		sale := Sale{ID: len(store.data.Sales) + 1, Time: time.Now().Format(time.RFC3339), Items: items, Subtotal: subtotal, Discount: in.Discount, Total: total, Payment: in.Payment, Received: in.Received, Seller: auth.username(), Status: "validée"}
		if in.Payment == "cash" {
			sale.Change = in.Received - total
		} else if in.Payment == "credit" {
			cid := 0
			for i := range store.data.Clients {
				if store.data.Clients[i].Phone == in.ClientPhone {
					cid = store.data.Clients[i].ID
					store.data.Clients[i].Name = in.ClientName
					store.data.Clients[i].Debt += total
					sale.ClientID = cid
					break
				}
			}
			if cid == 0 {
				cid = len(store.data.Clients) + 1
				store.data.Clients = append(store.data.Clients, Client{cid, in.ClientName, in.ClientPhone, total})
				sale.ClientID = cid
			}
			sale.Client = in.ClientName
			sale.Phone = in.ClientPhone
		}
		store.data.Sales = append(store.data.Sales, sale)
		for i := range store.data.Carts {
			if store.data.Carts[i].Slot == in.Slot {
				store.data.Carts[i].Items = []CartItem{}
			}
		}
		_ = store.saveLocked()
		okJSON(w, sale)
	})
	mux.HandleFunc("/api/client/payment", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			errJSON(w, 405, "Méthode non autorisée")
			return
		}
		var in struct {
			ClientID int     `json:"client_id"`
			Amount   float64 `json:"amount"`
		}
		if readJSON(r, &in) != nil || in.Amount <= 0 {
			errJSON(w, 400, "Montant invalide")
			return
		}
		store.mu.Lock()
		defer store.mu.Unlock()
		for i := range store.data.Clients {
			if store.data.Clients[i].ID == in.ClientID {
				if in.Amount > store.data.Clients[i].Debt+0.0001 {
					errJSON(w, 400, "Le paiement dépasse la dette")
					return
				}
				store.data.Clients[i].Debt -= in.Amount
				store.addCashMovementLocked("entrée", in.Amount, "Règlement crédit: "+store.data.Clients[i].Name, auth.username())
				_ = store.saveLocked()
				okJSON(w, store.data.Clients[i])
				return
			}
		}
		errJSON(w, 404, "Client introuvable")
	})
	mux.HandleFunc("/api/supplier", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "DELETE" {
			id, _ := strconv.Atoi(r.URL.Query().Get("id"))
			store.mu.Lock()
			defer store.mu.Unlock()
			for i, supplier := range store.data.Suppliers {
				if supplier.ID == id {
					if supplier.Balance > 0.0001 {
						errJSON(w, 400, "Réglez d'abord le solde fournisseur")
						return
					}
					store.data.Suppliers = append(store.data.Suppliers[:i], store.data.Suppliers[i+1:]...)
					_ = store.saveLocked()
					okJSON(w, supplier)
					return
				}
			}
			errJSON(w, 404, "Fournisseur introuvable")
			return
		}
		if r.Method != "POST" {
			errJSON(w, 405, "Méthode non autorisée")
			return
		}
		var supplier Supplier
		if readJSON(r, &supplier) != nil {
			errJSON(w, 400, "Données invalides")
			return
		}
		supplier.Name = strings.TrimSpace(supplier.Name)
		rawPhone := strings.TrimSpace(supplier.Phone)
		supplier.Phone = digits(rawPhone)
		supplier.Address = strings.TrimSpace(supplier.Address)
		if len(supplier.Name) < 2 || len(supplier.Name) > 120 || len(supplier.Address) > 220 || (rawPhone != "" && len(supplier.Phone) < 8) {
			errJSON(w, 400, "Informations fournisseur invalides")
			return
		}
		store.mu.Lock()
		defer store.mu.Unlock()
		if supplier.ID == 0 {
			for _, existing := range store.data.Suppliers {
				if existing.ID >= supplier.ID {
					supplier.ID = existing.ID + 1
				}
			}
			if supplier.ID == 0 {
				supplier.ID = 1
			}
			store.data.Suppliers = append(store.data.Suppliers, supplier)
		} else {
			found := false
			for i := range store.data.Suppliers {
				if store.data.Suppliers[i].ID == supplier.ID {
					supplier.Balance = store.data.Suppliers[i].Balance
					store.data.Suppliers[i] = supplier
					found = true
					break
				}
			}
			if !found {
				errJSON(w, 404, "Fournisseur introuvable")
				return
			}
		}
		_ = store.saveLocked()
		okJSON(w, supplier)
	})
	mux.HandleFunc("/api/supplier/payment", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			errJSON(w, 405, "Méthode non autorisée")
			return
		}
		var in struct {
			SupplierID int     `json:"supplier_id"`
			Amount     float64 `json:"amount"`
		}
		if readJSON(r, &in) != nil || in.Amount <= 0 {
			errJSON(w, 400, "Montant invalide")
			return
		}
		store.mu.Lock()
		defer store.mu.Unlock()
		for i := range store.data.Suppliers {
			if store.data.Suppliers[i].ID == in.SupplierID {
				if in.Amount > store.data.Suppliers[i].Balance+0.0001 {
					errJSON(w, 400, "Le paiement dépasse le solde")
					return
				}
				store.data.Suppliers[i].Balance -= in.Amount
				store.addCashMovementLocked("sortie", in.Amount, "Paiement fournisseur: "+store.data.Suppliers[i].Name, auth.username())
				_ = store.saveLocked()
				okJSON(w, store.data.Suppliers[i])
				return
			}
		}
		errJSON(w, 404, "Fournisseur introuvable")
	})
	mux.HandleFunc("/api/purchase", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			errJSON(w, 405, "Méthode non autorisée")
			return
		}
		var in struct {
			SupplierID int            `json:"supplier_id"`
			Reference  string         `json:"reference"`
			Items      []PurchaseItem `json:"items"`
			Paid       float64        `json:"paid"`
		}
		if readJSONLimit(r, &in, 4<<20) != nil || in.SupplierID <= 0 || len(in.Items) == 0 || len(in.Items) > 500 || in.Paid < 0 {
			errJSON(w, 400, "Bon d'achat invalide")
			return
		}
		in.Reference = strings.TrimSpace(in.Reference)
		store.mu.Lock()
		defer store.mu.Unlock()
		supplierIndex := -1
		for i := range store.data.Suppliers {
			if store.data.Suppliers[i].ID == in.SupplierID {
				supplierIndex = i
				break
			}
		}
		if supplierIndex < 0 {
			errJSON(w, 404, "Fournisseur introuvable")
			return
		}
		purchaseItems := make([]PurchaseItem, 0, len(in.Items))
		total := 0.0
		for _, item := range in.Items {
			if item.ProductID <= 0 || item.Qty <= 0 || item.UnitCost < 0 || item.UnitCost > 1000000 {
				errJSON(w, 400, "Une ligne d'achat est invalide")
				return
			}
			productIndex := -1
			for i := range store.data.Products {
				if store.data.Products[i].ID == item.ProductID {
					productIndex = i
					break
				}
			}
			if productIndex < 0 {
				errJSON(w, 404, "Produit d'achat introuvable")
				return
			}
			product := &store.data.Products[productIndex]
			line := float64(item.Qty) * item.UnitCost
			purchaseItems = append(purchaseItems, PurchaseItem{ProductID: product.ID, Name: product.Name, Qty: item.Qty, UnitCost: item.UnitCost, Total: line})
			total += line
		}
		if in.Paid > total+0.0001 {
			errJSON(w, 400, "Le paiement dépasse le total du bon")
			return
		}
		purchaseID := len(store.data.Purchases) + 1
		for _, item := range purchaseItems {
			for i := range store.data.Products {
				if store.data.Products[i].ID == item.ProductID {
					before := store.data.Products[i].Stock
					store.data.Products[i].Stock += item.Qty
					store.data.Products[i].PurchasePrice = item.UnitCost
					store.data.Products[i].SupplierID = in.SupplierID
					store.addStockMovementLocked(store.data.Products[i], "achat", item.Qty, before, store.data.Products[i].Stock, fmt.Sprintf("Achat #%d", purchaseID), auth.username())
				}
			}
		}
		balance := total - in.Paid
		store.data.Suppliers[supplierIndex].Balance += balance
		purchase := Purchase{ID: purchaseID, Time: time.Now().Format(time.RFC3339), SupplierID: in.SupplierID, SupplierName: store.data.Suppliers[supplierIndex].Name, Reference: in.Reference, Items: purchaseItems, Total: total, Paid: in.Paid, Balance: balance, CreatedBy: auth.username()}
		store.data.Purchases = append(store.data.Purchases, purchase)
		if in.Paid > 0 {
			store.addCashMovementLocked("sortie", in.Paid, fmt.Sprintf("Achat #%d - %s", purchaseID, purchase.SupplierName), auth.username())
		}
		_ = store.saveLocked()
		okJSON(w, purchase)
	})
	mux.HandleFunc("/api/cash-movement", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			errJSON(w, 405, "Méthode non autorisée")
			return
		}
		var in CashMovement
		if readJSON(r, &in) != nil || (in.Type != "entrée" && in.Type != "sortie") || in.Amount <= 0 || in.Amount > 1000000 || strings.TrimSpace(in.Reason) == "" {
			errJSON(w, 400, "Mouvement de caisse invalide")
			return
		}
		store.mu.Lock()
		defer store.mu.Unlock()
		movement := store.addCashMovementLocked(in.Type, in.Amount, strings.TrimSpace(in.Reason), auth.username())
		_ = store.saveLocked()
		okJSON(w, movement)
	})
	mux.HandleFunc("/api/sale/cancel", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			errJSON(w, 405, "Méthode non autorisée")
			return
		}
		var in struct {
			ID int `json:"id"`
		}
		if readJSON(r, &in) != nil || in.ID <= 0 {
			errJSON(w, 400, "Vente invalide")
			return
		}
		store.mu.Lock()
		defer store.mu.Unlock()
		for saleIndex := range store.data.Sales {
			sale := &store.data.Sales[saleIndex]
			if sale.ID != in.ID {
				continue
			}
			if sale.Status == "annulée" {
				errJSON(w, 400, "Cette vente est déjà annulée")
				return
			}
			for _, item := range sale.Items {
				for i := range store.data.Products {
					if store.data.Products[i].ID == item.ProductID {
						before := store.data.Products[i].Stock
						store.data.Products[i].Stock += item.Qty
						store.addStockMovementLocked(store.data.Products[i], "retour", item.Qty, before, store.data.Products[i].Stock, fmt.Sprintf("Annulation vente #%d", sale.ID), auth.username())
					}
				}
			}
			if sale.Payment == "credit" && sale.ClientID > 0 {
				for i := range store.data.Clients {
					if store.data.Clients[i].ID == sale.ClientID {
						store.data.Clients[i].Debt = maxFloat(0, store.data.Clients[i].Debt-sale.Total)
					}
				}
			}
			sale.Status = "annulée"
			sale.CancelledAt = time.Now().Format(time.RFC3339)
			sale.CancelledBy = auth.username()
			_ = store.saveLocked()
			okJSON(w, sale)
			return
		}
		errJSON(w, 404, "Vente introuvable")
	})
	mux.HandleFunc("/api/sim", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "DELETE" {
			id, _ := strconv.Atoi(r.URL.Query().Get("id"))
			if id <= 0 {
				errJSON(w, 400, "Identifiant SIM invalide")
				return
			}
			store.mu.Lock()
			defer store.mu.Unlock()
			for i := range store.data.SIMs {
				if store.data.SIMs[i].ID == id {
					removed := store.data.SIMs[i]
					store.data.SIMs = append(store.data.SIMs[:i], store.data.SIMs[i+1:]...)
					if err := store.saveLocked(); err != nil {
						errJSON(w, 500, "Impossible d'enregistrer la suppression")
						return
					}
					okJSON(w, removed)
					return
				}
			}
			errJSON(w, 404, "SIM introuvable")
			return
		}
		if r.Method != "POST" {
			errJSON(w, 405, "Méthode non autorisée")
			return
		}
		var in SIM
		if readJSON(r, &in) != nil {
			errJSON(w, 400, "Données invalides")
			return
		}
		in.Label = strings.TrimSpace(in.Label)
		in.Number = digits(in.Number)
		if in.Label == "" || len(in.Number) < 8 {
			errJSON(w, 400, "Libellé et numéro SIM obligatoires")
			return
		}
		if in.QuotaMB <= 0 {
			in.QuotaMB = 1000
		}
		store.mu.Lock()
		defer store.mu.Unlock()
		if in.ID == 0 {
			max := 0
			for _, x := range store.data.SIMs {
				if x.ID > max {
					max = x.ID
				}
			}
			in.ID = max + 1
			in.Active = true
			store.data.SIMs = append(store.data.SIMs, in)
		} else {
			found := false
			for i := range store.data.SIMs {
				if store.data.SIMs[i].ID == in.ID {
					store.data.SIMs[i] = in
					found = true
				}
			}
			if !found {
				errJSON(w, 404, "SIM introuvable")
				return
			}
		}
		_ = store.saveLocked()
		okJSON(w, in)
	})
	mux.HandleFunc("/api/settings/connection", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			errJSON(w, 405, "Méthode non autorisée")
			return
		}
		var in Settings
		if readJSON(r, &in) != nil || in.Conn500Price < 0 || in.Conn1000Price < 0 || in.Conn500Cost < 0 || in.Conn1000Cost < 0 {
			errJSON(w, 400, "Tarifs invalides")
			return
		}
		store.mu.Lock()
		defer store.mu.Unlock()
		old := store.data.Settings
		if old.Conn500Price != in.Conn500Price {
			store.data.PriceHistory = append(store.data.PriceHistory, PriceHistory{time.Now().Format(time.RFC3339), "connexion", "500 Mo", old.Conn500Price, in.Conn500Price})
		}
		if old.Conn1000Price != in.Conn1000Price {
			store.data.PriceHistory = append(store.data.PriceHistory, PriceHistory{time.Now().Format(time.RFC3339), "connexion", "1 Go", old.Conn1000Price, in.Conn1000Price})
		}
		old.Conn500Price = in.Conn500Price
		old.Conn500Cost = in.Conn500Cost
		old.Conn1000Price = in.Conn1000Price
		old.Conn1000Cost = in.Conn1000Cost
		store.data.Settings = old
		_ = store.saveLocked()
		okJSON(w, old)
	})
	mux.HandleFunc("/api/settings/theme", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			errJSON(w, 405, "Méthode non autorisée")
			return
		}
		var in struct {
			Theme string `json:"theme"`
		}
		if readJSON(r, &in) != nil {
			errJSON(w, 400, "Couleur invalide")
			return
		}
		in.Theme = strings.ToLower(strings.TrimSpace(in.Theme))
		if !validThemeColor(in.Theme) {
			errJSON(w, 400, "Couleur invalide")
			return
		}
		store.mu.Lock()
		defer store.mu.Unlock()
		store.data.Settings.Theme = in.Theme
		_ = store.saveLocked()
		okJSON(w, map[string]string{"theme": in.Theme})
	})
	mux.HandleFunc("/api/settings/business", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			errJSON(w, 405, "Méthode non autorisée")
			return
		}
		var in struct {
			Business   BusinessProfile `json:"business"`
			Theme      string          `json:"theme"`
			LogoData   string          `json:"logo_data"`
			RemoveLogo bool            `json:"remove_logo"`
		}
		if readJSONLimit(r, &in, 4<<20) != nil {
			errJSON(w, 400, "Paramètres invalides")
			return
		}
		profile, err := normalizeBusinessProfile(in.Business)
		if err != nil {
			errJSON(w, 400, err.Error())
			return
		}
		in.Theme = strings.ToLower(strings.TrimSpace(in.Theme))
		if !validThemeColor(in.Theme) {
			errJSON(w, 400, "Couleur du logiciel invalide")
			return
		}
		var logoBytes []byte
		if strings.TrimSpace(in.LogoData) != "" {
			logoBytes, err = decodeLogoDataURL(in.LogoData)
			if err != nil {
				errJSON(w, 400, err.Error())
				return
			}
		}
		oldLogo, oldLogoErr := os.ReadFile(customLogoPath())
		logoCommitted := false
		defer func() {
			if logoCommitted || (len(logoBytes) == 0 && !in.RemoveLogo) {
				return
			}
			if oldLogoErr == nil {
				_ = os.WriteFile(customLogoPath(), oldLogo, 0600)
			} else if os.IsNotExist(oldLogoErr) {
				_ = os.Remove(customLogoPath())
			}
		}()
		if len(logoBytes) > 0 {
			if err := os.WriteFile(customLogoPath(), logoBytes, 0600); err != nil {
				errJSON(w, 500, "Impossible d'enregistrer le logo")
				return
			}
			profile.LogoVersion = time.Now().UnixNano()
		} else if in.RemoveLogo {
			_ = os.Remove(customLogoPath())
			profile.LogoVersion = time.Now().UnixNano()
		} else {
			store.mu.Lock()
			profile.LogoVersion = store.data.Settings.Business.LogoVersion
			store.mu.Unlock()
		}
		store.mu.Lock()
		before := store.drivingSnapshotLocked()
		store.data.Settings.Business = profile
		store.data.Settings.Theme = in.Theme
		saved := store.commitDrivingLocked(w, before)
		settings := store.data.Settings
		store.mu.Unlock()
		if !saved {
			return
		}
		logoCommitted = true
		okJSON(w, settings)
	})
	mux.HandleFunc("/api/digital-service", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "DELETE" {
			id, _ := strconv.Atoi(r.URL.Query().Get("id"))
			if id <= 0 {
				errJSON(w, 400, "Service invalide")
				return
			}
			store.mu.Lock()
			defer store.mu.Unlock()
			for i := range store.data.DigitalServices {
				if store.data.DigitalServices[i].ID == id {
					removed := store.data.DigitalServices[i]
					store.data.DigitalServices = append(store.data.DigitalServices[:i], store.data.DigitalServices[i+1:]...)
					_ = store.saveLocked()
					okJSON(w, removed)
					return
				}
			}
			errJSON(w, 404, "Service introuvable")
			return
		}
		if r.Method != "POST" {
			errJSON(w, 405, "Méthode non autorisée")
			return
		}
		var service DigitalService
		if readJSON(r, &service) != nil {
			errJSON(w, 400, "Service invalide")
			return
		}
		if err := normalizeDigitalService(&service); err != nil {
			errJSON(w, 400, err.Error())
			return
		}
		store.mu.Lock()
		defer store.mu.Unlock()
		if service.ID == 0 {
			for _, existing := range store.data.DigitalServices {
				if existing.ID >= service.ID {
					service.ID = existing.ID + 1
				}
			}
			if service.ID == 0 {
				service.ID = 1
			}
			store.data.DigitalServices = append(store.data.DigitalServices, service)
		} else {
			found := false
			for i := range store.data.DigitalServices {
				if store.data.DigitalServices[i].ID == service.ID {
					store.data.DigitalServices[i] = service
					found = true
					break
				}
			}
			if !found {
				errJSON(w, 404, "Service introuvable")
				return
			}
		}
		_ = store.saveLocked()
		okJSON(w, service)
	})
	mux.HandleFunc("/api/connection", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			errJSON(w, 405, "Méthode non autorisée")
			return
		}
		var in struct {
			SIMID      int    `json:"sim_id"`
			ClientName string `json:"client_name"`
			DestPhone  string `json:"dest_phone"`
			PackageMB  int    `json:"package_mb"`
		}
		if readJSON(r, &in) != nil {
			errJSON(w, 400, "Données invalides")
			return
		}
		in.ClientName = strings.TrimSpace(in.ClientName)
		rawPhone := strings.TrimSpace(in.DestPhone)
		in.DestPhone = digits(rawPhone)
		if len(in.ClientName) > 120 || (rawPhone != "" && len(in.DestPhone) < 8) {
			errJSON(w, 400, "Nom ou téléphone client invalide")
			return
		}
		if in.PackageMB != 500 && in.PackageMB != 1000 {
			errJSON(w, 400, "Forfait invalide")
			return
		}
		store.mu.Lock()
		defer store.mu.Unlock()
		var sim *SIM
		for i := range store.data.SIMs {
			if store.data.SIMs[i].ID == in.SIMID {
				sim = &store.data.SIMs[i]
				break
			}
		}
		if sim == nil || !sim.Active {
			errJSON(w, 404, "SIM indisponible")
			return
		}
		today := time.Now().Format("2006-01-02")
		used := simUsageForDay(store.data.Transfers, sim.ID, today)
		if used+in.PackageMB > sim.QuotaMB {
			errJSON(w, 400, fmt.Sprintf("Quota journalier dépassé: %d Mo restants", sim.QuotaMB-used))
			return
		}
		price, cost := store.data.Settings.Conn500Price, store.data.Settings.Conn500Cost
		if in.PackageMB == 1000 {
			price, cost = store.data.Settings.Conn1000Price, store.data.Settings.Conn1000Cost
		}
		tr := Transfer{
			ID: len(store.data.Transfers) + 1, Time: time.Now().Format(time.RFC3339),
			SIMID: sim.ID, SIMLabel: sim.Label, ClientName: in.ClientName, DestPhone: in.DestPhone,
			SoldBy: auth.username(), PackageMB: in.PackageMB, Price: price, Cost: cost, Status: "Enregistré",
		}
		store.data.Transfers = append(store.data.Transfers, tr)
		_ = store.saveLocked()
		okJSON(w, tr)
	})
	mux.HandleFunc("/api/reports/sim/daily.pdf", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			errJSON(w, 405, "Méthode non autorisée")
			return
		}
		day := strings.TrimSpace(r.URL.Query().Get("date"))
		if day == "" {
			day = time.Now().Format("2006-01-02")
		}
		if _, err := time.Parse("2006-01-02", day); err != nil {
			errJSON(w, 400, "Date invalide")
			return
		}
		pdf := buildDailySIMReportPDF(store.snapshot(), day)
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=rapport-ventes-sim-%s.pdf", day))
		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Cache-Control", "no-store")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(pdf)
	})
	mux.HandleFunc("/api/reports/sales/daily.pdf", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			errJSON(w, 405, "Méthode non autorisée")
			return
		}
		day := strings.TrimSpace(r.URL.Query().Get("date"))
		if day == "" {
			day = time.Now().Format("2006-01-02")
		}
		if _, err := time.Parse("2006-01-02", day); err != nil {
			errJSON(w, 400, "Date invalide")
			return
		}
		pdf := buildDailySalesReportPDF(store.snapshot(), day)
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=cloture-caisse-%s.pdf", day))
		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write(pdf)
	})
	mux.HandleFunc("/api/reports/sale/invoice.pdf", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			errJSON(w, 405, "Méthode non autorisée")
			return
		}
		id, _ := strconv.Atoi(r.URL.Query().Get("id"))
		if id <= 0 {
			errJSON(w, 400, "Numéro de vente invalide")
			return
		}
		data := store.snapshot()
		var sale *Sale
		for i := range data.Sales {
			if data.Sales[i].ID == id {
				sale = &data.Sales[i]
				break
			}
		}
		if sale == nil {
			errJSON(w, 404, "Vente introuvable")
			return
		}
		pdf := buildSaleInvoicePDF(*sale, data.Settings.Business, currentStoreLogo())
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=facture-vente-%04d.pdf", sale.ID))
		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Cache-Control", "no-store")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(pdf)
	})
	mux.HandleFunc("/api/backup", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			errJSON(w, 405, "Méthode non autorisée")
			return
		}
		data := store.snapshot()
		bundle := BackupBundle{Format: "autoecole-pilot-pro-v1-backup", CreatedAt: time.Now().Format(time.RFC3339), Data: data}
		if logo, err := os.ReadFile(customLogoPath()); err == nil && len(logo) > 0 {
			bundle.HasCustomLogo = true
			bundle.LogoBase64 = base64.StdEncoding.EncodeToString(logo)
		}
		bundle.DocumentFiles = map[string]string{}
		for _, document := range data.DrivingDocuments {
			if document.StoredName == "" || filepath.Base(document.StoredName) != document.StoredName {
				continue
			}
			if content, err := os.ReadFile(filepath.Join(appDataDir(), "documents", document.StoredName)); err == nil && len(content) <= 5<<20 {
				bundle.DocumentFiles[document.StoredName] = base64.StdEncoding.EncodeToString(content)
			}
		}
		b, _ := json.MarshalIndent(bundle, "", "  ")
		w.Header().Set("Content-Disposition", "attachment; filename=autoecole-pilot-pro-v1-backup.json")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(b)
	})
	mux.HandleFunc("/api/restore", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			errJSON(w, 405, "Méthode non autorisée")
			return
		}
		defer r.Body.Close()
		raw, err := io.ReadAll(io.LimitReader(r.Body, 100<<20))
		if err != nil || len(raw) == 0 || len(raw) >= 100<<20 {
			errJSON(w, 400, "Fichier de sauvegarde invalide ou trop volumineux")
			return
		}
		var bundle BackupBundle
		var restored StoreData
		isBundle := json.Unmarshal(raw, &bundle) == nil && bundle.Format == "autoecole-pilot-pro-v1-backup"
		if isBundle {
			restored = bundle.Data
		} else if json.Unmarshal(raw, &restored) != nil {
			errJSON(w, 400, "Sauvegarde JSON invalide")
			return
		}
		candidate := &Store{data: restored}
		candidate.ensureDefaults()
		if err := validateRestoredData(&candidate.data); err != nil {
			errJSON(w, 400, err.Error())
			return
		}
		var restoredLogo []byte
		if isBundle && bundle.HasCustomLogo {
			restoredLogo, err = base64.StdEncoding.DecodeString(bundle.LogoBase64)
			if err != nil || len(restoredLogo) == 0 || len(restoredLogo) > 2<<20 {
				errJSON(w, 400, "Logo de sauvegarde invalide")
				return
			}
			if _, _, err := image.Decode(bytes.NewReader(restoredLogo)); err != nil {
				errJSON(w, 400, "Image de sauvegarde invalide")
				return
			}
		}
		restoredDocuments := map[string][]byte{}
		if isBundle {
			for name, encoded := range bundle.DocumentFiles {
				if name == "" || filepath.Base(name) != name {
					errJSON(w, 400, "Nom de document invalide dans la sauvegarde")
					return
				}
				content, decodeErr := base64.StdEncoding.DecodeString(encoded)
				if decodeErr != nil || len(content) == 0 || len(content) > 5<<20 {
					errJSON(w, 400, "Document invalide dans la sauvegarde")
					return
				}
				restoredDocuments[name] = content
			}
		}
		if len(restoredDocuments) > 0 {
			docDir := filepath.Join(appDataDir(), "documents")
			if err := os.MkdirAll(docDir, 0700); err != nil {
				errJSON(w, 500, "Impossible de préparer les documents restaurés")
				return
			}
			for name, content := range restoredDocuments {
				if err := os.WriteFile(filepath.Join(docDir, name), content, 0600); err != nil {
					errJSON(w, 500, "Impossible de restaurer les documents")
					return
				}
			}
		}
		store.mu.Lock()
		previous, _ := json.MarshalIndent(store.data, "", "  ")
		backupBeforeRestore := store.path + ".before-restore-" + time.Now().Format("20060102-150405") + ".json"
		_ = os.WriteFile(backupBeforeRestore, previous, 0600)
		if isBundle {
			candidate.data.Settings.Business.LogoVersion = time.Now().UnixNano()
		}
		store.data = candidate.data
		if err := store.saveLocked(); err != nil {
			store.mu.Unlock()
			errJSON(w, 500, "Impossible de restaurer les données")
			return
		}
		store.mu.Unlock()
		if isBundle {
			if bundle.HasCustomLogo {
				_ = os.WriteFile(customLogoPath(), restoredLogo, 0600)
			} else {
				_ = os.Remove(customLogoPath())
			}
		}
		okJSON(w, map[string]any{"restored": true, "business": candidate.data.Settings.Business})
	})
	mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, "AutoEcole-Pilot-Pro-v1") })

	port := strings.TrimSpace(os.Getenv("AUTOECOLE_PORT"))
	if port == "" {
		port = "19886"
	}
	address := "127.0.0.1:" + port
	url := "http://" + address + "/"
	ln, err := net.Listen("tcp", address)
	if err != nil {
		if existingAppRunning(url) {
			return
		}
		log.Fatalf("Impossible de démarrer le logiciel sur %s: %v", address, err)
	}
	fmt.Println("AutoÉcole Pilot Pro v1:", url)
	srv := &http.Server{Handler: license.protect(auth.protect(mux)), ReadHeaderTimeout: 5 * time.Second}
	if os.Getenv("NO_BROWSER") == "1" {
		if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
		return
	}
	serverDone := make(chan error, 1)
	go func() { serverDone <- srv.Serve(ln) }()
	if err := runDesktopWindow(url); err != nil {
		showDesktopError("Impossible d'ouvrir AutoÉcole Pilot Pro v1", err.Error())
	}
	_ = srv.Close()
	if err := <-serverDone; err != nil && err != http.ErrServerClosed {
		log.Print(err)
	}
}

func setAuthCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name: authCookieName, Value: token, Path: "/", MaxAge: 12 * 60 * 60,
		HttpOnly: true, SameSite: http.SameSiteStrictMode,
	})
}

func digits(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func validThemeColor(s string) bool {
	if len(s) != 7 || s[0] != '#' {
		return false
	}
	_, err := strconv.ParseUint(s[1:], 16, 24)
	return err == nil
}

func normalizeProduct(product *Product) error {
	product.Name = strings.TrimSpace(product.Name)
	product.Barcode = strings.TrimSpace(product.Barcode)
	product.Category = strings.TrimSpace(product.Category)
	product.Unit = strings.TrimSpace(product.Unit)
	product.ExpiryDate = strings.TrimSpace(product.ExpiryDate)
	product.PromoUntil = strings.TrimSpace(product.PromoUntil)
	product.Kind = "stock"
	if product.Unit == "" {
		product.Unit = "pièce"
	}
	if product.Category == "" {
		product.Category = "Autres"
	}
	if len(product.Name) < 2 || len(product.Name) > 160 || product.Barcode == "" || len(product.Barcode) > 80 || len(product.Category) > 80 || len(product.Unit) > 30 {
		return fmt.Errorf("nom, code-barres, catégorie et unité valides obligatoires")
	}
	if product.SalePrice < 0 || product.PurchasePrice < 0 || product.SalePrice > 1000000 || product.PurchasePrice > 1000000 || product.Stock < 0 || product.MinStock < 0 {
		return fmt.Errorf("prix et stock invalides")
	}
	if product.ExpiryDate != "" {
		if _, err := time.Parse("2006-01-02", product.ExpiryDate); err != nil {
			return fmt.Errorf("date de péremption invalide")
		}
	}
	if product.PromoPrice < 0 || product.PromoPrice > product.SalePrice || (product.PromoPrice > 0 && product.PromoPrice == product.SalePrice) {
		return fmt.Errorf("le prix promotionnel doit être inférieur au prix normal")
	}
	if product.PromoUntil != "" {
		if product.PromoPrice <= 0 {
			return fmt.Errorf("indiquez un prix promotionnel")
		}
		if _, err := time.Parse("2006-01-02", product.PromoUntil); err != nil {
			return fmt.Errorf("fin de promotion invalide")
		}
	}
	return nil
}

func effectiveProductPrice(product Product, at time.Time) float64 {
	if product.PromoPrice <= 0 {
		return product.SalePrice
	}
	if product.PromoUntil == "" || product.PromoUntil >= at.Format("2006-01-02") {
		return product.PromoPrice
	}
	return product.SalePrice
}

func (s *Store) addStockMovementLocked(product Product, movementType string, qty, before, after int, reference, username string) StockMovement {
	id := len(s.data.StockMovements) + 1
	movement := StockMovement{ID: id, Time: time.Now().Format(time.RFC3339), ProductID: product.ID, ProductName: product.Name, Type: movementType, Qty: qty, Before: before, After: after, Reference: reference, User: username}
	s.data.StockMovements = append(s.data.StockMovements, movement)
	return movement
}

func (s *Store) addCashMovementLocked(movementType string, amount float64, reason, username string) CashMovement {
	id := len(s.data.CashMovements) + 1
	movement := CashMovement{ID: id, Time: time.Now().Format(time.RFC3339), Type: movementType, Amount: amount, Reason: reason, User: username}
	s.data.CashMovements = append(s.data.CashMovements, movement)
	return movement
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func normalizeBusinessProfile(profile BusinessProfile) (BusinessProfile, error) {
	profile.Name = strings.TrimSpace(profile.Name)
	profile.Address = strings.TrimSpace(profile.Address)
	profile.WhatsApp = strings.TrimSpace(profile.WhatsApp)
	profile.Email = strings.TrimSpace(profile.Email)
	profile.TaxID = strings.TrimSpace(profile.TaxID)
	profile.InvoicePrimary = strings.ToLower(strings.TrimSpace(profile.InvoicePrimary))
	profile.InvoiceAccent = strings.ToLower(strings.TrimSpace(profile.InvoiceAccent))
	if len(profile.Name) < 2 || len(profile.Name) > 120 {
		return profile, fmt.Errorf("nom du commerce invalide")
	}
	if len(profile.Address) > 220 || len(profile.WhatsApp) > 60 || len(profile.Email) > 120 || len(profile.TaxID) > 80 {
		return profile, fmt.Errorf("une information du commerce est trop longue")
	}
	if profile.Email != "" && (!strings.Contains(profile.Email, "@") || strings.ContainsAny(profile.Email, "\r\n")) {
		return profile, fmt.Errorf("adresse email invalide")
	}
	if !validThemeColor(profile.InvoicePrimary) || !validThemeColor(profile.InvoiceAccent) {
		return profile, fmt.Errorf("couleurs de facture invalides")
	}
	return profile, nil
}

func decodeLogoDataURL(value string) ([]byte, error) {
	value = strings.TrimSpace(value)
	comma := strings.IndexByte(value, ',')
	if comma < 0 {
		return nil, fmt.Errorf("format du logo invalide")
	}
	header := strings.ToLower(value[:comma])
	if header != "data:image/png;base64" && header != "data:image/jpeg;base64" && header != "data:image/jpg;base64" {
		return nil, fmt.Errorf("le logo doit être PNG ou JPEG")
	}
	raw, err := base64.StdEncoding.DecodeString(value[comma+1:])
	if err != nil || len(raw) == 0 || len(raw) > 2<<20 {
		return nil, fmt.Errorf("logo invalide ou supérieur à 2 Mo")
	}
	config, format, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil || (format != "png" && format != "jpeg") || config.Width < 32 || config.Height < 32 || config.Width > 5000 || config.Height > 5000 {
		return nil, fmt.Errorf("dimensions du logo invalides")
	}
	return raw, nil
}

func normalizeDigitalService(service *DigitalService) error {
	service.Name = strings.TrimSpace(service.Name)
	service.URL = strings.TrimSpace(service.URL)
	service.Description = strings.TrimSpace(service.Description)
	service.Category = strings.TrimSpace(service.Category)
	service.Icon = strings.ToUpper(strings.TrimSpace(service.Icon))
	if len(service.Name) < 2 || len(service.Name) > 100 || len(service.Description) > 240 || len(service.Category) > 60 {
		return fmt.Errorf("informations du service invalides")
	}
	parsed, err := neturl.ParseRequestURI(service.URL)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "https" && parsed.Scheme != "http") {
		return fmt.Errorf("lien du service invalide: utilisez une adresse http ou https complète")
	}
	if len(service.Icon) < 1 || len(service.Icon) > 3 {
		runes := []rune(service.Name)
		if len(runes) > 2 {
			runes = runes[:2]
		}
		service.Icon = strings.ToUpper(string(runes))
	}
	return nil
}

func validateRestoredData(data *StoreData) error {
	if len(data.Products) > 100000 || len(data.Sales) > 1000000 || len(data.Purchases) > 1000000 || len(data.StockMovements) > 2000000 || len(data.CashMovements) > 1000000 ||
		len(data.Students) > 100000 || len(data.Lessons) > 2000000 || len(data.Exams) > 1000000 || len(data.DrivingPayments) > 1000000 || len(data.DrivingDocuments) > 1000000 || len(data.DrivingInvoices) > 1000000 {
		return fmt.Errorf("volume de sauvegarde invalide")
	}
	productIDs := map[int]bool{}
	barcodes := map[string]bool{}
	for i := range data.Products {
		if err := normalizeProduct(&data.Products[i]); err != nil {
			return fmt.Errorf("un produit de la sauvegarde est invalide: %w", err)
		}
		product := data.Products[i]
		barcode := strings.TrimSpace(product.Barcode)
		if product.ID <= 0 || productIDs[product.ID] || barcodes[barcode] {
			return fmt.Errorf("un produit de la sauvegarde est invalide")
		}
		productIDs[product.ID] = true
		barcodes[barcode] = true
	}
	supplierIDs := map[int]bool{}
	for _, supplier := range data.Suppliers {
		if supplier.ID <= 0 || strings.TrimSpace(supplier.Name) == "" || supplier.Balance < 0 || supplierIDs[supplier.ID] {
			return fmt.Errorf("un fournisseur de la sauvegarde est invalide")
		}
		supplierIDs[supplier.ID] = true
	}
	serviceIDs := map[int]bool{}
	for i := range data.DigitalServices {
		if data.DigitalServices[i].ID <= 0 || serviceIDs[data.DigitalServices[i].ID] {
			return fmt.Errorf("un service digital de la sauvegarde est invalide")
		}
		if err := normalizeDigitalService(&data.DigitalServices[i]); err != nil {
			return err
		}
		serviceIDs[data.DigitalServices[i].ID] = true
	}
	studentIDs := map[int]bool{}
	for _, student := range data.Students {
		if student.ID <= 0 || studentIDs[student.ID] || strings.TrimSpace(student.Name) == "" {
			return fmt.Errorf("un candidat de la sauvegarde est invalide")
		}
		studentIDs[student.ID] = true
	}
	for _, lesson := range data.Lessons {
		if lesson.ID <= 0 || !studentIDs[lesson.StudentID] || strings.TrimSpace(lesson.Start) == "" || lesson.Duration <= 0 {
			return fmt.Errorf("une séance de la sauvegarde est invalide")
		}
	}
	for _, exam := range data.Exams {
		if exam.ID <= 0 || !studentIDs[exam.StudentID] || strings.TrimSpace(exam.Date) == "" {
			return fmt.Errorf("un examen de la sauvegarde est invalide")
		}
	}
	for _, payment := range data.DrivingPayments {
		if payment.ID <= 0 || !studentIDs[payment.StudentID] || payment.Amount <= 0 {
			return fmt.Errorf("un paiement de la sauvegarde est invalide")
		}
	}
	for _, document := range data.DrivingDocuments {
		if document.ID <= 0 || !studentIDs[document.StudentID] || strings.TrimSpace(document.Category) == "" {
			return fmt.Errorf("un document candidat de la sauvegarde est invalide")
		}
	}
	for _, invoice := range data.DrivingInvoices {
		if invoice.ID <= 0 || !studentIDs[invoice.StudentID] || invoice.Amount <= 0 {
			return fmt.Errorf("une facture de la sauvegarde est invalide")
		}
	}
	profile, err := normalizeBusinessProfile(data.Settings.Business)
	if err != nil {
		return err
	}
	data.Settings.Business = profile
	if !validThemeColor(data.Settings.Theme) {
		return fmt.Errorf("thème de sauvegarde invalide")
	}
	return nil
}

func existingAppRunning(url string) bool {
	client := http.Client{Timeout: 900 * time.Millisecond}
	resp, err := client.Get(strings.TrimRight(url, "/") + "/api/health")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 64))
	return resp.StatusCode == http.StatusOK && string(b) == "AutoEcole-Pilot-Pro-v1"
}

func buildDailySalesReportPDF(data StoreData, day string) []byte {
	sales := make([]Sale, 0)
	for _, sale := range data.Sales {
		if strings.HasPrefix(sale.Time, day) {
			sales = append(sales, sale)
		}
	}
	sort.Slice(sales, func(i, j int) bool { return sales[i].Time < sales[j].Time })
	const rowsPerPage = 22
	pageCount := (len(sales) + rowsPerPage - 1) / rowsPerPage
	if pageCount == 0 {
		pageCount = 1
	}
	var cashTotal, cardTotal, creditTotal, profit float64
	validCount := 0
	for _, sale := range sales {
		if sale.Status == "annulée" {
			continue
		}
		validCount++
		switch sale.Payment {
		case "cash":
			cashTotal += sale.Total
		case "card":
			cardTotal += sale.Total
		case "credit":
			creditTotal += sale.Total
		}
		for _, item := range sale.Items {
			profit += float64(item.Qty) * (item.UnitPrice - item.UnitCost)
		}
		profit -= sale.Discount
	}
	var cashIn, cashOut float64
	for _, movement := range data.CashMovements {
		if !strings.HasPrefix(movement.Time, day) {
			continue
		}
		if movement.Type == "entrée" {
			cashIn += movement.Amount
		} else {
			cashOut += movement.Amount
		}
	}
	profile := data.Settings.Business
	if strings.TrimSpace(profile.Name) == "" {
		profile = defaultBusinessProfile()
	}
	primary := pdfRGB(profile.InvoicePrimary, "0.03 0.11 0.14")
	accent := pdfRGB(profile.InvoiceAccent, "0 0.72 0.58")
	streams := make([]string, 0, pageCount)
	for pageIndex := 0; pageIndex < pageCount; pageIndex++ {
		var content strings.Builder
		content.WriteString(primary + " rg 0 782 595 60 re f\n")
		pdfText(&content, 42, 810, 18, profile.Name+" - Cloture de caisse", "1 1 1")
		pdfText(&content, 42, 790, 9, "Rapport journalier du "+day, "0.82 0.91 0.93")
		content.WriteString("0.94 0.96 0.96 rg 38 731 519 29 re f\n")
		pdfText(&content, 45, 742, 8, "HEURE", "0.28 0.35 0.39")
		pdfText(&content, 102, 742, 8, "VENTE", "0.28 0.35 0.39")
		pdfText(&content, 160, 742, 8, "PAIEMENT", "0.28 0.35 0.39")
		pdfText(&content, 265, 742, 8, "ARTICLES", "0.28 0.35 0.39")
		pdfText(&content, 350, 742, 8, "VENDEUR", "0.28 0.35 0.39")
		pdfText(&content, 485, 742, 8, "TOTAL", "0.28 0.35 0.39")
		start := pageIndex * rowsPerPage
		end := start + rowsPerPage
		if end > len(sales) {
			end = len(sales)
		}
		y := 710.0
		if start == end {
			pdfText(&content, 48, y, 11, "Aucune vente pour cette date.", "0.35 0.40 0.43")
		} else {
			for _, sale := range sales[start:end] {
				timeLabel := sale.Time
				if parsed, err := time.Parse(time.RFC3339, sale.Time); err == nil {
					timeLabel = parsed.Format("15:04")
				}
				payment := map[string]string{"cash": "Especes", "card": "Carte", "credit": "Credit"}[sale.Payment]
				if sale.Status == "annulée" {
					payment = "ANNULEE"
				}
				qty := 0
				for _, item := range sale.Items {
					qty += item.Qty
				}
				pdfText(&content, 45, y, 9, timeLabel, "0.08 0.13 0.17")
				pdfText(&content, 102, y, 9, fmt.Sprintf("#%d", sale.ID), "0.08 0.13 0.17")
				pdfText(&content, 160, y, 8, payment, "0.08 0.13 0.17")
				pdfText(&content, 270, y, 9, strconv.Itoa(qty), "0.08 0.13 0.17")
				pdfText(&content, 350, y, 8, pdfShort(sale.Seller, 18), "0.08 0.13 0.17")
				pdfText(&content, 485, y, 9, fmt.Sprintf("%.3f", sale.Total), accent)
				content.WriteString(fmt.Sprintf("0.90 0.92 0.93 RG 42 %.1f m 553 %.1f l S\n", y-10, y-10))
				y -= 24
			}
		}
		if pageIndex == pageCount-1 {
			content.WriteString("0.91 0.97 0.95 rg 38 76 519 116 re f\n")
			pdfText(&content, 50, 166, 9, fmt.Sprintf("Ventes valides: %d", validCount), "0.10 0.30 0.24")
			pdfText(&content, 50, 146, 9, fmt.Sprintf("Especes: %.3f DT", cashTotal), "0.10 0.30 0.24")
			pdfText(&content, 50, 126, 9, fmt.Sprintf("Carte: %.3f DT", cardTotal), "0.10 0.30 0.24")
			pdfText(&content, 50, 106, 9, fmt.Sprintf("Credit: %.3f DT", creditTotal), "0.10 0.30 0.24")
			pdfText(&content, 285, 164, 10, fmt.Sprintf("CA TOTAL: %.3f DT", cashTotal+cardTotal+creditTotal), accent)
			pdfText(&content, 285, 140, 10, fmt.Sprintf("MARGE ESTIMEE: %.3f DT", profit), accent)
			pdfText(&content, 285, 116, 9, fmt.Sprintf("Mouvements caisse: +%.3f / -%.3f DT", cashIn, cashOut), "0.10 0.30 0.24")
			pdfText(&content, 285, 94, 10, fmt.Sprintf("ESPECES THEORIQUES: %.3f DT", cashTotal+cashIn-cashOut), "0.04 0.12 0.15")
		}
		pdfText(&content, 42, 38, 8, fmt.Sprintf("Page %d / %d - AutoÉcole Pilot Pro v1", pageIndex+1, pageCount), "0.45 0.49 0.52")
		streams = append(streams, content.String())
	}
	return buildPDFFromStreams(streams)
}

func buildDailySIMReportPDF(data StoreData, day string) []byte {
	transfers := make([]Transfer, 0)
	for _, tr := range data.Transfers {
		if strings.HasPrefix(tr.Time, day) {
			transfers = append(transfers, tr)
		}
	}
	sort.Slice(transfers, func(i, j int) bool { return transfers[i].Time < transfers[j].Time })

	const rowsPerPage = 22
	pageCount := (len(transfers) + rowsPerPage - 1) / rowsPerPage
	if pageCount == 0 {
		pageCount = 1
	}

	totalRevenue := 0.0
	totalMB := 0
	for _, tr := range transfers {
		totalRevenue += tr.Price
		totalMB += tr.PackageMB
	}

	pageStreams := make([]string, 0, pageCount)
	profile := data.Settings.Business
	if strings.TrimSpace(profile.Name) == "" {
		profile = defaultBusinessProfile()
	}
	primary := pdfRGB(profile.InvoicePrimary, "0.075 0.165 0.212")
	accent := pdfRGB(profile.InvoiceAccent, "0.08 0.48 0.35")
	for pageIndex := 0; pageIndex < pageCount; pageIndex++ {
		var content strings.Builder
		content.WriteString(primary + " rg 0 782 595 60 re f\n")
		pdfText(&content, 42, 810, 18, profile.Name+" - Rapport ventes SIM", "1 1 1")
		pdfText(&content, 42, 790, 9, "Rapport journalier du "+day, "0.86 0.93 0.96")

		content.WriteString("0.94 0.96 0.96 rg 38 731 519 29 re f\n")
		pdfText(&content, 45, 742, 8, "HEURE", "0.28 0.35 0.39")
		pdfText(&content, 100, 742, 8, "SIM", "0.28 0.35 0.39")
		pdfText(&content, 205, 742, 8, "CLIENT", "0.28 0.35 0.39")
		pdfText(&content, 350, 742, 8, "FORFAIT", "0.28 0.35 0.39")
		pdfText(&content, 410, 742, 8, "VENDEUR", "0.28 0.35 0.39")
		pdfText(&content, 492, 742, 8, "RECETTE", "0.28 0.35 0.39")

		start := pageIndex * rowsPerPage
		end := start + rowsPerPage
		if end > len(transfers) {
			end = len(transfers)
		}
		y := 710.0
		if start == end {
			pdfText(&content, 48, y, 11, "Aucune vente SIM enregistree pour cette date.", "0.35 0.40 0.43")
		} else {
			for _, tr := range transfers[start:end] {
				timeLabel := tr.Time
				if parsed, err := time.Parse(time.RFC3339, tr.Time); err == nil {
					timeLabel = parsed.Format("15:04")
				} else if len(timeLabel) >= 16 {
					timeLabel = timeLabel[11:16]
				}
				packageLabel := "500 Mo"
				if tr.PackageMB == 1000 {
					packageLabel = "1 Go"
				}
				clientLabel := strings.TrimSpace(strings.TrimSpace(tr.ClientName) + " " + strings.TrimSpace(tr.DestPhone))
				if clientLabel == "" {
					clientLabel = "Non renseigne"
				}
				seller := strings.TrimSpace(tr.SoldBy)
				if seller == "" {
					seller = "Admin"
				}
				pdfText(&content, 45, y, 9, timeLabel, "0.08 0.13 0.17")
				pdfText(&content, 100, y, 9, pdfShort(tr.SIMLabel, 15), "0.08 0.13 0.17")
				pdfText(&content, 205, y, 8, pdfShort(clientLabel, 24), "0.08 0.13 0.17")
				pdfText(&content, 350, y, 9, packageLabel, "0.08 0.13 0.17")
				pdfText(&content, 410, y, 8, pdfShort(seller, 12), "0.08 0.13 0.17")
				pdfText(&content, 492, y, 9, fmt.Sprintf("%.3f", tr.Price), accent)
				content.WriteString(fmt.Sprintf("0.90 0.92 0.93 RG 42 %.1f m 553 %.1f l S\n", y-10, y-10))
				y -= 24
			}
		}

		if pageIndex == pageCount-1 {
			content.WriteString("0.91 0.96 0.94 rg 38 92 519 76 re f\n")
			pdfText(&content, 52, 144, 11, fmt.Sprintf("Nombre de ventes: %d", len(transfers)), "0.10 0.30 0.24")
			pdfText(&content, 52, 122, 11, fmt.Sprintf("Volume vendu: %d Mo", totalMB), "0.10 0.30 0.24")
			pdfText(&content, 335, 128, 16, fmt.Sprintf("TOTAL: %.3f DT", totalRevenue), accent)
		}
		pdfText(&content, 42, 38, 8, fmt.Sprintf("Page %d / %d - Genere localement par %s", pageIndex+1, pageCount, profile.Name), "0.45 0.49 0.52")
		pageStreams = append(pageStreams, content.String())
	}

	return buildPDFFromStreams(pageStreams)
}

func simUsageForDay(transfers []Transfer, simID int, day string) int {
	used := 0
	for _, tr := range transfers {
		if tr.SIMID == simID && strings.HasPrefix(tr.Time, day) {
			used += tr.PackageMB
		}
	}
	return used
}

func buildSaleInvoicePDF(sale Sale, profile BusinessProfile, logoBytes []byte) []byte {
	const rowsPerPage = 12
	pageCount := (len(sale.Items) + rowsPerPage - 1) / rowsPerPage
	if pageCount == 0 {
		pageCount = 1
	}
	dateLabel := sale.Time
	if parsed, err := time.Parse(time.RFC3339, sale.Time); err == nil {
		dateLabel = parsed.Format("02/01/2006 15:04")
	}
	paymentLabel := "Especes"
	if sale.Payment == "credit" {
		paymentLabel = "Credit"
	} else if sale.Payment == "card" {
		paymentLabel = "Carte bancaire"
	}
	clientLabel := strings.TrimSpace(sale.Client)
	if clientLabel == "" {
		clientLabel = "Client comptoir"
	}
	if strings.TrimSpace(profile.Name) == "" {
		profile = defaultBusinessProfile()
	}
	primary := pdfRGB(profile.InvoicePrimary, "0.03 0.05 0.19")
	accent := pdfRGB(profile.InvoiceAccent, "0.53 0.61 0.16")

	streams := make([]string, 0, pageCount)
	for pageIndex := 0; pageIndex < pageCount; pageIndex++ {
		var content strings.Builder
		content.WriteString(primary + " rg 42 700 511 110 re f\n")
		content.WriteString("1 1 1 rg 42 700 170 110 re f\n")
		content.WriteString(accent + " rg 472 810 m 553 810 l 553 700 l 438 700 l 466 735 l h f\n")
		content.WriteString("q 145 0 0 117 55 696 cm /Logo Do Q\n")
		pdfText(&content, 235, 768, 18, "FACTURE CLIENT", "1 1 1")
		pdfText(&content, 235, 744, 10, fmt.Sprintf("FACTURE N. %04d", sale.ID), "0.82 0.86 0.94")
		pdfText(&content, 235, 724, 9, dateLabel, "0.82 0.86 0.94")
		if sale.Status == "annulée" {
			pdfText(&content, 420, 724, 10, "ANNULEE", "1 0.55 0.55")
		}

		content.WriteString("0.96 0.97 0.97 rg 42 601 511 82 re f\n")
		content.WriteString("0.88 0.90 0.92 RG 297 610 m 297 674 l S\n")
		pdfText(&content, 55, 663, 8, "FACTURE POUR", "0.34 0.39 0.44")
		pdfText(&content, 55, 645, 11, pdfShort(clientLabel, 35), "0.04 0.06 0.16")
		if sale.Phone != "" {
			pdfText(&content, 55, 628, 8, "Telephone: "+sale.Phone, "0.34 0.39 0.44")
		}
		pdfText(&content, 315, 663, 9, strings.ToUpper(profile.Name), "0.04 0.06 0.16")
		contactLines := make([]string, 0, 4)
		if profile.Address != "" {
			contactLines = append(contactLines, profile.Address)
		}
		if profile.WhatsApp != "" {
			contactLines = append(contactLines, "WhatsApp: "+profile.WhatsApp)
		}
		if profile.Email != "" {
			contactLines = append(contactLines, profile.Email)
		}
		if profile.TaxID != "" {
			contactLines = append(contactLines, "Matricule fiscal: "+profile.TaxID)
		}
		contactY := 647.0
		for _, line := range contactLines[:minInt(len(contactLines), 4)] {
			pdfText(&content, 315, contactY, 7, pdfShort(line, 42), "0.34 0.39 0.44")
			contactY -= 12
		}

		content.WriteString(primary + " rg 42 558 511 28 re f\n")
		pdfText(&content, 52, 568, 8, "N.", "1 1 1")
		pdfText(&content, 82, 568, 8, "DESIGNATION", "1 1 1")
		pdfText(&content, 326, 568, 8, "PRIX UNIT.", "1 1 1")
		pdfText(&content, 402, 568, 8, "QTE", "1 1 1")
		pdfText(&content, 474, 568, 8, "TOTAL", "1 1 1")

		start := pageIndex * rowsPerPage
		end := start + rowsPerPage
		if end > len(sale.Items) {
			end = len(sale.Items)
		}
		rowTop := 558.0
		content.WriteString("0.77 0.79 0.82 RG 0.7 w\n")
		for itemIndex, item := range sale.Items[start:end] {
			rowBottom := rowTop - 25
			pdfText(&content, 53, rowTop-17, 8, strconv.Itoa(start+itemIndex+1), "0.10 0.12 0.18")
			pdfText(&content, 82, rowTop-17, 9, pdfShort(item.Name, 43), "0.10 0.12 0.18")
			pdfText(&content, 326, rowTop-17, 9, fmt.Sprintf("%.3f", item.UnitPrice), "0.10 0.12 0.18")
			pdfText(&content, 405, rowTop-17, 9, strconv.Itoa(item.Qty), "0.10 0.12 0.18")
			pdfText(&content, 474, rowTop-17, 9, fmt.Sprintf("%.3f DT", item.Total), "0.10 0.12 0.18")
			content.WriteString(fmt.Sprintf("42 %.1f m 553 %.1f l S\n", rowBottom, rowBottom))
			rowTop = rowBottom
		}
		content.WriteString(fmt.Sprintf("42 586 m 553 586 l 553 %.1f l 42 %.1f l h S\n", rowTop, rowTop))
		for _, x := range []int{72, 310, 390, 432} {
			content.WriteString(fmt.Sprintf("%d 586 m %d %.1f l S\n", x, x, rowTop))
		}

		if pageIndex == pageCount-1 {
			pdfText(&content, 42, 194, 8, "MODE DE PAIEMENT", "0.34 0.39 0.44")
			pdfText(&content, 42, 176, 11, paymentLabel, "0.04 0.06 0.16")
			if sale.Seller != "" {
				pdfText(&content, 42, 128, 8, "Vendeur: "+pdfShort(sale.Seller, 24), "0.34 0.39 0.44")
			}
			if sale.Payment == "cash" {
				pdfText(&content, 42, 158, 8, fmt.Sprintf("Recu: %.3f DT", sale.Received), "0.34 0.39 0.44")
				pdfText(&content, 42, 144, 8, fmt.Sprintf("Monnaie: %.3f DT", sale.Change), "0.34 0.39 0.44")
			}
			content.WriteString("0.96 0.97 0.97 rg 330 110 223 102 re f\n")
			content.WriteString("0.77 0.79 0.82 RG 330 110 223 102 re S 430 110 m 430 212 l S 330 178 m 553 178 l S 330 148 m 553 148 l S\n")
			subtotal := sale.Subtotal
			if subtotal <= 0 {
				subtotal = sale.Total + sale.Discount
			}
			pdfText(&content, 344, 190, 8, "SOUS-TOTAL", "0.22 0.26 0.31")
			pdfText(&content, 458, 190, 9, fmt.Sprintf("%.3f DT", subtotal), "0.04 0.06 0.16")
			pdfText(&content, 344, 158, 8, "REMISE", "0.22 0.26 0.31")
			pdfText(&content, 458, 158, 9, fmt.Sprintf("-%.3f DT", sale.Discount), "0.04 0.06 0.16")
			pdfText(&content, 344, 124, 10, "TOTAL A PAYER", "0.04 0.06 0.16")
			pdfText(&content, 452, 124, 13, fmt.Sprintf("%.3f DT", sale.Total), accent)
			content.WriteString("0.34 0.39 0.44 RG 358 91 m 525 91 l S\n")
			pdfText(&content, 405, 77, 8, "SIGNATURE / CACHET", "0.34 0.39 0.44")
		}
		pdfText(&content, 42, 78, 8, fmt.Sprintf("Facture %04d - Page %d / %d", sale.ID, pageIndex+1, pageCount), "0.34 0.39 0.44")
		content.WriteString(primary + " rg 42 28 511 35 re f\n")
		content.WriteString(accent + " rg 42 28 86 35 re f 128 28 m 152 63 l 176 63 l 152 28 l h f\n")
		pdfText(&content, 205, 42, 9, "MERCI POUR VOTRE CONFIANCE", "1 1 1")
		streams = append(streams, content.String())
	}
	logo, err := newPDFImage(logoBytes)
	if err != nil {
		return buildPDFFromStreams(streams)
	}
	return buildPDFFromStreamsWithImage(streams, logo)
}

func buildPDFFromStreams(pageStreams []string) []byte {
	return buildPDFFromStreamsWithImage(pageStreams, nil)
}

type pdfImage struct {
	width  int
	height int
	data   []byte
}

func newPDFImage(source []byte) (*pdfImage, error) {
	img, _, err := image.Decode(bytes.NewReader(source))
	if err != nil {
		return nil, err
	}
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	raw := make([]byte, 0, width*height*3)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := img.At(x, y).RGBA()
			// image.Color values are alpha-premultiplied. Composite transparent
			// pixels over white so PNG logos do not render with a black box.
			r += 0xffff - a
			g += 0xffff - a
			b += 0xffff - a
			raw = append(raw, byte(r>>8), byte(g>>8), byte(b>>8))
		}
	}
	var compressed bytes.Buffer
	zw := zlib.NewWriter(&compressed)
	if _, err := zw.Write(raw); err != nil {
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return &pdfImage{width: width, height: height, data: compressed.Bytes()}, nil
}

func buildPDFFromStreamsWithImage(pageStreams []string, logo *pdfImage) []byte {
	pageCount := len(pageStreams)
	pageIDs := make([]int, pageCount)
	contentIDs := make([]int, pageCount)
	for i := 0; i < pageCount; i++ {
		pageIDs[i] = 4 + i*2
		contentIDs[i] = pageIDs[i] + 1
	}
	kids := make([]string, 0, pageCount)
	for _, id := range pageIDs {
		kids = append(kids, fmt.Sprintf("%d 0 R", id))
	}
	objects := [][]byte{
		[]byte("<< /Type /Catalog /Pages 2 0 R >>"),
		[]byte(fmt.Sprintf("<< /Type /Pages /Kids [%s] /Count %d >>", strings.Join(kids, " "), pageCount)),
		[]byte("<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica /Encoding /WinAnsiEncoding >>"),
	}
	imageID := 4 + pageCount*2
	for i, stream := range pageStreams {
		resources := "/Font << /F1 3 0 R >>"
		if logo != nil {
			resources += fmt.Sprintf(" /XObject << /Logo %d 0 R >>", imageID)
		}
		objects = append(objects,
			[]byte(fmt.Sprintf("<< /Type /Page /Parent 2 0 R /MediaBox [0 0 595 842] /Resources << %s >> /Contents %d 0 R >>", resources, contentIDs[i])),
			[]byte(fmt.Sprintf("<< /Length %d >>\nstream\n%sendstream", len([]byte(stream)), stream)),
		)
	}
	if logo != nil {
		var imageObject bytes.Buffer
		fmt.Fprintf(&imageObject, "<< /Type /XObject /Subtype /Image /Width %d /Height %d /ColorSpace /DeviceRGB /BitsPerComponent 8 /Filter /FlateDecode /Length %d >>\nstream\n", logo.width, logo.height, len(logo.data))
		imageObject.Write(logo.data)
		imageObject.WriteString("\nendstream")
		objects = append(objects, imageObject.Bytes())
	}
	var pdf bytes.Buffer
	pdf.WriteString("%PDF-1.4\n%\xe2\xe3\xcf\xd3\n")
	offsets := make([]int, len(objects)+1)
	for i, object := range objects {
		offsets[i+1] = pdf.Len()
		fmt.Fprintf(&pdf, "%d 0 obj\n", i+1)
		pdf.Write(object)
		pdf.WriteString("\nendobj\n")
	}
	xrefOffset := pdf.Len()
	fmt.Fprintf(&pdf, "xref\n0 %d\n", len(objects)+1)
	pdf.WriteString("0000000000 65535 f \n")
	for i := 1; i <= len(objects); i++ {
		fmt.Fprintf(&pdf, "%010d 00000 n \n", offsets[i])
	}
	fmt.Fprintf(&pdf, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objects)+1, xrefOffset)
	return pdf.Bytes()
}

func pdfText(b *strings.Builder, x, y float64, size int, value, color string) {
	fmt.Fprintf(b, "%s rg BT /F1 %d Tf 1 0 0 1 %.1f %.1f Tm (%s) Tj ET\n", color, size, x, y, pdfEscape(value))
}

func pdfEscape(value string) string {
	value = pdfWinANSI(value)
	value = strings.ReplaceAll(value, "\\", "\\\\")
	value = strings.ReplaceAll(value, "(", "\\(")
	value = strings.ReplaceAll(value, ")", "\\)")
	return value
}

func pdfWinANSI(value string) string {
	extra := map[rune]byte{'€': 0x80, '‚': 0x82, 'ƒ': 0x83, '„': 0x84, '…': 0x85, '†': 0x86, '‡': 0x87, 'ˆ': 0x88, '‰': 0x89, 'Š': 0x8A, '‹': 0x8B, 'Œ': 0x8C, 'Ž': 0x8E, '‘': 0x91, '’': 0x92, '“': 0x93, '”': 0x94, '•': 0x95, '–': 0x96, '—': 0x97, '˜': 0x98, '™': 0x99, 'š': 0x9A, '›': 0x9B, 'œ': 0x9C, 'ž': 0x9E, 'Ÿ': 0x9F}
	var out strings.Builder
	for _, r := range value {
		if r >= 32 && r <= 255 {
			out.WriteByte(byte(r))
		} else if b, ok := extra[r]; ok {
			out.WriteByte(b)
		} else if r == '\n' || r == '\r' || r == '\t' {
			out.WriteByte(' ')
		} else {
			out.WriteByte('?')
		}
	}
	return out.String()
}

func pdfShort(value string, max int) string {
	runes := []rune(value)
	if len(runes) <= max {
		return value
	}
	if max <= 3 {
		return string(runes[:max])
	}
	return string(runes[:max-3]) + "..."
}

func pdfRGB(value, fallback string) string {
	value = strings.TrimSpace(value)
	if !validThemeColor(value) {
		return fallback
	}
	r, _ := strconv.ParseUint(value[1:3], 16, 8)
	g, _ := strconv.ParseUint(value[3:5], 16, 8)
	b, _ := strconv.ParseUint(value[5:7], 16, 8)
	return fmt.Sprintf("%.3f %.3f %.3f", float64(r)/255, float64(g)/255, float64(b)/255)
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// kept to make accidental string numeric conversions explicit in future extensions
func atoi(s string) int { n, _ := strconv.Atoi(s); return n }
