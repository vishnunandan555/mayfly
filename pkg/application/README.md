# ⚙️ MayFly Application Orchestration Service (`pkg/application`)

[![Zero Dependencies](https://img.shields.io/badge/Dependencies-0%20(Pure%20Stdlib)-brightgreen)](file:///home/vishnunandan555/Projects/mayfly/STDLIB.md)
[![Pattern](https://img.shields.io/badge/Architecture-Clean%20%2F%20Hexagonal-blue)](file:///home/vishnunandan555/Projects/mayfly/pkg/application/service.go)

`pkg/application` contains the primary business logic and orchestration service layer connecting storage vaults, in-memory execution, audit trails, and filesystem scanning.

---

## 🎯 Key Responsibilities

1. **Vault Lifecycle Orchestration:** Unlocking, locking, auto-lock timeout scheduling, password rotation, and encrypted JSON export/import backups.
2. **Project & Secret Management:** Managing multi-project secret namespaces with lock-guarded concurrency.
3. **Execution Pipeline:** Coordinating credential retrieval, in-memory execution injection, and audit logging.
4. **Static Security Scanning:** Orchestrating project-level directory analysis and `.mayflyignore` evaluations.
