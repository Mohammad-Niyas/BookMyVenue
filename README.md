# BookMyVenue

# 🏟️ BookMyVenue

A robust, multi-tenant venue and slot booking backend service built in **Go (Golang)** using **Clean Architecture**. BookMyVenue handles the end-to-end operational lifecycle: venue owner onboarding, AWS S3 image uploads via presigned URLs, schedule generation, Redis-based distributed concurrency holds, and multi-stage payment settlements with cryptographic verification.

---

## 📌 Executive Summary

Online booking platforms for physical facilities (turfs, sports courts, banquet halls) face distinct backend engineering challenges:
1. **Concurrency Conflicts:** High-demand time slots risk double-booking when multiple users checkout simultaneously.
2. **Operational Diversity:** Hourly venues (turfs, badminton courts) require dynamic time-block scheduling, whereas full-day venues (banquet halls, auditoriums) require exclusive daily bookings.
3. **Financial State Consistency:** Managing multi-stage payments (30% advance deposit at booking vs. 70% balance settled before the event) requires strict transaction integrity and immutable audit logging.

BookMyVenue solves these problems with a decoupled, layered Go architecture backed by **PostgreSQL** transactions and sub-millisecond **Redis distributed locks**.

---

## 🛠️ Tech Stack & Dependencies

- **Language:** Go (1.22+)
- **HTTP Routing & Middleware:** Gin Web Framework
- **Relational Database & ORM:** PostgreSQL 15+ with GORM
- **Caching & Distributed Locks:** Redis 7+ (`go-redis/v9`)
- **Cloud Object Storage:** AWS S3 (`aws-sdk-go-v2`) for direct-to-cloud presigned URL generation
- **Payment Processing & Cryptography:** Razorpay API with server-side `crypto/hmac` and `crypto/sha256` verification
- **Authentication:** JWT (JSON Web Tokens) with custom Role-Based Access Control (RBAC)

---


## 🏗️ Architecture & Project Structure
The project strictly follows **Clean Architecture (Ports and Adapters)**, ensuring business logic in the `service` layer remains completely decoupled from database queries (`repository`) and HTTP transports (`handler`).
```text
BookMyVenue/back-end/
├── cmd/
│   └── api/
│       └── main.go                 # Dependency injection, database migrations & server startup
├── config/
│   ├── config.go                   # Environment configuration loader
│   ├── database.go                 # PostgreSQL GORM connection pool
│   └── redis.go                    # Redis client initialization
├── internal/
│   ├── domain/                     # Core business entities & GORM models
│   │   ├── user.go
│   │   ├── venue.go
│   │   ├── space.go
│   │   ├── slot.go
│   │   ├── booking.go
│   │   ├── payment.go
│   │   └── audit_log.go
│   ├── handler/                    # HTTP transport layer, request parsing & validation
│   │   ├── auth_handler.go
│   │   ├── venue_handler.go
│   │   ├── booking_handler.go
│   │   ├── payment_handler.go
│   │   └── middleware.go           # JWT Auth, Role Guard, and Redis Rate Limiter
│   ├── repository/                 # Data access layer (Interfaces & GORM implementations)
│   │   ├── user_repository.go
│   │   ├── venue_repository.go
│   │   ├── space_repository.go
│   │   ├── booking_repository.go
│   │   └── payment_repository.go
│   ├── router/
│   │   └── router.go               # Route groups, middleware bindings & engine setup
│   └── service/                    # Pure business logic layer
│       ├── auth_service.go
│       ├── venue_service.go
│       ├── booking_service.go
│       └── payment_service.go
└── pkg/
    └── s3/
        └── s3.go                   # AWS S3 Presigned URL generator                 # AWS S3 Presigned URL generator

```
---

## ✨ Implemented & Working Features

### 1. 🔐 Multi-Tenant Authentication & RBAC
- Separate registration and login endpoints for **Users** and **Venue Owners**.
- Dedicated **Admin** authentication channel.
- Stateless **JWT-based authorization** enforced across protected routes.
- Distributed **Token Bucket / Sliding Window Rate Limiting** using Redis to protect against brute-force attacks.

### 2. 🏢 Venue & Space Management (Owner Module)
- **Multi-Tenant Venue Isolation:** Venue owners can only modify spaces and schedules they explicitly own.
- **Direct S3 Uploads via Presigned URLs:** Venue owners request time-limited (15-minute) presigned S3 URLs from the backend, uploading high-resolution venue photos directly to AWS S3 without consuming server memory or network bandwidth.
- **Dual Space Types:**
  - `hourly`: Turfs, courts, and conference spaces with variable hourly rates.
  - `daily`: Banquet halls and auditoriums with mandatory single-slot enforcement per calendar date.
- **Idempotent Schedule Replacement (`ReplaceSlots`):** Replaces non-booked slots for a specific date in an atomic database transaction while strictly preserving existing customer bookings.

### 3. 🛡️ Admin Moderation & Shadow Copies
- Platform moderation for all newly registered venues (`/approve` and `/reject`).
- **Shadow Copy Draft Pattern (`venue_edit_drafts`):** When an approved venue updates details, the live public listing remains active and bookable, while proposed updates are stored in a draft table until reviewed and approved by an Admin.

