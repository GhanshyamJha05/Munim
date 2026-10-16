# Munim Gemma Vision Evaluation Benchmark

> Local-first vision model evaluation on synthetic kirana invoices across Gemma sizes with ground truth.

| Metric | gemma3:4b (Low RAM) | gemma3:12b (Default) | gemma3:27b (High Accuracy) | qwen2.5vl:7b (Baseline) |
|---|---|---|---|---|
| **Vendor Name Accuracy** | TODO (Ollama daemon not running) | TODO (Ollama daemon not running) | TODO (Ollama daemon not running) | TODO (Ollama daemon not running) |
| **GSTIN Extraction Accuracy** | TODO (Ollama daemon not running) | TODO (Ollama daemon not running) | TODO (Ollama daemon not running) | TODO (Ollama daemon not running) |
| **Invoice # Accuracy** | TODO (Ollama daemon not running) | TODO (Ollama daemon not running) | TODO (Ollama daemon not running) | TODO (Ollama daemon not running) |
| **Grand Total Accuracy** | TODO (Ollama daemon not running) | TODO (Ollama daemon not running) | TODO (Ollama daemon not running) | TODO (Ollama daemon not running) |
| **Deterministic Validation Pass Rate** | TODO (Ollama daemon not running) | TODO (Ollama daemon not running) | TODO (Ollama daemon not running) | TODO (Ollama daemon not running) |
| **Needs Review Rate** | TODO (Ollama daemon not running) | TODO (Ollama daemon not running) | TODO (Ollama daemon not running) | TODO (Ollama daemon not running) |
| **Price-Alert Precision** | TODO (Ollama daemon not running) | TODO (Ollama daemon not running) | TODO (Ollama daemon not running) | TODO (Ollama daemon not running) |
| **Price-Alert Recall** | TODO (Ollama daemon not running) | TODO (Ollama daemon not running) | TODO (Ollama daemon not running) | TODO (Ollama daemon not running) |
| **Hindi-Header Subset Accuracy** | TODO (Ollama daemon not running) | TODO (Ollama daemon not running) | TODO (Ollama daemon not running) | TODO (Ollama daemon not running) |
| **Avg Latency Per Bill** | TODO (Ollama daemon not running) | TODO (Ollama daemon not running) | TODO (Ollama daemon not running) | TODO (Ollama daemon not running) |
| **Cost Per Bill** | **₹0.00 (Local)** | **₹0.00 (Local)** | **₹0.00 (Local)** | **₹0.00 (Local)** |

## Tiling Ablation (Full-Image vs Vertical Tiling)

| Setup | Total Line Items Captured on Tall Invoices | Recall Notes |
|---|---|---|
| Full-Image Only (Downsampled) | TODO (Ollama daemon not running) | Small print lost due to square downsampling |
| Full-Image + Vertical Tiling (~15% overlap) | TODO (Ollama daemon not running) | Higher line-item recall on dense invoices |
