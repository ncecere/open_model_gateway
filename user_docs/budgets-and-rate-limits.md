# Budgets and Rate Limits

Budgets and rate limits protect your spend and throughput.

## How Budget Precedence Works

Budgets apply in this order:

1. Global defaults (Admin Settings).
2. Tenant overrides.
3. API key overrides.

The most restrictive limit wins. Tenant caps act as a ceiling; per-key limits
cannot exceed tenant limits.

## Set Budget Defaults

1. Open **Admin -> Settings**.
2. Set the default budget:
   - **Schedule**: monthly, weekly, or rolling.
   - **Limit**: USD cap.
   - **Warning threshold**: fraction of limit (0.0–1.0).
3. Save to apply defaults to new tenants and keys.

## Override a Tenant Budget

1. Open **Admin -> Tenants**.
2. Select a tenant and open **Budgets**.
3. Set the limit and warning threshold.
4. Save to apply immediately.

## Override an API Key Budget

1. Create or edit an API key.
2. Set a key-specific budget.
3. Save; the UI displays the effective ceiling.

## Rate Limits Explained

Rate limits follow the same precedence order:

- Global defaults.
- Tenant overrides.
- API key overrides.

Limit types:

- **RPM**: requests per minute.
- **TPM**: tokens per minute.
- **Parallel**: maximum concurrent requests.

## Budget Alerts

Alerts can be delivered by:

- Email (SMTP).
- Webhook.

Configure destinations in **Admin -> Settings**. Alerts fire when a warning
threshold is crossed and when the budget is exceeded.

## Common Scenarios

- **Lower a tenant ceiling** during an incident, then restore it later.
- **Give a single key more headroom** without increasing the tenant cap.
- **Disable alerts** by setting the warning threshold to 1.0.

## Troubleshooting

- **Unexpected 402**: budget exceeded; check key and tenant limits.
- **429 errors**: rate limit exceeded; verify RPM/TPM/parallel settings.
