// Munim Web App Logic
let currentLanguage = 'en'; // 'en' or 'hi'
let activeReviewBill = null;
let activePayBill = null;
let cachedBills = [];
let cachedAlerts = [];
let cachedDues = [];
let selectedSpendMonth = ""; // Format: YYYY-MM

// Shared currency formatter using Indian digit grouping (en-IN)
function formatCurrency(val) {
  if (typeof val !== 'number' || isNaN(val)) val = 0;
  return new Intl.NumberFormat('en-IN', {
    style: 'currency',
    currency: 'INR',
    minimumFractionDigits: 2,
    maximumFractionDigits: 2
  }).format(val);
}

// Complete Bilingual Dictionary for Every UI String
const translations = {
  en: {
    brand: "मुनीम (Munim)",
    sub: "Powered by Gemma, running on this computer",
    monthlySpend: "Spend",
    openAlerts: "Open Price Alerts",
    priceSpikes: "Price spikes & savings",
    supplierDues: "Supplier Dues",
    needsReview: "Needs Review",
    reviewSub: "Low confidence or math errors",
    uploadTitle: "Photograph or Upload Bill",
    uploadSub: "Take a photo from your phone or drop a supplier bill here. Local vision AI extracts items in seconds.",
    uploadBtn: "Upload / Capture Bill",
    alertsTitle: "Price-Change Alerts",
    duesTitle: "Supplier Dues & Outstanding",
    ledgerTitle: "Invoices & Bills Ledger",
    readingBill: "Munim is reading your bill...",
    readingSub: "Local vision AI analyzing items & calculating taxes...",
    qrNavBtn: "Phone QR",
    qrTitle: "Open on Mobile Camera",
    qrSub: "Scan this code with your phone while connected to the same Wi-Fi:",
    btnCaPack: "📥 CA Pack CSV",
    btnTally: "📊 Tally CSV",
    thSupplier: "Supplier Name",
    thTotalBilled: "Total Billed",
    thTotalPaid: "Total Paid",
    thBalanceDue: "Balance Due",
    thEarliestDue: "Due Date",
    thStatus: "Status",
    thAction: "Action",
    thInvNo: "Bill #",
    thInvDate: "Date",
    thInvSupplier: "Supplier",
    thInvTotal: "Amount",
    thInvVal: "Validation",
    thInvPayment: "Payment",
    thInvActions: "Actions",
    noAlerts: "No active price alerts. All supplier rates are stable.",
    noDues: "No outstanding supplier dues.",
    noBills: "No bills uploaded yet.",
    activeBadge: "Active",
    overdueBills: "overdue bills",
    billsProcessed: "bills processed",
    statusVerified: "Verified",
    statusApproved: "Approved",
    statusNeedsReview: "Needs Review",
    statusPaid: "Paid",
    statusPartial: "Partial",
    statusUnpaid: "Unpaid",
    statusOverdue: "OVERDUE",
    statusOk: "OK",
    btnPay: "Pay",
    btnEdit: "🔍 Edit",
    btnRecordPay: "💳 Pay",
    modalReviewTitle: "Review & Verify Bill",
    zoomHint: "Click image to toggle zoom",
    headerDetails: "Invoice Details",
    lblSupplier: "Supplier Name",
    lblGstin: "Supplier GSTIN",
    lblInvNum: "Invoice Number",
    lblInvDate: "Invoice Date",
    itemsDetails: "Line Items",
    taxTotals: "Taxes & Totals",
    lblSubtotal: "Subtotal (₹)",
    lblCgst: "CGST (₹)",
    lblSgst: "SGST (₹)",
    lblIgst: "IGST (₹)",
    lblRoundoff: "Round Off (₹)",
    lblGrandtotal: "Grand Total (₹)",
    lblTotalSummary: "Total",
    btnCancelReview: "Cancel",
    btnApproveSave: "Approve & Save",
    btnApproveAnyway: "Approve anyway",
    confirmApproveRemaining: "This bill still has validation discrepancies. Are you sure you want to approve it anyway?",
    modalPayTitle: "Record Supplier Payment",
    lblPaySupplier: "Supplier:",
    lblPayBillnum: "Bill #:",
    lblPayBalance: "Remaining Balance:",
    lblPayAmount: "Payment Amount (₹)",
    lblPayDate: "Payment Date",
    lblPayMode: "Payment Mode",
    btnSavePayment: "Record Payment",
    evidenceLabel: "Evidence",
    selectMonthHint: "Select month",
    allInvoices: "All Invoices",
    duplicateWarning: "This bill was already added. Do you want to add it anyway?",
    viewExisting: "View Existing Bill",
    daysOverdue: "days overdue",
  },
  hi: {
    brand: "मुनीम (Munim)",
    sub: "Gemma द्वारा संचालित, आपके अपने कंप्यूटर पर",
    monthlySpend: "कुल खर्च",
    openAlerts: "कीमत बढ़ोतरी अलर्ट",
    priceSpikes: "दाम में अंतर व बचत",
    supplierDues: "सप्लायर का बकाया",
    needsReview: "जांच की आवश्यकता",
    reviewSub: "कम स्पष्टता या जोड़ में अंतर",
    uploadTitle: "बिल की फोटो खींचें या अपलोड करें",
    uploadSub: "मोबाइल कैमरे से बिल की फोटो लें या यहाँ ड्रॉप करें। लोकल Gemma AI सेकंडों में सामान दर्ज करेगा।",
    uploadBtn: "बिल अपलोड करें",
    alertsTitle: "रेट में बदलाव की चेतावनी",
    duesTitle: "सप्लायरों का बकाया हिसाब",
    ledgerTitle: "सभी बिलों की डायरी",
    readingBill: "मुनीम बिल पढ़ रहा है...",
    readingSub: "लोकल Gemma मॉडल सामान और टैक्स की गणना कर रहा है...",
    qrNavBtn: "मोबाइल QR",
    qrTitle: "मोबाइल कैमरे से खोलें",
    qrSub: "एक ही वाई-फाई से जुड़े फोन से यह QR कोड स्कैन करें:",
    btnCaPack: "📥 CA पैक CSV",
    btnTally: "📊 टैली CSV",
    thSupplier: "सप्लायर का नाम",
    thTotalBilled: "कुल बिल राशि",
    thTotalPaid: "कुल भुगतान",
    thBalanceDue: "बकाया राशि",
    thEarliestDue: "अंतिम तारीख",
    thStatus: "स्थिति",
    thAction: "कार्रवाई",
    thInvNo: "बिल संख्या",
    thInvDate: "तारीख",
    thInvSupplier: "सप्लायर",
    thInvTotal: "रकम",
    thInvVal: "जांच स्थिति",
    thInvPayment: "भुगतान स्थिति",
    thInvActions: "विकल्प",
    noAlerts: "कोई सक्रिय मूल्य वृद्धि अलर्ट नहीं है। सभी रेट सामान्य हैं।",
    noDues: "कोई सप्लायर बकाया नहीं है।",
    noBills: "अभी कोई बिल अपलोड नहीं किया गया है।",
    activeBadge: "सक्रिय",
    overdueBills: "अवधि पार बिल",
    billsProcessed: "बिल दर्ज",
    statusVerified: "सत्यापित",
    statusApproved: "स्वीकृत",
    statusNeedsReview: "जांचें",
    statusPaid: "चुकता",
    statusPartial: "आंशिक",
    statusUnpaid: "बाकी",
    statusOverdue: "अवधि पार",
    statusOk: "सामान्य",
    btnPay: "भुगतान",
    btnEdit: "🔍 संपादित करें",
    btnRecordPay: "💳 भुगतान दर्ज करें",
    modalReviewTitle: "बिल की जांच व सुधार",
    zoomHint: "बड़ा करने के लिए फोटो पर क्लिक करें",
    headerDetails: "बिल विवरण",
    lblSupplier: "सप्लायर का नाम",
    lblGstin: "सप्लायर GSTIN",
    lblInvNum: "बिल संख्या",
    lblInvDate: "बिल तारीख",
    itemsDetails: "सामान सूची",
    taxTotals: "टैक्स और कुल योग",
    lblSubtotal: "सब-टोटल (₹)",
    lblCgst: "CGST (₹)",
    lblSgst: "SGST (₹)",
    lblIgst: "IGST (₹)",
    lblRoundoff: "राउंड ऑफ (₹)",
    lblGrandtotal: "कुल योग (₹)",
    lblTotalSummary: "कुल रकम",
    btnCancelReview: "रद्द करें",
    btnApproveSave: "स्वीकारें और सहेजें",
    btnApproveAnyway: "फिर भी स्वीकृत करें",
    confirmApproveRemaining: "इस बिल में अभी भी गणित या GSTIN में अंतर है। क्या आप सचमुच इसे स्वीकृत करना चाहते हैं?",
    modalPayTitle: "सप्लायर भुगतान दर्ज करें",
    lblPaySupplier: "सप्लायर:",
    lblPayBillnum: "बिल #:",
    lblPayBalance: "बाकी रकम:",
    lblPayAmount: "भुगतान राशि (₹)",
    lblPayDate: "भुगतान तारीख",
    lblPayMode: "भुगतान का माध्यम",
    btnSavePayment: "भुगतान सहेजें",
    evidenceLabel: "प्रमाण",
    selectMonthHint: "महीना चुनें",
    allInvoices: "सभी बिल",
    duplicateWarning: "यह बिल पहले ही जोड़ा जा चुका है। क्या आप इसे फिर भी जोड़ना चाहते हैं?",
    viewExisting: "मौजूदा बिल देखें",
    daysOverdue: "दिन अवधि पार",
  }
};

