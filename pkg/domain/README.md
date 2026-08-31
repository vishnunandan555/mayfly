# 🌐 MayFly Domain Package (`pkg/domain`)

[![Zero Dependencies](https://img.shields.io/badge/Dependencies-0%20(Pure%20Stdlib)-brightgreen)](../../STDLIB.md)
[![Architecture](https://img.shields.io/badge/Pattern-Domain%20Driven%20Design%20(DDD)-blue)](domain.go)


`pkg/domain` defines the core data contracts, entity models, value objects, and repository interfaces for the MayFly workspace ecosystem.

---

## 🏛️ Core Domain Models

* **`SecretName` & `Secret`:** Strongly-typed secret identifiers with POSIX-compliant naming validations (`^[a-zA-Z_][a-zA-Z0-9_]*$`).
* **`Project` & `ProjectVault`:** Workspace metadata entity mapping canonical paths to unique project IDs.
* **`AuditEvent` & `Action`:** Cryptographically bound audit trail entry types (`secret:injected`, `vault:unlocked`, `password:rotated`, etc.).
* **`ScanFinding` & `Severity`:** Static analysis diagnostic issue representation with line references and severity classifications.
* **`Version`:** Canonical single-source-of-truth version string (`0.0.4`).


