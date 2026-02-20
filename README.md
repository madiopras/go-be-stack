# Simple CRUD API in Go with Authentication

This is a simple CRUD backend API built with Go 1.25.6, PostgreSQL, JWT authentication, and Redis caching.

## Setup

1. Install Go 1.25.6
2. Install PostgreSQL and create a database named `main_db`
3. Install Redis
4. Update database credentials in `internal/database/db.go` if needed
5. Update Redis address in `internal/handlers/auth.go` if needed
6. Run `go mod tidy` to download dependencies
7. Create the users table:
   ```sql
   CREATE TABLE users (
       id SERIAL PRIMARY KEY,
       name VARCHAR(100),
       email VARCHAR(100) UNIQUE,
       password VARCHAR(255),
       created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
   );

   ```
7. Run the server: `go run main.go`

## Authentication

- **Register**: POST /register (JSON: {"name": "string", "email": "string", "password": "string"})
- **Login**: POST /login (JSON: {"email": "string", "password": "string"}) - Returns access token and sets refresh token cookie
- **Refresh**: POST /refresh - Hanya pakai refresh token di cookie (tidak perlu access token). Mengembalikan access token baru.
- **Logout**: POST /api/logout - **Memerlukan** `Authorization: Bearer <access_token>`. Menghapus refresh token dan cookie.

## API Endpoints

Endpoint yang memerlukan **access token** (header: `Authorization: Bearer <access_token>`):

- POST /api/logout - Logout (invalidate session)
- GET /api/users - Get all users (dengan pagination)
- GET /api/users/{id} - Get user by ID
- POST /api/users - Create a new user (JSON body: {"name": "string", "email": "string"})
- PUT /api/users/{id} - Update user by ID
- DELETE /api/users/{id} - Delete user by ID

## How to Use the API

### 1. Register a New User
**Endpoint:** `POST /register`

**Request Body:**
```json
{
  "name": "John Doe",
  "email": "john@example.com",
  "password": "securepassword"
}
```

**Response:**
```json
{
  "user": {
    "id": 1,
    "name": "John Doe",
    "email": "john@example.com",
    "created_at": "2026-01-20T09:31:09Z"
  },
  "access_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

### 2. Login
**Endpoint:** `POST /login`

**Request Body:**
```json
{
  "email": "john@example.com",
  "password": "securepassword"
}
```

**Response:** Same as register, includes access token and sets refresh token cookie.

### 3. Get All Users dengan Pagination (Protected)
**Endpoint:** `GET /api/users`

**Headers:**
- `Authorization: Bearer <access_token>`

**Query Parameters:**
- `page` (optional): Nomor halaman (default: 1, minimum: 1)
- `limit` (optional): Jumlah data per halaman (default: 10, maksimal: 100)

**Contoh Request:**
```
GET /api/users?page=1&limit=10
GET /api/users?page=2&limit=20
GET /api/users  (akan menggunakan default: page=1, limit=10)
```

**Response:**
```json
{
  "success": true,
  "message": "Users retrieved successfully",
  "data": [
    {
      "id": 1,
      "name": "John Doe",
      "email": "john@example.com",
      "created_at": "2026-01-20T09:31:09Z"
    },
    {
      "id": 2,
      "name": "Jane Doe",
      "email": "jane@example.com",
      "created_at": "2026-01-21T10:15:30Z"
    }
  ],
  "meta": {
    "page": 1,
    "per_page": 10,
    "total": 25,
    "total_pages": 3
  }
}
```

**Catatan:**
- Data diurutkan berdasarkan `created_at DESC` (terbaru dulu)
- Jika `page` melebihi `total_pages`, akan mengembalikan array kosong dengan metadata yang benar
- Parameter `limit` maksimal 100 untuk menghindari overload server

### 4. Get User by ID (Protected)
**Endpoint:** `GET /api/users/{id}` (e.g., /api/users/1)

**Headers:**
- `Authorization: Bearer <access_token>`

**Response:**
```json
{
  "success": true,
  "message": "User retrieved successfully",
  "data": {
    "id": 1,
    "name": "John Doe",
    "email": "john@example.com",
    "created_at": "2026-01-20T09:31:09Z"
  }
}
```

### 5. Create a New User (Protected)
**Endpoint:** `POST /api/users`

**Headers:**
- `Authorization: Bearer <access_token>`
- `Content-Type: application/json`

**Request Body:**
```json
{
  "name": "Jane Doe",
  "email": "jane@example.com"
}
```

**Response:**
```json
{
  "id": 2,
  "name": "Jane Doe",
  "email": "jane@example.com",
  "created_at": "2026-01-20T09:31:09Z"
}
```

### 6. Update a User (Protected)
**Endpoint:** `PUT /api/users/{id}` (e.g., /api/users/1)

**Headers:**
- `Authorization: Bearer <access_token>`
- `Content-Type: application/json`

**Request Body:**
```json
{
  "name": "John Smith",
  "email": "johnsmith@example.com"
}
```

**Response:**
```json
{
  "id": 1,
  "name": "John Smith",
  "email": "johnsmith@example.com",
  "created_at": "2026-01-20T09:31:09Z"
}
```

### 7. Delete a User (Protected)
**Endpoint:** `DELETE /api/users/{id}` (e.g., /api/users/1)

**Headers:**
- `Authorization: Bearer <access_token>`

**Response:** 204 No Content

### 8. Refresh Access Token
**Endpoint:** `POST /refresh`

**Note:** Tidak perlu access token. Refresh token dikirim otomatis via cookie. Gunakan saat access token sudah kadaluarsa.

**Response:** JSON dengan format standar, `data` berisi `access_token` baru.

### 9. Logout
**Endpoint:** `POST /api/logout`

**Headers:**
- `Authorization: Bearer <access_token>` (wajib)

Menghapus refresh token dari Redis dan menghapus cookie. Hanya user yang sedang login (punya access token valid) yang bisa memanggil logout.

### Notes
- Access tokens expire in 15 minutes.
- Refresh tokens expire in 7 days and are stored in HTTP-only cookies.
- Use tools like Postman or cURL for testing.
- Ensure PostgreSQL and Redis are running.

- `internal/models/` - Data models
- `internal/database/` - Database connection
- `internal/handlers/` - HTTP handlers (auth and user)
- `internal/middleware/` - JWT middleware
- `internal/routes/` - Route setup