document.addEventListener('DOMContentLoaded', () => {
  setupLanguageToggle();
  setupSettings();
  setupUpload();
  setupModals();
  setupMonthPickers();
  loadDashboard();
});

function setupSettings() {
  const sel = document.getElementById('model-select');
  if (!sel) return;

  fetch('/api/settings')
    .then(r => r.json())
    .then(data => {
      if (data && data.model) sel.value = data.model;
    })
    .catch(() => {});

  sel.addEventListener('change', () => {
    fetch('/api/settings', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ model: sel.value })
    })
    .then(r => r.json())
    .then(data => {
      alert(`Active model switched to ${data.model}`);
    })
    .catch(() => {});
  });
}

function setupLanguageToggle() {
  const btn = document.getElementById('lang-toggle');
  btn.addEventListener('click', () => {
    currentLanguage = currentLanguage === 'en' ? 'hi' : 'en';
    applyTranslations();
    renderAll();
  });
}

function applyTranslations() {
  const t = translations[currentLanguage];
  const setTxt = (id, txt) => {
    const el = document.getElementById(id);
    if (el) el.textContent = txt;
  };

  setTxt('t-brand', t.brand);
  setTxt('t-sub', t.sub);
  setTxt('t-monthly-spend', t.monthlySpend);
  setTxt('t-open-alerts', t.openAlerts);
  setTxt('t-price-spikes', t.priceSpikes);
  setTxt('t-total-dues', t.supplierDues);
  setTxt('t-needs-review', t.needsReview);
  setTxt('t-review-sub', t.reviewSub);
  setTxt('t-upload-title', t.uploadTitle);
  setTxt('t-upload-sub', t.uploadSub);
  setTxt('t-upload-btn', t.uploadBtn);
  setTxt('t-alerts-title', t.alertsTitle);
  setTxt('t-dues-title', t.duesTitle);
  setTxt('t-ledger-title', t.ledgerTitle);
  setTxt('t-reading-bill', t.readingBill);
  setTxt('t-reading-sub', t.readingSub);
  setTxt('t-qr-nav-btn', t.qrNavBtn);
  setTxt('t-qr-title', t.qrTitle);
  setTxt('t-qr-sub', t.qrSub);
  setTxt('t-btn-ca-pack', t.btnCaPack);
  setTxt('t-btn-tally', t.btnTally);
  setTxt('th-supplier', t.thSupplier);
  setTxt('th-total-billed', t.thTotalBilled);
  setTxt('th-total-paid', t.thTotalPaid);
  setTxt('th-balance-due', t.thBalanceDue);
  setTxt('th-earliest-due', t.thEarliestDue);
  setTxt('th-status', t.thStatus);
  setTxt('th-action', t.thAction);
  setTxt('th-inv-no', t.thInvNo);
  setTxt('th-inv-date', t.thInvDate);
  setTxt('th-inv-supplier', t.thInvSupplier);
  setTxt('th-inv-total', t.thInvTotal);
  setTxt('th-inv-val', t.thInvVal);
  setTxt('th-inv-payment', t.thInvPayment);
  setTxt('th-inv-actions', t.thInvActions);
  setTxt('t-zoom-hint', t.zoomHint);
  setTxt('t-header-details', t.headerDetails);
  setTxt('lbl-supplier', t.lblSupplier);
  setTxt('lbl-gstin', t.lblGstin);
  setTxt('lbl-inv-num', t.lblInvNum);
  setTxt('lbl-inv-date', t.lblInvDate);
  setTxt('t-items-details', t.itemsDetails);
  setTxt('t-tax-totals', t.taxTotals);
  setTxt('lbl-subtotal', t.lblSubtotal);
  setTxt('lbl-cgst', t.lblCgst);
  setTxt('lbl-sgst', t.lblSgst);
  setTxt('lbl-igst', t.lblIgst);
  setTxt('lbl-roundoff', t.lblRoundoff);
  setTxt('lbl-grandtotal', t.lblGrandtotal);
  setTxt('lbl-total-summary', t.lblTotalSummary);
  setTxt('btn-cancel-review', t.btnCancelReview);
  setTxt('t-pay-title', t.modalPayTitle);
  setTxt('lbl-pay-supplier', t.lblPaySupplier);
  setTxt('lbl-pay-billnum', t.lblPayBillnum);
  setTxt('lbl-pay-balance', t.lblPayBalance);
  setTxt('lbl-pay-amount', t.lblPayAmount);
  setTxt('lbl-pay-date', t.lblPayDate);
  setTxt('lbl-pay-mode', t.lblPayMode);
  setTxt('t-save-payment-btn', t.btnSavePayment);
}

