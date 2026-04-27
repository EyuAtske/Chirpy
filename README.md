# Chirpy

Chirpy is a simple HTTP server written in Go that mimics a lightweight social platform where users can create and manage short posts (“chirps”). The project is designed as a hands-on way to learn backend development concepts such as authentication, database integration, and API design.

---

## 🚀 Features

* User registration and authentication
* Password hashing for secure credential storage
* JWT-based authentication (access + refresh tokens)
* Create, read, update and delete chirps
* Webhook handling (Polka integration)
* Metrics tracking for file server usage
* PostgreSQL database integration
* Database migrations using Goose
* Type-safe queries using SQLC

---

## 🛠️ Tech Stack

* **Language:** Go
* **Database:** PostgreSQL
* **Migrations:** Goose
* **Query Generation:** SQLC
* **Authentication:** JWT (JSON Web Tokens)
* **Environment Management:** godotenv

---

## 📦 Requirements

Make sure you have the following installed:

* Go (>= 1.20 recommended)
* PostgreSQL
* Goose
* SQLC

---

## ⚙️ Installation

### 1. Clone the repository

```bash
git clone https://github.com/EyuAtske/Chirpy.git
cd Chirpy
```

### 2. Initialize Go module

```bash
go mod init github.com/EyuAtske/Chirpy
```

> If the project already includes a `go.mod` file, you can skip this step.

### 3. Install dependencies

```bash
go mod tidy
```

### 4. Set up environment variables

Create a `.env` file in the root directory:

```env
DB_URL=postgres://username:password@localhost:5432/chirpy?sslmode=disable
SECRET_KEY=your_jwt_secret
POLKA_KEY=your_polka_key
PLATFORM=dev
```

---

## 🗄️ Database Setup

### Run migrations with Goose

```bash
goose -dir sql/schema postgres "$DB_URL" up
```

### Generate queries with SQLC

```bash
sqlc generate
```

---

## ▶️ Running the Server

```bash
go run main.go
```

Server will start on:

```
http://localhost:8080
```

---

## 📡 API Endpoints

### Health Check

```
GET /api/healthz
```

---

### Users

* `POST /api/users` → Create a new user
* `PUT /api/users` → Update user info

---

### Authentication

* `POST /api/login` → Login user
* `POST /api/refresh` → Refresh JWT token
* `POST /api/revoke` → Revoke refresh token

---

### Chirps

* `POST /api/chirps` → Create chirp
* `GET /api/chirps` → Get all chirps
* `GET /api/chirps/{chirpID}` → Get single chirp
* `DELETE /api/chirps/{chirpID}` → Delete chirp

---

### Admin

* `GET /admin/metrics` → View metrics
* `POST /admin/reset` → Reset metrics

---

### Webhooks

* `POST /api/polka/webhooks` → Handle Polka webhook events

---

## 🔐 Authentication Flow

1. User logs in with credentials
2. Server validates and returns a JWT access token + refresh token
3. Access token is used for protected routes
4. Refresh token is used to obtain a new access token
5. Tokens can be revoked when needed

---

## 📊 Metrics

Chirpy tracks how many times the file server is accessed using atomic counters. This is exposed via:

```
GET /admin/metrics
```

---

## 🧠 Concepts Learned

This project demonstrates:

* Building RESTful APIs in Go
* Middleware usage
* Secure password hashing
* Token-based authentication (JWT)
* Database schema management
* Writing and generating SQL queries
* Handling webhooks
* Structuring scalable backend services

---

## 📁 Project Structure (Simplified)

```
.
├── main.go
├── internal/
│   └── database/
├── sql/
│   ├── schema/
│   └── queries/
├── go.mod
└── README.md
```

---

## 🧪 Future Improvements

* Add pagination for chirps
* Implement rate limiting
* Improve error handling and logging
* Add unit and integration tests
* Deploy to cloud (Docker + CI/CD)

---

## 👤 Author

**Eyuel Fekade**

---

## 📄 License

This project is open source and available under the MIT License.
