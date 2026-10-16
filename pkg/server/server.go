package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	qrcode "github.com/skip2/go-qrcode"

	"github.com/munim-app/munim/pkg/db"
	"github.com/munim-app/munim/pkg/export"
	"github.com/munim-app/munim/pkg/models"
	"github.com/munim-app/munim/pkg/ollama"
	"github.com/munim-app/munim/pkg/preproc"
	"github.com/munim-app/munim/pkg/pricealert"
	"github.com/munim-app/munim/pkg/validator"
	"github.com/munim-app/munim/web"
)

// Server coordinates HTTP endpoints, local uploads, and web UI.
type Server struct {
	store       *db.Store
	ollama      *ollama.Client
	alertEngine *pricealert.Engine
	uploadDir   string
	port        int
}

// Config defines server initialization arguments.
type Config struct {
	Port        int
	Store       *db.Store
	Ollama      *ollama.Client
	AlertEngine *pricealert.Engine
	UploadDir   string
}

func NewServer(cfg Config) *Server {
	if cfg.Port == 0 {
		cfg.Port = 8080
	}
	if cfg.UploadDir == "" {
		cfg.UploadDir = "data/bills"
	}
	_ = os.MkdirAll(cfg.UploadDir, 0755)

	return &Server{
		store:       cfg.Store,
		ollama:      cfg.Ollama,
		alertEngine: cfg.AlertEngine,
		uploadDir:   cfg.UploadDir,
		port:        cfg.Port,
	}
}

// Start launches the server and prints local network QR code in ASCII and image endpoint.
func (s *Server) Start() error {
	mux := http.NewServeMux()

	// Static UI assets
	mux.Handle("/", http.FileServer(http.FS(web.StaticFS)))

	// API endpoints
	mux.HandleFunc("/api/upload", s.handleUpload)
	mux.HandleFunc("/api/bills", s.handleBills)
	mux.HandleFunc("/api/alerts", s.handleAlerts)
	mux.HandleFunc("/api/dues", s.handleDues)
	mux.HandleFunc("/api/bills/review", s.handleReviewBill)
	mux.HandleFunc("/api/bills/pay", s.handleRecordPayment)
	mux.HandleFunc("/api/export/ca", s.handleExportCA)
	mux.HandleFunc("/api/export/tally", s.handleExportTally)
	mux.HandleFunc("/api/qr", s.handleQRCode)
	mux.HandleFunc("/api/lan-url", s.handleLANURL)
	mux.HandleFunc("/api/settings", s.handleSettings)
	mux.HandleFunc("/bills/", s.handleServeBillImage)

	lanIP := getLocalIP()
	localhostURL := fmt.Sprintf("http://localhost:%d", s.port)
	var serverURL string
	hasLAN := lanIP != "" && lanIP != "127.0.0.1"

	fmt.Println("\n========================================================")
	fmt.Println("       मुनीम (MUNIM) — Local-First Bill Diary")
	fmt.Println("========================================================")
	fmt.Printf(" [Desktop Browser] : %s\n", localhostURL)
	if hasLAN {
		serverURL = fmt.Sprintf("http://%s:%d", lanIP, s.port)
		fmt.Printf(" [Phone on WiFi]   : %s\n", serverURL)
		fmt.Println("--------------------------------------------------------")
		fmt.Println(" Scan this QR Code with your phone to upload bills:")
		printTerminalQR(serverURL)
	} else {
		serverURL = localhostURL
		fmt.Println(" [Phone on WiFi]   : [No active LAN connection found]")
		fmt.Println(" (Connect to your Wi-Fi router to upload bills from mobile camera)")
	}
	fmt.Println("========================================================")

	addr := fmt.Sprintf("0.0.0.0:%d", s.port)
	return http.ListenAndServe(addr, mux)
}

func printTerminalQR(url string) {
	qr, err := qrcode.New(url, qrcode.Medium)
	if err == nil {
		fmt.Println(qr.ToSmallString(false))
	}
}