function setupMonthPickers() {
  const spendPicker = document.getElementById('spend-month-picker');
  const exportPicker = document.getElementById('export-month');
  const ledgerFilter = document.getElementById('ledger-month-filter');
  const payDateInput = document.getElementById('pay-date-input');

  const now = new Date();
  const currentMonthStr = `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}`;
  selectedSpendMonth = currentMonthStr;

  if (spendPicker) {
    spendPicker.value = currentMonthStr;
    spendPicker.addEventListener('change', (e) => {
      selectedSpendMonth = e.target.value;
      calculateMetrics(cachedBills);
    });
  }

  if (exportPicker) {
    exportPicker.value = currentMonthStr;
    exportPicker.addEventListener('change', (e) => {
      const val = e.target.value;
      document.getElementById('ca-pack-btn').href = `/api/export/ca?month=${val}`;
      document.getElementById('tally-btn').href = `/api/export/tally?month=${val}`;
    });
  }

  if (ledgerFilter) {
    ledgerFilter.addEventListener('change', () => {
      renderBills(cachedBills);
    });
  }

  if (payDateInput) {
    payDateInput.value = now.toISOString().split('T')[0];
  }
}

function setupUpload() {
  const dropZone = document.getElementById('drop-zone');
  const fileInput = document.getElementById('file-input');

  dropZone.addEventListener('dragover', (e) => {
    e.preventDefault();
    dropZone.classList.add('dragover');
  });

  dropZone.addEventListener('dragleave', () => {
    dropZone.classList.remove('dragover');
  });

  dropZone.addEventListener('drop', (e) => {
    e.preventDefault();
    dropZone.classList.remove('dragover');
    if (e.dataTransfer.files.length > 0) uploadFile(e.dataTransfer.files[0]);
  });

  fileInput.addEventListener('change', () => {
    if (fileInput.files.length > 0) uploadFile(fileInput.files[0]);
  });
}

