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
   - Migrasi RBAC (tabel `roles`, `permissions`, `user_roles`, `role_permissions`, `organizations`, `organization_users`) dijalankan otomatis dari folder `migrations/`.

## Authentication

- **Register**: POST /register (JSON: {"name": "string", "email": "string", "password": "string"})
- **Login**: POST /login (JSON: {"email": "string", "password": "string"}) - Returns access token and sets refresh token cookie
- **Refresh**: POST /refresh - Hanya pakai refresh token di cookie (tidak perlu access token). Mengembalikan access token baru.
- **Logout**: POST /api/logout - **Memerlukan** `Authorization: Bearer <access_token>`. Menghapus refresh token dan cookie.

## API Endpoints (https://7n2q5l0qq1.apidog.io/ - pass : madio123)

Endpoint yang memerlukan **access token** (header: `Authorization: Bearer <access_token>`):

- POST /api/logout - Logout (invalidate session)
- **Users** (permission: users:list / users:read / users:create / users:update / users:delete)
  - GET /api/users - Get all users (pagination)
  - GET /api/users/{id} - Get user by ID
  - POST /api/users - Create user
  - PUT /api/users/{id} - Update user
  - DELETE /api/users/{id} - Delete user
- **Roles** (permission: roles:manage)
  - GET /api/roles - List roles
  - GET /api/roles/{id} - Get role
  - POST /api/roles - Create role
  - PUT /api/roles/{id} - Update role
  - DELETE /api/roles/{id} - Delete role
- **Permissions** (permission: roles:manage)
  - GET /api/permissions - List permissions
  - GET /api/permissions/{id} - Get permission
  - GET /api/permissions/roles/{roleId} - Permissions of a role
  - POST /api/permissions/roles/{roleId}/permissions/{permissionId} - Assign permission to role
  - DELETE /api/permissions/roles/{roleId}/permissions/{permissionId} - Revoke permission from role
- **User roles** (permission: roles:manage)
  - GET /api/users/{userId}/roles - List roles of user
  - POST /api/users/{userId}/roles - Assign role (body: `{"role_id": 1}`)
  - DELETE /api/users/{userId}/roles/{roleId} - Revoke role from user
- **Organizations** (permission: organizations:manage)
  - GET /api/organizations - List organizations (pagination)
  - GET /api/organizations/{id} - Get organization
  - POST /api/organizations - Create organization
  - PUT /api/organizations/{id} - Update organization
  - DELETE /api/organizations/{id} - Delete organization
  - GET /api/organizations/{id}/users - List members
  - POST /api/organizations/{id}/users - Add member (body: `{"user_id": 1, "role_id": 2}` optional)
  - DELETE /api/organizations/{id}/users/{userId} - Remove member

## Request Body (endpoint yang memerlukan body)

Semua request body berikut menggunakan **Content-Type: application/json**.

### Auth (public)

**POST /register**
```json
{
  "name": "string (required)",
  "email": "string (required)",
  "password": "string (required)"
}
```

**POST /login**
```json
{
  "email": "string (required)",
  "password": "string (required)"
}
```

### Users (protected)

**POST /api/users**
```json
{
  "name": "string (required)",
  "email": "string (required)"
}
```

**PUT /api/users/{id}**
```json
{
  "name": "string (required)",
  "email": "string (required)"
}
```

### Roles (protected, permission: roles:manage)

**POST /api/roles**
```json
{
  "name": "string (required)",
  "code": "string (required, unique)",
  "description": "string (optional)"
}
```

**PUT /api/roles/{id}**
```json
{
  "name": "string (required)",
  "code": "string (required)",
  "description": "string (optional)"
}
```

### User roles (protected, permission: roles:manage)

**POST /api/users/{userId}/roles**
```json
{
  "role_id": 1
}
```

### Organizations (protected, permission: organizations:manage)

**POST /api/organizations**
```json
{
  "name": "string (required)",
  "code": "string (required, unique)",
  "description": "string (optional)"
}
```

**PUT /api/organizations/{id}**
```json
{
  "name": "string (required)",
  "code": "string (required)",
  "description": "string (optional)"
}
```