func (s *Server) handleQRCode(w http.ResponseWriter, r *http.Request) {
	lanIP := getLocalIP()
	targetURL := fmt.Sprintf("http://localhost:%d", s.port)
	if lanIP != "" && lanIP != "127.0.0.1" {
		targetURL = fmt.Sprintf("http://%s:%d", lanIP, s.port)
	}
	png, err := qrcode.Encode(targetURL, qrcode.Medium, 256)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Write(png)
}

func (s *Server) handleLANURL(w http.ResponseWriter, r *http.Request) {
	lanIP := getLocalIP()
	hasLAN := lanIP != "" && lanIP != "127.0.0.1"
	url := fmt.Sprintf("http://localhost:%d", s.port)
	if hasLAN {
		url = fmt.Sprintf("http://%s:%d", lanIP, s.port)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"lan_ip":  lanIP,
		"url":     url,
		"has_lan": hasLAN,
	})
}

func (s *Server) handleServeBillImage(w http.ResponseWriter, r *http.Request) {
	cleanPath := filepath.Clean(r.URL.Path[len("/bills/"):])
	fullPath := filepath.Join(s.uploadDir, cleanPath)
	http.ServeFile(w, r, fullPath)
}

func (s *Server) handleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 32MB max upload
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		http.Error(w, fmt.Sprintf("Failed to parse form: %v", err), http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("bill")
	if err != nil {
		http.Error(w, "No bill file uploaded", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Preprocess image
	filename := fmt.Sprintf("bill_%d_%s", time.Now().UnixNano(), filepath.Base(header.Filename))
	procImg, err := preproc.ProcessBytes(file, filename, s.uploadDir)
	if err != nil {
		http.Error(w, fmt.Sprintf("Preprocessing failed: %v", err), http.StatusInternalServerError)
		return
	}

	ctx := r.Context()

	// Extract via Gemma (supporting vertical tiling if tall image)
	extraction, err := s.ollama.ExtractBill(ctx, procImg.Base64Data, procImg.Tiles)
	if err != nil {
		// Log structured error without sensitive info
		log.Printf("Ollama extraction failed: %v", err)
		http.Error(w, fmt.Sprintf("AI extraction failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Deterministic validation
	valRes := validator.ValidateInvoice(extraction)

	// If failed, 1-shot repair retry
	if !valRes.IsValid {
		priorJSON, _ := json.Marshal(extraction)
		repairExtraction, repairErr := s.ollama.RepairBill(ctx, procImg.Base64Data, string(priorJSON), valRes.Errors)
		if repairErr == nil {
			repairVal := validator.ValidateInvoice(repairExtraction)
			if repairVal.IsValid {
				extraction = repairExtraction
				valRes = repairVal
			}
		}
	}

	invDate, _ := validator.ParseDate(extraction.InvoiceDate)
	var dueDate *time.Time
	if extraction.DueDate != nil && *extraction.DueDate != "" {
		if d, err := validator.ParseDate(*extraction.DueDate); err == nil {
			dueDate = &d
		}
	}

	status := models.StatusVerified
	if !valRes.IsValid {
		status = models.StatusNeedsReview
	}

	relImagePath := filepath.Base(procImg.SavedPath)

	// Check for duplicates before inserting (supplier + invoice number + date OR near-identical perceptual hash)
	confirmDuplicate := r.FormValue("confirm_duplicate") == "true"
	dupBill, dupReason, dupErr := s.store.FindDuplicateBill(ctx, extraction.VendorName, extraction.InvoiceNumber, invDate.Format("2006-01-02"), procImg.PerceptualHash)
	if dupErr == nil && dupBill != nil && !confirmDuplicate {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"duplicate":          true,
			"duplicate_message":  "This bill was already added (" + dupReason + ").",
			"existing_bill_id":   dupBill.ID,
			"existing_inv_num":   dupBill.InvoiceNumber,
			"existing_bill_date": dupBill.InvoiceDate.Format("2006-01-02"),
			"existing_image":     dupBill.ImagePath,
			"extraction":         extraction,
		})
		return
	}

	billRecord := &models.BillRecord{
		VendorName:       extraction.VendorName,
		VendorGSTIN:      extraction.VendorGSTIN,
		InvoiceNumber:    extraction.InvoiceNumber,
		InvoiceDate:      invDate,
		DueDate:          dueDate,
		Subtotal:         extraction.Subtotal,
		CGST:             extraction.CGST,
		SGST:             extraction.SGST,
		IGST:             extraction.IGST,
		RoundOff:         extraction.RoundOff,
		Total:            extraction.Total,
		ImagePath:        relImagePath,
		ValidationStatus: status,
		ValidationErrors: valRes.Errors,
		Confidence:       extraction.Confidence,
		LanguageDetected: extraction.LanguageDetected,
		LineItems:        extraction.LineItems,
		AmountPaid:       0.0,
		PaymentStatus:    "unpaid",
		ImageHash:        procImg.PerceptualHash,
	}

	billID, err := s.store.InsertBill(ctx, billRecord)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to store bill: %v", err), http.StatusInternalServerError)
		return
	}

	// Check price alerts across past history
	history, _ := s.store.GetHistoricalPurchases(ctx)
	var generatedAlerts []models.PriceAlert
	for _, item := range extraction.LineItems {
		itemAlerts := s.alertEngine.CheckItemPrice(billID, extraction.InvoiceNumber, relImagePath, invDate, extraction.VendorName, item, history)
		for _, a := range itemAlerts {
			_ = s.store.SavePriceAlert(ctx, a)
			generatedAlerts = append(generatedAlerts, a)
		}
	}

	resp := map[string]interface{}{
		"success":           true,
		"bill_id":           billID,
		"validation_status": status,
		"errors":            valRes.Errors,
		"plain_errors":      valRes.PlainErrors,
		"alerts_count":      len(generatedAlerts),
		"bill":              billRecord,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (s *Server) handleBills(w http.ResponseWriter, r *http.Request) {
	bills, err := s.store.GetAllBills(r.Context())
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(bills)
}

func (s *Server) handleAlerts(w http.ResponseWriter, r *http.Request) {
	alerts, err := s.store.GetAllPriceAlerts(r.Context())
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(alerts)
}

func (s *Server) handleDues(w http.ResponseWriter, r *http.Request) {
	dues, err := s.store.GetSupplierDues(r.Context())
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(dues)
}

func (s *Server) handleReviewBill(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", 405)
		return
	}

	var req struct {
		BillID        int64             `json:"bill_id"`
		Status        string            `json:"status"` // "approved"
		VendorName    string            `json:"vendor_name"`
		VendorGSTIN   string            `json:"vendor_gstin"`
		InvoiceNumber string            `json:"invoice_number"`
		InvoiceDate   string            `json:"invoice_date"`
		DueDate       *string           `json:"due_date"`
		Subtotal      float64           `json:"subtotal"`
		CGST          float64           `json:"cgst"`
		SGST          float64           `json:"sgst"`
		IGST          float64           `json:"igst"`
		RoundOff      float64           `json:"round_off"`
		Total         float64           `json:"total"`
		LineItems     []models.LineItem `json:"line_items"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	// Validate edited bill to determine if errors still remain
	invExtraction := &models.InvoiceExtraction{
		VendorName:    req.VendorName,
		VendorGSTIN:   req.VendorGSTIN,
		InvoiceNumber: req.InvoiceNumber,
		InvoiceDate:   req.InvoiceDate,
		DueDate:       req.DueDate,
		Subtotal:      req.Subtotal,
		CGST:          req.CGST,
		SGST:          req.SGST,
		IGST:          req.IGST,
		RoundOff:      req.RoundOff,
		Total:         req.Total,
		LineItems:     req.LineItems,
		Confidence:    0.95, // manually reviewed
	}
	valRes := validator.ValidateInvoice(invExtraction)

	parsedDate, _ := validator.ParseDate(req.InvoiceDate)
	var dueDate *time.Time
	if req.DueDate != nil && *req.DueDate != "" {
		if d, err := validator.ParseDate(*req.DueDate); err == nil {
			dueDate = &d
		}
	}

	billRecord := &models.BillRecord{
		ID:               req.BillID,
		VendorName:       req.VendorName,
		VendorGSTIN:      req.VendorGSTIN,
		InvoiceNumber:    req.InvoiceNumber,
		InvoiceDate:      parsedDate,
		DueDate:          dueDate,
		Subtotal:         req.Subtotal,
		CGST:             req.CGST,
		SGST:             req.SGST,
		IGST:             req.IGST,
		RoundOff:         req.RoundOff,
		Total:            req.Total,
		ValidationStatus: models.BillValidationStatus(req.Status),
		ValidationErrors: valRes.Errors,
		LineItems:        req.LineItems,
	}

	err := s.store.UpdateBillDetails(r.Context(), billRecord)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success":          true,
		"validation_valid": valRes.IsValid,
		"errors":           valRes.Errors,
		"plain_errors":     valRes.PlainErrors,
	})
}

func (s *Server) handleRecordPayment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", 405)
		return
	}

	var req struct {
		BillID      int64   `json:"bill_id"`
		Amount      float64 `json:"amount"`
		PaymentDate string  `json:"payment_date"`
		PaymentMode string  `json:"payment_mode"` // cash, UPI, bank
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	err := s.store.RecordPaymentWithDetails(r.Context(), req.BillID, req.Amount, req.PaymentDate, req.PaymentMode)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func (s *Server) handleExportCA(w http.ResponseWriter, r *http.Request) {
	monthStr := r.URL.Query().Get("month") // YYYY-MM
	var filterMonth time.Time
	if monthStr != "" {
		filterMonth, _ = time.Parse("2006-01", monthStr)
	}

	bills, err := s.store.GetAllBills(r.Context())
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	data, err := export.GenerateCAPack(bills, filterMonth)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	filename := fmt.Sprintf("Munim_CA_Pack_%s.csv", time.Now().Format("20060102"))
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	w.Write(data)
}

func (s *Server) handleExportTally(w http.ResponseWriter, r *http.Request) {
	monthStr := r.URL.Query().Get("month")
	var filterMonth time.Time
	if monthStr != "" {
		filterMonth, _ = time.Parse("2006-01", monthStr)
	}

	bills, err := s.store.GetAllBills(r.Context())
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	data, err := export.GenerateTallyVouchers(bills, filterMonth)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	filename := fmt.Sprintf("Munim_Tally_Vouchers_%s.csv", time.Now().Format("20060102"))
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	w.Write(data)
}

func (s *Server) handleSettings(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"model": s.ollama.ModelName(),
			"available_models": []string{
				ollama.ModelGemma3_4B,
				ollama.ModelGemma3_12B,
				ollama.ModelGemma3_27B,
			},
		})
		return
	}

	if r.Method == http.MethodPost {
		var req struct {
			Model string `json:"model"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}

		s.ollama.SetModel(req.Model)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"model":   s.ollama.ModelName(),
		})
		return
	}

	http.Error(w, "Method not allowed", 405)
}

// Get non-loopback local LAN IPv4 address (e.g. 192.168.x.x, 10.x.x.x)
func getLocalIP() string {
	// 1. Try dialing an outbound UDP socket (doesn't send packets, but reveals preferred outbound interface)
	conn, err := net.DialTimeout("udp", "8.8.8.8:80", 200*time.Millisecond)
	if err == nil {
		defer conn.Close()
		localAddr := conn.LocalAddr().(*net.UDPAddr)
		ip := localAddr.IP.To4()
		if ip != nil && !ip.IsLoopback() && !ip.IsLinkLocalUnicast() {
			return ip.String()
		}
	}

	// 2. Fallback to interface inspection, skipping 169.254.x.x link-local and 127.x.x.x loopbacks
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "127.0.0.1"
	}
	for _, addr := range addrs {
		if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
			ip := ipNet.IP.To4()
			if ip != nil && !ip.IsLinkLocalUnicast() {
				return ip.String()
			}
		}
	}
	return "127.0.0.1"
}
