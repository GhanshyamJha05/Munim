# Hackathon Post Notes: Munim (मुनीम) Powered by Gemma

## Key Narrative Angles
1. **Gemma 3 Quirks Handled Gracefully**:
   - No system role: all instructions sent in user turn.
   - Fixed vision resolution: vertical tiling with 15% overlap solves small-print loss on tall supermarket & distributor receipts.
   - Deterministic extraction: Temperature 0, Seed 42, and num_ctx 8192 explicitly configured.
2. **Zero Marginal Cost Advantage**: At ₹0 per bill, a shopkeeper scanning 20 bills/day saves ₹15,000–₹20,000 annually compared to cloud vision APIs.
3. **Privacy as a Business Moat**: Kirana owners protect negotiated margins by keeping invoice photos 100% on their own machines.

## Evaluation Snapshot
See [eval/RESULTS.md](../eval/RESULTS.md) for full metrics table across Gemma 3 sizes (4B, 12B, 27B) and tiling ablation.