function uploadFile(file, confirmDuplicate = false) {
  const loading = document.getElementById('loading-overlay');
  loading.style.display = 'flex';

  const formData = new FormData();
  formData.append('bill', file);
  if (confirmDuplicate) {
    formData.append('confirm_duplicate', 'true');
  }

  fetch('/api/upload', { method: 'POST', body: formData })
    .then(async res => {
      const data = await res.json();
      if (!res.ok) {
        throw new Error(data.error || "Upload processing error");
      }
      return data;
    })
    .then(data => {
      loading.style.display = 'none';

      if (data.duplicate) {
        const t = translations[currentLanguage];
        const msg = `${data.duplicate_message || t.duplicateWarning}\n\nInvoice: ${data.existing_bill.invoice_number || 'N/A'}\nSupplier: ${data.existing_bill.vendor_name || 'N/A'}\nDate: ${data.existing_bill.invoice_date ? data.existing_bill.invoice_date.split('T')[0] : 'N/A'}\n\nClick OK to upload anyway, or Cancel to view the existing bill.`;
        if (confirm(msg)) {
          uploadFile(file, true);
        } else {
          viewBillDetails(data.existing_bill.id);
        }
        return;
      }

      loadDashboard();
      if (data.validation_status === 'needs_review') {
        openReviewModal(data.bill);
      }
    })
    .catch(err => {
      loading.style.display = 'none';
      alert("Error uploading bill: " + err.message);
    });
}

function setupModals() {
  document.getElementById('qr-btn').addEventListener('click', () => {
    const modal = document.getElementById('qr-modal');
    const urlEl = document.getElementById('qr-lan-url');
    fetch('/api/lan-url')
      .then(r => r.json())
      .then(data => {
        if (data.lan_url) {
          urlEl.textContent = data.lan_url;
        } else {
          urlEl.textContent = "No active LAN IPv4 detected. Please connect to Wi-Fi/LAN.";
        }
      })
      .catch(() => {
        urlEl.textContent = `http://${window.location.host}`;
      });
    modal.style.display = 'flex';
  });

  document.getElementById('review-approve-btn').addEventListener('click', saveReviewBill);
  document.getElementById('confirm-pay-btn').addEventListener('click', savePayment);
}

function toggleZoomImage() {
  const container = document.getElementById('zoom-container');
  container.classList.toggle('zoomed');
}

function loadDashboard() {
  Promise.all([
    fetch('/api/bills').then(r => r.json()),
    fetch('/api/alerts').then(r => r.json()),
    fetch('/api/dues').then(r => r.json())
  ]).then(([bills, alerts, dues]) => {
    cachedBills = bills || [];
    cachedAlerts = alerts || [];
    cachedDues = dues || [];

    // Populate ledger month filter options dynamically
    const ledgerFilter = document.getElementById('ledger-month-filter');
    if (ledgerFilter) {
      const currentSelected = ledgerFilter.value;
      const monthsSet = new Set();
      cachedBills.forEach(b => {
        if (b.invoice_date) {
          monthsSet.add(b.invoice_date.split('T')[0].substring(0, 7));
        }
      });
      const sortedMonths = Array.from(monthsSet).sort().reverse();
      const t = translations[currentLanguage];
      ledgerFilter.innerHTML = `<option value="ALL">${t.allInvoices}</option>` +
        sortedMonths.map(m => `<option value="${m}">${m}</option>`).join('');
      if (currentSelected && (currentSelected === "ALL" || sortedMonths.includes(currentSelected))) {
        ledgerFilter.value = currentSelected;
      }
    }

    // Auto-align spend month picker with the latest bill's month if bills exist
    if (cachedBills.length > 0) {
      const latestDateStr = cachedBills[0].invoice_date.split('T')[0];
      const latestMonth = latestDateStr.substring(0, 7);
      const spendPicker = document.getElementById('spend-month-picker');
      const exportPicker = document.getElementById('export-month');
      if (spendPicker && !spendPicker.dataset.userChanged) {
        selectedSpendMonth = latestMonth;
        spendPicker.value = latestMonth;
      }
      if (exportPicker) {
        exportPicker.value = selectedSpendMonth;
        document.getElementById('ca-pack-btn').href = `/api/export/ca?month=${selectedSpendMonth}`;
        document.getElementById('tally-btn').href = `/api/export/tally?month=${selectedSpendMonth}`;
      }
    }

    renderAll();
  }).catch(() => {});
}

