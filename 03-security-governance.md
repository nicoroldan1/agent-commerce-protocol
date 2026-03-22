# ANS — Security, Governance & Risk

**Date:** 2026-02-22

---

## 1) Principles

- **Secure by default**: risky actions require explicit approval
- **Least privilege**: minimal scopes per agent
- **Immutable audit trail**: logs are queryable and exportable
- **Defense in depth**: auth + policy + budget + anomaly detection
- **Reversibility**: rollback and compensating transactions

---

## 2) Risk Matrix by Action (examples)

| Action | Risk | Default | Mitigation |
|---|---:|---|---|
| Edit description | Low | ALLOW | input validation |
| Upload images | Low | ALLOW | antivirus/scan + size limits |
| Adjust inventory | Medium | ALLOW with limits | per-SKU limits + rate limit |
| Publish product | Medium | APPROVAL | checklist + preview |
| Change price | High | APPROVAL or limited ALLOW | ±X% bands, alerts |
| Issue refund | High | APPROVAL | amount cap + double approval |
| Ad spend | High | APPROVAL | budget + whitelists |

---

## 3) Approvals (Human-in-the-loop)

### 3.1 Types
- **Manual**: human approves via UI
- **Policy-auto**: rules auto-approve if conditions are met (e.g., delta < 2%)
- **Multi-sig**: requires 2 approvals (high-value refunds)

### 3.2 Minimum data in an approval request
- actor
- action and payload
- estimated impact (e.g., margin delta)
- agent's recommendation
- evidence (inputs / links)

---

## 4) Budgets and Rate Limits

- Budget scoped by:
  - store
  - agent
  - connector (ads / payments)
- Rate limits per endpoint and per action type
- Alerts triggered by:
  - spikes (3× baseline)
  - loops (same action repeated in a short window)
  - repeated connector failures

---

## 5) Audit Trail and Forensics

- Hash-chained log entries (optional, for tamper-evidence)
- Export to WORM storage for Enterprise tier
- Correlation IDs end-to-end across all services

---

## 6) Compliance and Privacy

- PII: encrypted at rest, masked in logs
- Configurable retention (e.g., 30 / 90 / 365 days)
- Access control via RBAC (role-based)
