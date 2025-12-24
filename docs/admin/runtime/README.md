---
title: runtime hub
description: Link-rich admin runtime map
---
[**Open Model Gateway**](/) centralizes runtime references for admin operators.

---

#### Survey references
| Guide | Purpose |
| --- | --- |
| [config.md](config.md) | Describe every router.yaml block plus sample overrides. |
| [router-example.md](router-example.md) | Copy an annotated config that maps to new provider blocks. |
| [bootstrap.md](bootstrap.md) | Seed tenants, admins, budgets, and quotas safely. |
| [pricing.md](pricing.md) | Model tier metadata for chat, images, audio, and video. |
| [usage.md](usage.md) | Export usage and dispatch billing webhooks per tenant. |
| [batches.md](batches.md) | Align batch ingestion with OpenAI and Azure semantics. |
| [moderations.md](moderations.md) | Operate moderation aliases across real-time and batch flows. |
| [observability.md](observability.md) | Turn on metrics, OTEL tracing, and collector manifests. |

---

#### Apply workflows
Review `config.md` before touching workflow-specific guides so routing, budgets, and telemetry stay aligned.
Share this README with admins onboarding personal tenants so every runtime control maps back to the correct reference.

---

#### Mirror layout
Generate the site structure after copying these guides into your deployment repo.

```bash
# expected layout
docs/admin/runtime
├── README.md
├── batches.md
├── bootstrap.md
├── config.md
├── moderations.md
├── observability.md
├── pricing.md
├── router-example.md
└── usage.md
```

---

#### Research
- Migrated legacy runtime guides from the old `docs/runtime/*.md` location to this directory.
- Cross-checked admin/user request docs under `docs/admin/requests` for referencing style and latest endpoints.