**POST /api/organizations/{id}/users**
```json
{
  "user_id": 1,
  "role_id": 2
}
```
- `user_id` (required): ID user yang ditambahkan ke organisasi.
- `role_id` (optional): ID role user di organisasi tersebut. Bisa tidak dikirim (null).

---

**Catatan:** Endpoint GET dan DELETE tidak memakai request body. Assign/revoke permission ke role memakai URL path (`/api/permissions/roles/{roleId}/permissions/{permissionId}`), tidak ada body.

## RBAC (Role-Based Access Control)

Sistem mengontrol akses berdasarkan **permission** (kode aksi pada resource). User punya **roles**, tiap role punya **permissions**. Akses API dicek per endpoint.

**Tabel:**
- `users` - User (sudah ada)
- `roles` - Role (name, code, description)
- `permissions` - Permission (name, code, resource, action)
- `user_roles` - User ↔ Role (many-to-many)
- `role_permissions` - Role ↔ Permission (many-to-many)
- `organizations` - Organisasi
- `organization_users` - User dalam organisasi (bisa punya role di org)

**Role default (seed):**
- `super_admin` - Semua permission
- `admin` - users, roles, organizations (tanpa permissions:manage)
- `user` - users:list, users:read

**Permission default (seed):**  
`users:list`, `users:read`, `users:create`, `users:update`, `users:delete`, `roles:manage`, `permissions:manage`, `organizations:manage`.

User baru dari **Register** otomatis diberi role **user**. Untuk akses admin, assign role `admin` atau `super_admin` lewat API (perlu token user yang sudah punya `roles:manage`). User pertama bisa diberi role `super_admin` manual di DB: `INSERT INTO user_roles (user_id, role_id) SELECT 1, id FROM roles WHERE code = 'super_admin' LIMIT 1;` (ganti `1` dengan ID user).

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

### Skenario: Access token sudah expired dan user tidak melakukan refresh

**Pertanyaan:** Apakah refresh token akan terhapus atau di-blacklist?

**Jawaban:** **Tidak.** Refresh token **tetap valid** sampai:
- masa berlaku habis (7 hari), atau
- user **logout** (refresh token dihapus dari Redis + cookie), atau
- refresh token **dipakai** untuk refresh (token di-rotate, yang lama dihapus).

**Alasan (saran terbaik):**
- Tujuan refresh token adalah agar user bisa mendapat **access token baru** setelah access token lama kadaluarsa. Jadi wajar jika user tidak refresh selama access token masih valid, lalu baru memanggil `POST /refresh` setelah access token expired.
- Jika refresh token ikut dihapus/di-blacklist hanya karena access token expired, user akan ter-logout paksa dan harus login lagi. Itu buruk untuk UX dan tidak standar.
- Access token yang sudah expired **tidak perlu** di-blacklist: middleware menolak JWT yang `exp`-nya sudah lewat. Key Redis `access_token:{jti}` juga punya TTL 15 menit sehingga otomatis hilang.

**Ringkas:**
| Kejadian | Access token | Refresh token |
|----------|--------------|---------------|
| Access token expired, user tidak refresh | Ditolak (JWT exp) | Tetap valid, bisa dipakai untuk `POST /refresh` |
| User logout | Di-blacklist | Dihapus dari Redis + cookie |
| User panggil refresh | Yang lama di-blacklist, dapat yang baru | Yang lama dihapus (rotation), dapat yang baru |

### Notes
- Access tokens expire in 15 minutes.
- Refresh tokens expire in 7 days and are stored in HTTP-only cookies.
- Use tools like Postman or cURL for testing.
- Ensure PostgreSQL and Redis are running.

- `internal/models/` - Data models (user, role, permission, organization)
- `internal/database/` - DB connection + migrations
- `internal/handlers/` - HTTP handlers (auth, user, role, permission, organization)
- `internal/middleware/` - JWT + RBAC (RequirePermission)
- `internal/rbac/` - GetUserPermissionCodes, HasPermission
- `internal/routes/` - Route setup
- `migrations/` - SQL migrations (RBAC tables + seed)