function renderAll() {
  applyTranslations();
  calculateMetrics(cachedBills);
  renderAlerts(cachedAlerts);
  renderDues(cachedDues);
  renderBills(cachedBills);
}

function calculateMetrics(bills) {
  const t = translations[currentLanguage];
  let monthlySpend = 0;
  let monthBillsCount = 0;
  let reviewCount = 0;

  bills.forEach(b => {
    if (b.validation_status !== 'rejected') {
      const invDateStr = b.invoice_date.split('T')[0];
      const invMonth = invDateStr.substring(0, 7);
      if (selectedSpendMonth === "" || invMonth === selectedSpendMonth) {
        monthlySpend += b.total;
        monthBillsCount++;
      }
    }
    if (b.validation_status === 'needs_review') {
      reviewCount++;
    }
  });

  document.getElementById('val-monthly-spend').textContent = formatCurrency(monthlySpend);
  document.getElementById('val-bills-count').textContent = `${monthBillsCount} ${t.billsProcessed}`;
  document.getElementById('val-review-count').textContent = reviewCount;
}

function renderAlerts(alerts) {
  const t = translations[currentLanguage];
  const container = document.getElementById('alerts-container');
  const badge = document.getElementById('alerts-badge');
  const countVal = document.getElementById('val-alerts-count');

  badge.textContent = `${alerts.length} ${t.activeBadge}`;
  countVal.textContent = alerts.length;

  if (alerts.length === 0) {
    container.innerHTML = `<div style="color: var(--text-muted); font-size: 0.9rem; padding: 12px 0;">${t.noAlerts}</div>`;
    return;
  }

  container.innerHTML = alerts.map(a => {
    const isCheaper = a.type === 'cheaper_elsewhere';
    const tagClass = isCheaper ? 'cheaper' : '';
    const icon = isCheaper ? '💡' : '⚠️';
    
    const currInv = a.current_invoice_number || `Bill #${a.current_bill_id}`;
    const currLink = a.current_image_path ? `/bills/${a.current_image_path}` : '#';
    const refInv = a.reference_invoice_number || `Bill #${a.reference_bill_id}`;
    const refLink = a.reference_image_path ? `/bills/${a.reference_image_path}` : '#';

    return `
      <div class="alert-item ${tagClass}">
        <div>
          <div class="alert-desc">${icon} ${a.message}</div>
          <div class="evidence-badge">
            ${t.evidenceLabel}: <a href="${currLink}" target="_blank" style="color:var(--accent-gold); text-decoration:underline;">${currInv}</a> (${a.current_bill_date.split('T')[0]}) vs <a href="${refLink}" target="_blank" style="color:var(--accent-gold); text-decoration:underline;">${refInv}</a> (${a.reference_bill_date.split('T')[0]})
          </div>
        </div>
      </div>
    `;
  }).join('');
}

function renderDues(dues) {
  const t = translations[currentLanguage];
  const tbody = document.getElementById('dues-tbody');
  let totalDue = 0;
  let overdueCount = 0;

  if (dues.length === 0) {
    tbody.innerHTML = `<tr><td colspan="7" style="text-align:center; color:var(--text-muted)">${t.noDues}</td></tr>`;
    document.getElementById('val-dues-amount').textContent = formatCurrency(0);
    document.getElementById('val-overdue-sub').textContent = `0 ${t.overdueBills}`;
    return;
  }

  tbody.innerHTML = dues.map(d => {
    totalDue += d.balance_due;
    if (d.is_overdue) overdueCount += d.overdue_bills;

    const overdueLabel = d.overdue_days > 0 ? ` (${d.overdue_days} ${t.daysOverdue})` : ` (${d.overdue_bills})`;
    const statusBadge = d.is_overdue 
      ? `<span class="badge badge-danger">${t.statusOverdue}${overdueLabel}</span>`
      : `<span class="badge badge-success">${t.statusOk}</span>`;

    const dueDateStr = d.earliest_due ? d.earliest_due.split('T')[0] : 'N/A';

    return `
      <tr>
        <td style="font-weight:600;">${d.supplier_name}</td>
        <td>${formatCurrency(d.total_billed)}</td>
        <td>${formatCurrency(d.total_paid)}</td>
        <td style="color:${d.balance_due > 0 ? 'var(--accent-gold)' : 'var(--accent-green)'}; font-weight:700;">${formatCurrency(d.balance_due)}</td>
        <td>${dueDateStr}</td>
        <td>${statusBadge}</td>
        <td>
          ${d.balance_due > 0 ? `<button class="btn btn-secondary" style="padding:4px 10px; font-size:0.8rem;" onclick="openPayModal('${d.supplier_name}', ${d.balance_due})">${t.btnPay}</button>` : '—'}
        </td>
      </tr>
    `;
  }).join('');

  document.getElementById('val-dues-amount').textContent = formatCurrency(totalDue);
  document.getElementById('val-overdue-sub').textContent = `${overdueCount} ${t.overdueBills}`;
}

