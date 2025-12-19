# Pricing Flexibility Tasks

| # | Task | Owner | Notes |
| - | ---- | ----- | ----- |
| 1 | Add `pricing_tiers_json` column + Go structs/loader support seeded from existing `price_input`/`price_output` and `price_image*_cents` metadata | Backend | Includes DB migration, sqlc regen, config parser updates, and runtime catalog merge logic (see PLAN.md). |
| 2 | Implement pricing helper + usage metrics plumbing so executors/usage logger resolve tiered costs (LLM brackets, per-image overrides, audio minute/character billing) | Backend | Capture audio duration + TTS char counts, expose override hooks for batch worker + SSE; keep scalar fallback until all models converted. |
| 3 | Extend Admin API/UI + bootstrap YAML to edit/display tier definitions and expose active tier in usage endpoints | Full-stack | Model editor requires tier table, validations, and adoption cues; API responses should return normalized tiers. |
| 4 | Update docs (`docs/runtime/config.md`, `deploy/router.example.yaml`, changelog) and add regression tests covering tier selection + reporting | Docs/QA | Include migration runbook and usage dashboard expectations. |
