# Payment Gateway

A payment gateway API that lets a merchant process a card payment (forwarding it to an acquiring bank) and retrieve a previously processed payment.

## Running it

```bash
docker-compose up -d
```

This builds and starts both containers:

| Service | Container | Port |
|---|---|---|
| Bank simulator | `bank_simulator` | `:8080` |
| Payment gateway | `payment_gateway` | `:8090` |

To stop everything:

```bash
docker-compose down
```

## API

| Endpoint | Description |
|---|---|
| `POST /api/payments` | Process a card payment |
| `GET /api/payments/{id}` | Retrieve a previously processed payment |
| `GET /swagger/index.html` | Swagger UI |

## Testing

```bash
go test ./... 
```

## Design decisions and assumptions

* **Three Payment Outcomes:**
    * Authorized / Declined: Result from contacting the bank. Stored in memory.
    * Rejected: Failed validation before calling the bank. Returns `400 Bad Request` with a list of invalid fields and is not stored.
* **Supported Currencies:** Limited to `GBP`, `USD`, and `EUR` as specified.
* **Positive Amount Only:** Payment amounts must be greater than zero.
* **Expiry Month Rule:** A card expiring in the current month is treated as expired (not in the future) and will be rejected.
* **Handling Bank Outages:** If the acquiring bank is unavailable (`503`), the gateway returns `502 Bad Gateway` and does not store the payment because the final outcome is unknown.
* **Security & Sensitive Data:**
    * Only the last 4 digits of the card are kept.
    * Full card numbers and CVVs are never logged or stored.
* **No Card Fingerprinting:** Not implemented as the task does not require card recognition across payments. In production, a one-way hash of the full card number would typically be stored alongside the masked digits to support fraud checks, velocity limits etc. without exposing the raw card number
* **No Idempotency Keys:** Not implemented as the spec does not define an idempotency key. Without one, a retried request is treated as a new payment and the customer could be charged twice. In production, merchants would supply an Idempotency-Key per payment attempt, which the gateway would use to detect retries and return the original result without calling the bank again.
* **In-Memory Storage:** Payments are saved in a `map` protected by `sync.RWMutex` to handle concurrent HTTP requests safely.