function renderBills(bills) {
  const t = translations[currentLanguage];
  const tbody = document.getElementById('bills-tbody');
  const ledgerFilter = document.getElementById('ledger-month-filter');
  const filterVal = ledgerFilter ? ledgerFilter.value : 'ALL';

  let filtered = bills;
  if (filterVal && filterVal !== 'ALL') {
    filtered = bills.filter(b => b.invoice_date && b.invoice_date.split('T')[0].startsWith(filterVal));
  }

  if (filtered.length === 0) {
    tbody.innerHTML = `<tr><td colspan="7" style="text-align:center; color:var(--text-muted)">${t.noBills}</td></tr>`;
    return;
  }

  tbody.innerHTML = filtered.map(b => {
    const valBadge = b.validation_status === 'verified'
      ? `<span class="badge badge-success">${t.statusVerified}</span>`
      : b.validation_status === 'approved'
      ? `<span class="badge badge-success">${t.statusApproved}</span>`
      : `<span class="badge badge-danger">${t.statusNeedsReview}</span>`;

    const payBadge = b.payment_status === 'paid'
      ? `<span class="badge badge-success">${t.statusPaid}</span>`
      : b.payment_status === 'partially_paid'
      ? `<span class="badge badge-warning">${t.statusPartial}</span>`
      : `<span class="badge badge-danger">${t.statusUnpaid}</span>`;

    return `
      <tr>
        <td style="font-weight:600;"><a href="/bills/${b.image_path}" target="_blank" style="color:var(--accent-gold); text-decoration:none;">${b.invoice_number || 'B-'+b.id}</a></td>
        <td>${b.invoice_date.split('T')[0]}</td>
        <td>${b.vendor_name}</td>
        <td style="font-weight:700;">${formatCurrency(b.total)}</td>
        <td>${valBadge}</td>
        <td>${payBadge}</td>
        <td>
          <button class="btn btn-secondary" style="padding:4px 8px; font-size:0.8rem;" onclick="viewBillDetails(${b.id})">${t.btnEdit}</button>
          ${b.payment_status !== 'paid' ? `<button class="btn btn-secondary" style="padding:4px 8px; font-size:0.8rem;" onclick="openDirectPay(${b.id}, '${b.vendor_name}', '${b.invoice_number}', ${b.total - b.amount_paid})">${t.btnRecordPay}</button>` : ''}
        </td>
      </tr>
    `;
  }).join('');
}

function viewBillDetails(billId) {
  fetch('/api/bills')
    .then(r => r.json())
    .then(bills => {
      const b = bills.find(x => x.id === billId);
      if (b) openReviewModal(b);
    });
}