### 4. ⚡ High-Concurrency Booking Engine
- **Slot Hold via Redis `SET NX`:** When a user selects an available slot, a distributed lock (`hold:slot:<slot_id>`) is acquired in Redis for 10 minutes.
- **Race Condition Defense:** The payment order creation enforces a minimum 1-minute TTL check (`rdb.TTL >= 1 Minute`). If the hold is expiring, the order is rejected to prevent last-second phantom payments.
- **Advance Booking Guard:** Large daily venues (`capacity > 4`) enforce a mandatory 30-day advance booking notice.

### 5. 💳 Multi-Stage Payment Engine (Razorpay)
- **Dynamic Pricing & Installment Calculation:**
  - **Hourly Spaces & Small Rooms ($\le$ 4 Capacity):** 100% full payment charged immediately.
  - **Large Daily Venues ($>$ 4 Capacity):** 30% advance token deposit charged upon booking creation.
  - **Installment 2 (70% Balance):** When paying the remaining balance, the service dynamically computes `TotalAmount - AmountPaid` without redundant calculated columns.
  - **7-Day Pre-Event Cutoff:** Enforces a strict deadline preventing users from paying the remaining balance if less than 7 days remain before the event.
- **Server-Side Cryptographic Verification:** Recomputes `HMAC-SHA256(order_id + "|" + payment_id, secret)` in Go standard library (`crypto/hmac`) to guarantee payment authenticity.
- **Atomic 3-Table ACID Transaction:** Capturing payment updates `payments.status = "captured"`, `bookings.status = "confirmed"`, `slots.is_booked = true`, and appends an immutable entry to `payment_audit_logs` in a single PostgreSQL commit.
- **Automatic Lock Release:** Automatically deletes the Redis temporary lock once the slot is permanently booked in PostgreSQL.

---

## 📡 API Endpoint Reference

### Public Routes
| Method | Endpoint | Description |
|---|---|---|
| `POST` | `/api/auth/register/user` | Register customer account |
| `POST` | `/api/auth/register/owner` | Register venue owner account |
| `POST` | `/api/auth/login` | Login and receive JWT access token |
| `POST` | `/api/admin/auth/login` | Admin authentication |
| `GET` | `/api/venues` | Search venues by name, city, or type |
| `GET` | `/api/venues/:id` | View public venue details and spaces |
| `GET` | `/api/venues/spaces/:id/slots` | Query unbooked slots for a specific date |

### Protected Customer Routes (`Bearer <JWT>`, Role: `user`)
| Method | Endpoint | Description |
|---|---|---|
| `POST` | `/api/user/bookings` | Reserve a slot (acquires 10-min Redis hold) |
| `POST` | `/api/user/payments/order` | Generate payment order (30% advance or full) |
| `POST` | `/api/user/payments/verify` | Verify HMAC signature & confirm booking atomically |

### Protected Venue Owner Routes (`Bearer <JWT>`, Role: `owner`)
| Method | Endpoint | Description |
|---|---|---|
| `POST` | `/api/owner/venues` | Submit new venue listing for approval |
| `GET` | `/api/owner/venues` | List venues owned by authenticated owner |
| `GET` | `/api/owner/venues/:id` | Get venue details by ID |
| `PUT` | `/api/owner/venues/:id` | Update venue profile |
| `DELETE`| `/api/owner/venues/:id` | Delete venue listing |
| `PATCH`| `/api/owner/venues/:id/toggle` | Toggle venue active/inactive status |
| `GET` | `/api/owner/venues/presigned-url` | Generate AWS S3 presigned upload URL |
| `POST` | `/api/owner/venues/:id/spaces` | Create space (hourly/daily) under venue |
| `PUT` | `/api/owner/spaces/:id` | Update space properties and pricing |
| `DELETE`| `/api/owner/spaces/:id` | Delete space |
| `POST` | `/api/owner/spaces/:id/slots` | Idempotently generate daily slot schedule |

### Protected Admin Routes (`Bearer <JWT>`, Role: `admin`)
| Method | Endpoint | Description |
|---|---|---|
| `GET` | `/api/admin/dashboard` | Admin status check |
| `GET` | `/api/admin/venues/pending` | List pending venues awaiting moderation |
| `POST` | `/api/admin/venues/:id/approve` | Approve venue for public discovery |
| `POST` | `/api/admin/venues/:id/reject` | Reject venue listing |
| `GET` | `/api/admin/venues/drafts/pending`| List pending shadow-copy edit drafts |
| `POST` | `/api/admin/venues/drafts/:draft_id/approve` | Merge approved edit draft to live venue |
| `POST` | `/api/admin/venues/drafts/:draft_id/reject` | Reject venue edit draft |

---

## 💻 Local Setup & Installation

### 1. Prerequisites
- **Go:** `1.22+`
- **PostgreSQL:** `15+`
- **Redis:** `7+`
- **AWS Account:** S3 bucket configured for image uploads
- **Razorpay Account:** API Key ID & Secret

### 2. Clone & Environment Configuration
```bash
git clone https://github.com/<your-username>/BookMyVenue.git
cd BookMyVenue/back-end
```bash
git clone https://github.com/<your-username>/BookMyVenue.git
cd BookMyVenue/back-end
```

Create a `.env` file in the root directory:
```ini
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=your_password
DB_NAME=bookmyvenue
DB_SSLMODE=disable
SERVER_PORT=8080
JWT_SECRET=your_jwt_secret
REDIS_HOST=localhost
REDIS_PORT=6379
RAZORPAY_KEY_ID=your_key_id
RAZORPAY_KEY_SECRET=your_key_secret
```

### 3. Run the Server
```bash
go mod download
go run cmd/api/main.go
```