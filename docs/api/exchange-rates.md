# Exchange Rates Module - `/api/exchange-rates`

Currency exchange rates from RapidAPI. **All routes require a Bearer token.**

Results are stored in Valkey/Redis for 15 minutes when `VALKEY_URL` is enabled. If cache is disabled, the endpoint still works and fetches from the provider on each request.

## GET `/api/exchange-rates`

Fetch the exchange rate from one currency to another.

**Query**

| Param | Required | Description |
|-------|----------|-------------|
| `from` | Yes | 3-letter currency code, for example `USD` |
| `to` | Yes | 3-letter currency code, for example `IDR` |

Currency codes are normalized to uppercase. Inputs other than 3 alphabetic letters return `400`.

**Example**

```http
GET /api/exchange-rates?from=USD&to=IDR
Authorization: Bearer <access_token>
```

**Success - 200**

```json
{
  "success": true,
  "message": "Exchange rate fetched successfully",
  "data": {
    "from": "USD",
    "to": "IDR",
    "symbol": "USDIDR=X",
    "rate": 16250,
    "source": "RapidAPI",
    "cached": false,
    "fetchedAt": "2026-06-01T10:00:00Z"
  }
}
```

| Field | Type | Description |
|-------|------|-------------|
| `from` | string | Source currency |
| `to` | string | Target currency |
| `symbol` | string | Currency pair symbol used |
| `rate` | number | Value of 1 `from` in `to` |
| `source` | string | Data provider |
| `cached` | boolean | `true` when the response came from cache |
| `fetchedAt` | string | Time when the fetch/cache record was created, RFC3339 UTC |

**Provider Notes**

The service first tries the direct pair, for example `USDIDR=X`. If it is unavailable, the service tries the inverse pair, for example `IDRUSD=X`, then calculates `1 / rate`.

When `from` and `to` are the same, the response returns `rate = 1`.

**Errors**

| HTTP | Condition |
|------|-----------|
| 400 | `from` or `to` is not a 3-letter currency code |
| 500 | Provider failure or pair not found |