// Requirement 5: Review modal showing and allowing editing of supplier, GSTIN, invoice number, date, subtotal, each tax, round-off, total, HSN, and GST rate per line. Highlight failing fields with plain language in English & Hindi.
function openReviewModal(bill) {
  activeReviewBill = bill;
  const t = translations[currentLanguage];
  const modal = document.getElementById('review-modal');

  // Set bill image
  document.getElementById('review-bill-img').src = `/bills/${bill.image_path}`;
  document.getElementById('zoom-container').classList.remove('zoomed');

  // Fill Header Inputs
  document.getElementById('edit-vendor-name').value = bill.vendor_name || '';
  document.getElementById('edit-vendor-gstin').value = bill.vendor_gstin || '';
  document.getElementById('edit-invoice-number').value = bill.invoice_number || '';
  document.getElementById('edit-invoice-date').value = bill.invoice_date ? bill.invoice_date.split('T')[0] : '';

  // Fill Taxes & Totals
  document.getElementById('edit-subtotal').value = (bill.subtotal || 0).toFixed(2);
  document.getElementById('edit-cgst').value = (bill.cgst || 0).toFixed(2);
  document.getElementById('edit-sgst').value = (bill.sgst || 0).toFixed(2);
  document.getElementById('edit-igst').value = (bill.igst || 0).toFixed(2);
  document.getElementById('edit-roundoff').value = (bill.round_off || 0).toFixed(2);
  document.getElementById('edit-total').value = (bill.total || 0).toFixed(2);
  document.getElementById('review-bill-total-display').textContent = formatCurrency(bill.total || 0);

  // Clear previous error highlights
  document.querySelectorAll('.form-group').forEach(el => el.classList.remove('field-error'));

  // Render Line Items Table (with HSN and GST rate editable per line)
  const tbody = document.getElementById('review-items-tbody');
  tbody.innerHTML = (bill.line_items || []).map((it, idx) => `
    <tr>
      <td><input type="text" value="${it.description}" id="item-desc-${idx}" class="form-input" style="width:100%; text-align:left;"></td>
      <td><input type="text" value="${it.hsn_code || ''}" id="item-hsn-${idx}" class="form-input" style="width:70px;"></td>
      <td><input type="text" value="${it.pack_size || ''}" id="item-pack-${idx}" class="form-input" style="width:75px;"></td>
      <td><input type="number" step="0.1" value="${it.quantity}" id="item-qty-${idx}" class="form-input" style="width:65px;" oninput="recalculateLineItem(${idx})"></td>
      <td><input type="number" step="0.01" value="${it.unit_price}" id="item-rate-${idx}" class="form-input" style="width:75px;" oninput="recalculateLineItem(${idx})"></td>
      <td><input type="number" step="0.5" value="${it.gst_rate || 0}" id="item-gstrate-${idx}" class="form-input" style="width:60px;"></td>
      <td><span id="item-taxable-${idx}" style="font-weight:600;">${formatCurrency(it.taxable_value || 0)}</span></td>
    </tr>
  `).join('');

  // Plain-Language Error Highlighting & Banner
  const errorsDiv = document.getElementById('review-validation-errors');
  const hasErrors = bill.validation_errors && bill.validation_errors.length > 0;
  const approveBtn = document.getElementById('review-approve-btn');

  if (hasErrors) {
    approveBtn.textContent = t.btnApproveAnyway;
    approveBtn.classList.remove('btn-primary');
    approveBtn.classList.add('btn-secondary');
    approveBtn.style.color = '#fbbf24';
    approveBtn.style.borderColor = '#fbbf24';

    // Highlight failing fields
    const bannerItems = [];
    bill.validation_errors.forEach(err => {
      let fld = "";
      let eng = err;
      let hnd = "";

      if (err.includes("GSTIN")) {
        fld = "vendor_gstin";
        eng = "Supplier GSTIN is invalid or checksum digit does not match.";
        hnd = "सप्लायर का GSTIN अमान्य है या चेक-डिजिट मेल नहीं खा रहा है।";
      } else if (err.includes("Line items taxable total") || err.includes("subtotal")) {
        fld = "subtotal";
        eng = "Line items taxable sum does not match the subtotal.";
        hnd = "सामानों का कुल जोड़ सब-टोटल से मेल नहीं खा रहा है।";
      } else if (err.includes("Grand Total") || err.includes("does not match Grand Total")) {
        fld = "total";
        eng = "Subtotal plus taxes does not equal the printed Grand Total.";
        hnd = "सब-टोटल और टैक्स जोड़कर ग्रैंड टोटल के बराबर नहीं हो रहा है।";
      } else if (err.includes("Intra-state tax mismatch") || err.includes("CGST")) {
        fld = "cgst";
        eng = "CGST and SGST must be equal on intra-state invoices.";
        hnd = "लोकल बिलों पर CGST और SGST बराबर होना आवश्यक है।";
      } else if (err.includes("confidence")) {
        fld = "confidence";
        eng = "Bill was blurry, faded, or handwritten. Please verify values.";
        hnd = "बिल धुंधला या हाथ से लिखा हुआ है। कृपया आंकड़े जांच लें।";
      }

      if (fld && document.getElementById(`group-${fld}`)) {
        document.getElementById(`group-${fld}`).classList.add('field-error');
      }

      bannerItems.push(`
        <div class="validation-banner-item">
          <div>⚠️ <b>${eng}</b></div>
          <div class="validation-hindi">${hnd}</div>
        </div>
      `);
    });

    errorsDiv.innerHTML = bannerItems.join('');
    errorsDiv.style.display = 'block';
  } else {
    approveBtn.textContent = t.btnApproveSave;
    approveBtn.classList.remove('btn-secondary');
    approveBtn.classList.add('btn-primary');
    approveBtn.style.color = '#0f172a';
    errorsDiv.style.display = 'none';
  }

  modal.style.display = 'flex';
}

function recalculateLineItem(idx) {
  const qty = parseFloat(document.getElementById(`item-qty-${idx}`).value) || 0;
  const rate = parseFloat(document.getElementById(`item-rate-${idx}`).value) || 0;
  const taxable = qty * rate;
  document.getElementById(`item-taxable-${idx}`).textContent = formatCurrency(taxable);

  // Recalculate subtotal
  let sum = 0;
  const count = (activeReviewBill.line_items || []).length;
  for (let i = 0; i < count; i++) {
    const q = parseFloat(document.getElementById(`item-qty-${i}`).value) || 0;
    const r = parseFloat(document.getElementById(`item-rate-${i}`).value) || 0;
    sum += q * r;
  }
  document.getElementById('edit-subtotal').value = sum.toFixed(2);
  recalculateGrandTotal();
}

function recalculateGrandTotal() {
  const subtotal = parseFloat(document.getElementById('edit-subtotal').value) || 0;
  const cgst = parseFloat(document.getElementById('edit-cgst').value) || 0;
  const sgst = parseFloat(document.getElementById('edit-sgst').value) || 0;
  const igst = parseFloat(document.getElementById('edit-igst').value) || 0;
  const roundoff = parseFloat(document.getElementById('edit-roundoff').value) || 0;

  const total = subtotal + cgst + sgst + igst + roundoff;
  document.getElementById('edit-total').value = total.toFixed(2);
  document.getElementById('review-bill-total-display').textContent = formatCurrency(total);
}

function saveReviewBill() {
  if (!activeReviewBill) return;
  const t = translations[currentLanguage];

  const hasRemainingErrors = activeReviewBill.validation_errors && activeReviewBill.validation_errors.length > 0;
  if (hasRemainingErrors) {
    if (!confirm(t.confirmApproveRemaining)) {
      return;
    }
  }

  const items = (activeReviewBill.line_items || []).map((it, idx) => {
    const qty = parseFloat(document.getElementById(`item-qty-${idx}`).value) || 0;
    const rate = parseFloat(document.getElementById(`item-rate-${idx}`).value) || 0;
    return {
      description: document.getElementById(`item-desc-${idx}`).value,
      hsn_code: document.getElementById(`item-hsn-${idx}`).value,
      pack_size: document.getElementById(`item-pack-${idx}`).value,
      quantity: qty,
      unit_price: rate,
      taxable_value: qty * rate,
      gst_rate: parseFloat(document.getElementById(`item-gstrate-${idx}`).value) || 0,
      unit: it.unit
    };
  });

  const payload = {
    bill_id: activeReviewBill.id,
    status: 'approved',
    vendor_name: document.getElementById('edit-vendor-name').value,
    vendor_gstin: document.getElementById('edit-vendor-gstin').value,
    invoice_number: document.getElementById('edit-invoice-number').value,
    invoice_date: document.getElementById('edit-invoice-date').value,
    subtotal: parseFloat(document.getElementById('edit-subtotal').value) || 0,
    cgst: parseFloat(document.getElementById('edit-cgst').value) || 0,
    sgst: parseFloat(document.getElementById('edit-sgst').value) || 0,
    igst: parseFloat(document.getElementById('edit-igst').value) || 0,
    round_off: parseFloat(document.getElementById('edit-roundoff').value) || 0,
    total: parseFloat(document.getElementById('edit-total').value) || 0,
    line_items: items
  };

  fetch('/api/bills/review', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload)
  })
  .then(r => r.json())
  .then(() => {
    document.getElementById('review-modal').style.display = 'none';
    loadDashboard();
  })
  .catch(err => alert("Error saving bill: " + err.message));
}

// Requirement 6: Payment modal showing supplier, bill number, remaining balance, date, and payment mode (cash/UPI/bank)
function openDirectPay(billId, supplier, invNumber, balance) {
  activePayBill = { id: billId, supplier: supplier, invNumber: invNumber, balance: balance };
  document.getElementById('pay-supplier-name').textContent = supplier || '—';
  document.getElementById('pay-bill-number').textContent = invNumber || ('Bill #' + billId);
  document.getElementById('pay-remaining-balance').textContent = formatCurrency(balance);
  document.getElementById('pay-amount-input').value = balance.toFixed(2);
  document.getElementById('pay-date-input').value = new Date().toISOString().split('T')[0];
  document.getElementById('pay-modal').style.display = 'flex';
}

function openPayModal(supplier, totalBalance) {
  const b = cachedBills.find(x => x.vendor_name === supplier && x.payment_status !== 'paid');
  if (b) {
    openDirectPay(b.id, b.vendor_name, b.invoice_number, b.total - b.amount_paid);
  } else {
    openDirectPay(0, supplier, "All Outstanding", totalBalance);
  }
}

function savePayment() {
  if (!activePayBill) return;
  const amount = parseFloat(document.getElementById('pay-amount-input').value);
  const paymentDate = document.getElementById('pay-date-input').value;
  const paymentMode = document.getElementById('pay-mode-select').value;

  if (!amount || amount <= 0) {
    alert("Please enter a valid payment amount");
    return;
  }

  fetch('/api/bills/pay', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      bill_id: activePayBill.id,
      amount: amount,
      payment_date: paymentDate,
      payment_mode: paymentMode
    })
  })
  .then(r => r.json())
  .then(() => {
    document.getElementById('pay-modal').style.display = 'none';
    loadDashboard();
  })
  .catch(err => alert("Error recording payment: " + err.message));
}
