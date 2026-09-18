# Smart Journal Backend

Backend Go untuk Smart Journal dengan omni-input, Firebase Auth, Firestore, Gemini API, session cookie, dan email reminder via Resend.

## Struktur

```text
cmd/server/main.go              Entry point HTTP server
configs/config.go               Loader konfigurasi dari environment variable
internal/handler                HTTP handler berbasis net/http
internal/middleware             Middleware Firebase Auth
internal/service                Business logic, Gemini REST client, Resend email sender
internal/repository             Adapter Firestore
internal/scheduler              Minute-polling scheduler untuk reminder
internal/model                  Entity dan validasi domain
```

## Environment Variable

```env
APP_ENV=development
HOST=127.0.0.1
PORT=8080
CORS_ALLOWED_ORIGINS=http://127.0.0.1:5173
FRONTEND_BASE_URL=http://127.0.0.1:5173
COOKIE_SECURE=false
FIREBASE_PROJECT_ID=your-firebase-project-id
FIREBASE_CREDENTIALS_FILE=/path/to/service-account.json
GEMINI_API_KEY=your-gemini-api-key
GEMINI_MODEL=gemini-2.0-flash
GEMINI_BASE_URL=https://generativelanguage.googleapis.com/v1beta
REQUEST_TIMEOUT_SECONDS=20
SESSION_DURATION_HOURS=120
EMAIL_PROVIDER=resend_api
RESEND_API_KEY=your-resend-api-key
RESEND_API_BASE_URL=https://api.resend.com
EMAIL_FROM="Smart Journal <onboarding@resend.dev>"
NOTIFICATION_TO_EMAIL=
SMTP_HOST=smtp.resend.com
SMTP_PORT=465
SMTP_USERNAME=resend
SMTP_PASSWORD=your-resend-api-key
SMTP_IMPLICIT_TLS=true
EMAIL_VERIFICATION_BASE_URL=http://127.0.0.1:8080
EMAIL_VERIFICATION_REDIRECT_URL=http://127.0.0.1:5173/email-verified
EMAIL_VERIFICATION_DURATION_HOURS=24
SCHEDULER_ENABLED=true
SCHEDULER_INTERVAL_SECONDS=60
```

## Menjalankan

Buat file `.env` dari template:

```bash
cp .env.example .env
```

Isi nilai Firebase dan Gemini di `.env`, lalu jalankan:

```bash
go mod tidy
go run ./cmd/server
```

Secara default server listen di `127.0.0.1:8080`. Untuk VPS atau reverse proxy yang perlu bind ke semua interface, ubah:

```env
HOST=0.0.0.0
PORT=8080
```

## Email Resend

Backend mendukung dua mode pengiriman email:

```env
EMAIL_PROVIDER=resend_api
```

Mode ini memakai Resend REST API:

```env
RESEND_API_KEY=your-resend-api-key
RESEND_API_BASE_URL=https://api.resend.com
EMAIL_FROM="Smart Journal <onboarding@your-domain.com>"
```

Atau:

```env
EMAIL_PROVIDER=smtp
SMTP_HOST=smtp.resend.com
SMTP_PORT=465
SMTP_USERNAME=resend
SMTP_PASSWORD=your-resend-api-key
SMTP_IMPLICIT_TLS=true
EMAIL_FROM="Smart Journal <onboarding@your-domain.com>"
```

Untuk production, pakai domain yang sudah diverifikasi di Resend pada `EMAIL_FROM`.

`NOTIFICATION_TO_EMAIL` bersifat opsional. Jika diisi, semua reminder dikirim ke email tersebut. Jika kosong, reminder dikirim ke email profile user.

## API

Auth web memakai Firebase session cookie:

- `POST /api/auth/session` menerima Firebase ID token dari header `Authorization`.
- Endpoint lama memakai cookie `smart_journal_session`, bukan bearer token.
- Method mutasi `POST`, `PATCH`, `PUT`, dan `DELETE` perlu header `X-CSRF-Token` yang nilainya sama dengan cookie `smart_journal_csrf`.
- Fetch dari frontend harus memakai `credentials: "include"`.

Ringkasan header:

```http
GET /api/*
Cookie: smart_journal_session=<firebase_session_cookie>
```

```http
POST/PATCH/PUT/DELETE /api/*
Cookie: smart_journal_session=<firebase_session_cookie>; smart_journal_csrf=<csrf_token>
X-CSRF-Token: <csrf_token>
```

Pengecualian: `POST /api/auth/session` hanya butuh `Authorization: Bearer <firebase_id_token>` karena CSRF cookie belum dibuat.

Email verifikasi tidak dikirim oleh Firebase client. Setelah user register dan session cookie dibuat, frontend memanggil `POST /api/auth/email-verification`; backend membuat token verifikasi, menyimpan hash token di Firestore, lalu mengirim email melalui Resend.

Urutan register yang disarankan:

1. Frontend register user dengan Firebase Auth client.
2. Frontend ambil Firebase ID token.
3. Frontend panggil `POST /api/auth/session`.
4. Frontend panggil `POST /api/profile`.
5. Frontend panggil `POST /api/auth/email-verification`.

### Time Sync FE/BE/DB

Semua timestamp API memakai RFC3339/ISO 8601. Kontrak utamanya: **FE mengirim waktu dengan timezone, BE menyimpan UTC, DB menyimpan UTC, FE menampilkan sesuai timezone user**.

Aturan frontend:

- Simpan timezone user dari profile, misalnya `Asia/Jakarta`.
- Saat user memilih tanggal/jam lokal, kirim timestamp lengkap dengan offset.
- Jangan kirim timestamp tanpa timezone.
- Saat menerima response dengan suffix `Z`, anggap itu UTC lalu format ke timezone profile user.

Contoh user memilih reminder jam `14:50` WIB:

```json
{
  "remind_at": "2026-09-16T14:50:00+07:00"
}
```

Contoh yang tidak boleh dikirim:

```json
{
  "remind_at": "2026-09-16T14:50:00"
}
```

Aturan backend:

- Parse request timestamp sebagai RFC3339.
- Convert timestamp ke UTC sebelum simpan.
- Return semua timestamp dalam UTC dengan suffix `Z`.
- Validasi profile timezone memakai IANA timezone seperti `Asia/Jakarta`; value seperti `Jakarta` akan ditolak.

Aturan Firestore:

- Field waktu seperti `created_at`, `updated_at`, `remind_at`, dan `sent_at` disimpan sebagai Firestore timestamp/UTC instant.
- Jangan simpan jam lokal sebagai string utama untuk reminder.

Aturan AI/Gemini:

- Backend mengirim `current_time` sesuai `timezone` dari `users/{userId}/profile/settings`.
- Backend mengirim `current_timezone` sebagai IANA timezone, misalnya `Asia/Jakarta`.
- Jika profile belum ada atau timezone kosong, default-nya `Asia/Jakarta`.
- Gemini diminta mengembalikan `remind_at` dengan offset eksplisit, misalnya `2026-09-16T15:07:00+07:00`.
- Jika Gemini tetap mengembalikan timestamp UTC `Z` padahal input user tidak menyebut UTC/GMT/Zulu, backend memperlakukannya sebagai jam lokal profile untuk mencegah kasus `15:07 WIB` tersimpan sebagai `15:07Z`.

Contoh alur sinkron:

```text
User input: 15:07 WIB
FE request: 2026-09-16T15:07:00+07:00
BE store:   2026-09-16T08:07:00Z
BE return:  2026-09-16T08:07:00Z
FE display: 15:07 WIB
```

Reminder lama yang sudah tersimpan dengan basis waktu salah tidak otomatis berubah; data lama perlu dibuat ulang atau dimigrasikan.

### Auth Session

Tukar Firebase ID token menjadi cookie session setelah user login/register di frontend:

```http
POST /api/auth/session
Authorization: Bearer <firebase_id_token>
```

Response `200 OK` akan menyetel dua cookie:

```http
Set-Cookie: smart_journal_session=<firebase_session_cookie>; HttpOnly; SameSite=Lax; Path=/
Set-Cookie: smart_journal_csrf=<csrf_token>; SameSite=Lax; Path=/
```

Saat `APP_ENV=production` atau `COOKIE_SECURE=true`, kedua cookie memakai atribut `Secure`.

Response body:

```json
{
  "authenticated": true,
  "csrf_token": "csrf-token-value",
  "user": {
    "uid": "firebase-user-id",
    "email": "user@example.com",
    "email_verified": true
  }
}
```

Cek session dari cookie:

```http
GET /api/auth/session
Cookie: smart_journal_session=<firebase_session_cookie>; smart_journal_csrf=<csrf_token>
```

Response `200 OK`:

```json
{
  "authenticated": true,
  "csrf_token": "csrf-token-value",
  "user": {
    "uid": "firebase-user-id",
    "email": "user@example.com",
    "email_verified": true
  }
}
```

### Email Verification

Kirim email verifikasi ke email profile user yang sedang login:

```http
POST /api/auth/email-verification
Cookie: smart_journal_session=<firebase_session_cookie>; smart_journal_csrf=<csrf_token>
X-CSRF-Token: csrf-token-value
```

Response `200 OK`:

```json
{
  "sent": true
}
```

Token verifikasi hanya berlaku untuk email profile saat token dibuat. Kalau user mengganti email sebelum link diklik, link lama akan ditolak dan user perlu meminta email verifikasi baru.

Link yang dikirim ke email akan mengarah ke endpoint ini:

```http
GET /api/auth/email-verification?token=<verification_token>
```

Setelah token diproses, backend akan redirect ke frontend:

```http
303 See Other
Location: http://127.0.0.1:5173/email-verified?status=success
```

Jika token gagal, expired, sudah pernah dipakai, atau email profile sudah berubah, backend redirect ke URL yang sama dengan status gagal:

```http
303 See Other
Location: http://127.0.0.1:5173/email-verified?reason=expired&status=failed
```

Frontend sebaiknya membuat route `/email-verified` dengan tampilan sederhana: status berhasil/gagal dan tombol kembali ke home/dashboard. Query `status` bernilai `success` atau `failed`; query `reason` bisa bernilai `expired`, `already_used`, `email_changed`, atau `invalid`.

Contoh data profile setelah verifikasi berhasil:

```json
{
  "user_id": "firebase-user-id",
  "email": "user@example.com",
  "email_verified_at": "2026-09-15T10:15:00Z",
  "name": "Mikha",
  "avatar_url": "https://example.com/avatar.png",
  "bio": "Building quietly.",
  "timezone": "Asia/Jakarta",
  "created_at": "2026-09-15T10:00:00Z",
  "updated_at": "2026-09-15T10:15:00Z"
}
```

Endpoint verify juga menandai `email_verified=true` di Firebase Auth lewat Firebase Admin SDK. Session cookie yang sudah terlanjur dibuat bisa masih membawa claim lama sampai user membuat session baru, jadi frontend sebaiknya memakai `email_verified_at` dari profile sebagai status verifikasi aplikasi.

Logout:

```http
DELETE /api/auth/session
Cookie: smart_journal_session=<firebase_session_cookie>; smart_journal_csrf=<csrf_token>
X-CSRF-Token: csrf-token-value
```

Response `200 OK`:

```json
{
  "authenticated": false
}
```

### Health Check

```http
GET /healthz
```

Response `200 OK`:

```json
{
  "status": "ok"
}
```

### Omni-Input Journal Entry

```http
POST /api/journals
Content-Type: application/json
Cookie: smart_journal_session=<firebase_session_cookie>; smart_journal_csrf=<csrf_token>
X-CSRF-Token: csrf-token-value

{
  "raw_text": "Beli paket XL 60rb, ingatkan 3 hari sebelum masa aktif habis"
}
```

Response berisi note yang selalu dibuat, hasil AI, transaksi bila intent finansial ditemukan, dan reminder bila intent pengingat ditemukan.

Response `201 Created`:

```json
{
  "note": {
    "id": "OxOIDA95JWgi6yNtmC1K",
    "raw_text": "Beli paket XL 60rb, ingatkan 3 hari sebelum masa aktif habis",
    "title": "Beli Paket XL",
    "created_at": "2026-09-12T15:04:24Z",
    "updated_at": "2026-09-12T15:04:24Z"
  },
  "ai_result": {
    "intents": ["expense", "reminder"],
    "title": "Beli Paket XL",
    "financial_data": {
      "amount": 60000,
      "category": "Internet"
    },
    "reminder_data": {
      "remind_at": "2026-10-09T15:04:24Z",
      "message": "Masa aktif paket XL akan habis dalam 3 hari"
    }
  },
  "transactions": [
    {
      "id": "fhtTDinvjMaZr76fOrks",
      "note_id": "OxOIDA95JWgi6yNtmC1K",
      "type": "expense",
      "amount": 60000,
      "category": "Internet",
      "created_at": "2026-09-12T15:04:24Z",
      "updated_at": "2026-09-12T15:04:24Z"
    }
  ],
  "reminders": [
    {
      "id": "7b6qHUGbtrHV6tCNUKDi",
      "user_id": "firebase-user-id",
      "note_id": "OxOIDA95JWgi6yNtmC1K",
      "remind_at": "2026-10-09T15:04:24Z",
      "message": "Masa aktif paket XL akan habis dalam 3 hari",
      "status": "pending",
      "created_at": "2026-09-12T15:04:24Z",
      "updated_at": "2026-09-12T15:04:24Z"
    }
  ]
}
```

### Profile

Frontend sebaiknya memanggil `POST /api/profile` setelah user selesai register. Endpoint ini idempotent: kalau profile sudah ada, data existing dikembalikan.

Field `timezone` wajib berupa IANA timezone yang valid, misalnya `Asia/Jakarta`, `Asia/Makassar`, `Asia/Jayapura`, atau `UTC`. Jika kosong, backend memakai default `Asia/Jakarta`.

```http
POST /api/profile
Content-Type: application/json
Cookie: smart_journal_session=<firebase_session_cookie>; smart_journal_csrf=<csrf_token>
X-CSRF-Token: csrf-token-value

{
  "email": "user@example.com",
  "name": "Mikha",
  "avatar_url": "https://example.com/avatar.png",
  "bio": "Building quietly.",
  "timezone": "Asia/Jakarta"
}
```

Response `201 Created`:

```json
{
  "user_id": "firebase-user-id",
  "email": "user@example.com",
  "name": "Mikha",
  "avatar_url": "https://example.com/avatar.png",
  "bio": "Building quietly.",
  "timezone": "Asia/Jakarta",
  "created_at": "2026-09-15T10:00:00Z",
  "updated_at": "2026-09-15T10:00:00Z"
}
```

```http
GET /api/profile
PATCH /api/profile
```

Untuk `GET /api/profile`, browser cukup mengirim cookie session via `credentials: "include"`. Untuk `PATCH /api/profile`, kirim juga header `X-CSRF-Token`.

Field setting yang bisa diedit:

```json
{
  "name": "Nama Baru",
  "avatar_url": "https://example.com/avatar-baru.png",
  "bio": "Bio baru",
  "timezone": "Asia/Jakarta"
}
```

Response `GET /api/profile` `200 OK`:

```json
{
  "user_id": "firebase-user-id",
  "email": "user@example.com",
  "name": "Nama Baru",
  "avatar_url": "https://example.com/avatar-baru.png",
  "bio": "Bio baru",
  "timezone": "Asia/Jakarta",
  "created_at": "2026-09-15T10:00:00Z",
  "updated_at": "2026-09-15T10:05:00Z"
}
```

Response `PATCH /api/profile` `200 OK`:

```json
{
  "user_id": "firebase-user-id",
  "email": "user@example.com",
  "name": "Nama Baru",
  "avatar_url": "https://example.com/avatar-baru.png",
  "bio": "Bio baru",
  "timezone": "Asia/Jakarta",
  "created_at": "2026-09-15T10:00:00Z",
  "updated_at": "2026-09-15T10:05:00Z"
}
```

### Notes

Manual note tetap bisa dibuat tanpa AI:

```http
GET /api/notes?limit=50
POST /api/notes
GET /api/notes/{noteId}
PATCH /api/notes/{noteId}
DELETE /api/notes/{noteId}
```

Header request:

```http
GET /api/notes?limit=50
Cookie: smart_journal_session=<firebase_session_cookie>
```

```http
POST /api/notes
Content-Type: application/json
Cookie: smart_journal_session=<firebase_session_cookie>; smart_journal_csrf=<csrf_token>
X-CSRF-Token: <csrf_token>
```

Create body:

```json
{
  "raw_text": "Catatan manual saat AI sedang mati",
  "title": "Catatan Manual"
}
```

Response `POST /api/notes` `201 Created`:

```json
{
  "id": "note-id",
  "raw_text": "Catatan manual saat AI sedang mati",
  "title": "Catatan Manual",
  "created_at": "2026-09-15T10:00:00Z",
  "updated_at": "2026-09-15T10:00:00Z"
}
```

Update body:

```json
{
  "raw_text": "Isi catatan yang diedit",
  "title": "Judul Baru"
}
```

Response `GET /api/notes?limit=50` `200 OK`:

```json
{
  "notes": [
    {
      "id": "note-id",
      "raw_text": "Catatan manual saat AI sedang mati",
      "title": "Catatan Manual",
      "created_at": "2026-09-15T10:00:00Z",
      "updated_at": "2026-09-15T10:00:00Z"
    }
  ]
}
```

Response `GET /api/notes/{noteId}` `200 OK`:

```json
{
  "id": "note-id",
  "raw_text": "Catatan manual saat AI sedang mati",
  "title": "Catatan Manual",
  "created_at": "2026-09-15T10:00:00Z",
  "updated_at": "2026-09-15T10:00:00Z"
}
```

Response `PATCH /api/notes/{noteId}` `200 OK`:

```json
{
  "id": "note-id",
  "raw_text": "Isi catatan yang diedit",
  "title": "Judul Baru",
  "created_at": "2026-09-15T10:00:00Z",
  "updated_at": "2026-09-15T10:10:00Z"
}
```

Response `DELETE /api/notes/{noteId}`:

```http
204 No Content
```

### Transactions

```http
GET /api/transactions?limit=50
POST /api/transactions
GET /api/transactions/{transactionId}
PATCH /api/transactions/{transactionId}
DELETE /api/transactions/{transactionId}
```

Header request:

```http
GET /api/transactions?limit=50
Cookie: smart_journal_session=<firebase_session_cookie>
```

```http
POST /api/transactions
Content-Type: application/json
Cookie: smart_journal_session=<firebase_session_cookie>; smart_journal_csrf=<csrf_token>
X-CSRF-Token: <csrf_token>
```

Create body:

```json
{
  "note_id": "optional-note-id",
  "type": "expense",
  "amount": 60000,
  "category": "Internet"
}
```

`type` hanya menerima `income` atau `expense`.

Response `POST /api/transactions` `201 Created`:

```json
{
  "id": "transaction-id",
  "note_id": "optional-note-id",
  "type": "expense",
  "amount": 60000,
  "category": "Internet",
  "created_at": "2026-09-15T10:00:00Z",
  "updated_at": "2026-09-15T10:00:00Z"
}
```

Response `GET /api/transactions?limit=50` `200 OK`:

```json
{
  "transactions": [
    {
      "id": "transaction-id",
      "note_id": "optional-note-id",
      "type": "expense",
      "amount": 60000,
      "category": "Internet",
      "created_at": "2026-09-15T10:00:00Z",
      "updated_at": "2026-09-15T10:00:00Z"
    }
  ]
}
```

Response `GET /api/transactions/{transactionId}` `200 OK`:

```json
{
  "id": "transaction-id",
  "note_id": "optional-note-id",
  "type": "expense",
  "amount": 60000,
  "category": "Internet",
  "created_at": "2026-09-15T10:00:00Z",
  "updated_at": "2026-09-15T10:00:00Z"
}
```

Response `PATCH /api/transactions/{transactionId}` `200 OK`:

```json
{
  "id": "transaction-id",
  "note_id": "optional-note-id",
  "type": "income",
  "amount": 75000,
  "category": "Project",
  "created_at": "2026-09-15T10:00:00Z",
  "updated_at": "2026-09-15T10:10:00Z"
}
```

Response `DELETE /api/transactions/{transactionId}`:

```http
204 No Content
```

### Reminders

```http
GET /api/reminders?limit=50
POST /api/reminders
GET /api/reminders/{reminderId}
PATCH /api/reminders/{reminderId}
DELETE /api/reminders/{reminderId}
POST /api/reminders/send-due
```

Header request:

```http
GET /api/reminders?limit=50
Cookie: smart_journal_session=<firebase_session_cookie>
```

```http
POST /api/reminders
Content-Type: application/json
Cookie: smart_journal_session=<firebase_session_cookie>; smart_journal_csrf=<csrf_token>
X-CSRF-Token: <csrf_token>
```

Create body:

```json
{
  "note_id": "optional-note-id",
  "remind_at": "2026-10-09T22:04:24+07:00",
  "message": "Masa aktif paket XL akan habis dalam 3 hari",
  "status": "pending"
}
```

`status` hanya menerima `pending` atau `sent`.

`remind_at` harus RFC3339 lengkap dengan timezone. Untuk waktu lokal Indonesia Barat, gunakan suffix `+07:00`; backend akan menyimpan dan mengembalikan nilai UTC `Z`.

Saat scheduler mengirim reminder, email dikirim ke `NOTIFICATION_TO_EMAIL` bila env itu diisi. Jika kosong, backend memakai `email` dari `users/{userId}/profile/settings`.

Response `POST /api/reminders` `201 Created`:

```json
{
  "id": "reminder-id",
  "user_id": "firebase-user-id",
  "note_id": "optional-note-id",
  "remind_at": "2026-10-09T15:04:24Z",
  "message": "Masa aktif paket XL akan habis dalam 3 hari",
  "status": "pending",
  "created_at": "2026-09-15T10:00:00Z",
  "updated_at": "2026-09-15T10:00:00Z"
}
```

Response `GET /api/reminders?limit=50` `200 OK`:

```json
{
  "reminders": [
    {
      "id": "reminder-id",
      "user_id": "firebase-user-id",
      "note_id": "optional-note-id",
      "remind_at": "2026-10-09T15:04:24Z",
      "message": "Masa aktif paket XL akan habis dalam 3 hari",
      "status": "pending",
      "created_at": "2026-09-15T10:00:00Z",
      "updated_at": "2026-09-15T10:00:00Z"
    }
  ]
}
```

Response `GET /api/reminders/{reminderId}` `200 OK`:

```json
{
  "id": "reminder-id",
  "user_id": "firebase-user-id",
  "note_id": "optional-note-id",
  "remind_at": "2026-10-09T15:04:24Z",
  "message": "Masa aktif paket XL akan habis dalam 3 hari",
  "status": "pending",
  "created_at": "2026-09-15T10:00:00Z",
  "updated_at": "2026-09-15T10:00:00Z"
}
```

Response `PATCH /api/reminders/{reminderId}` `200 OK`:

```json
{
  "id": "reminder-id",
  "user_id": "firebase-user-id",
  "note_id": "optional-note-id",
  "remind_at": "2026-10-10T09:00:00Z",
  "message": "Reminder sudah diedit",
  "status": "pending",
  "created_at": "2026-09-15T10:00:00Z",
  "updated_at": "2026-09-15T10:10:00Z"
}
```

Response `DELETE /api/reminders/{reminderId}`:

```http
204 No Content
```

Response `POST /api/reminders/send-due` `200 OK`:

```json
{
  "sent": 1
}
```

### Error Response

Response error memakai bentuk yang sama di semua endpoint:

```json
{
  "error": "raw_text is required"
}
```

Contoh status yang umum:

```http
400 Bad Request
401 Unauthorized
404 Not Found
405 Method Not Allowed
500 Internal Server Error
```

### Frontend Fetch Example

Login session:

```ts
const idToken = await user.getIdToken();

const response = await fetch("http://127.0.0.1:8080/api/auth/session", {
  method: "POST",
  credentials: "include",
  headers: {
    Authorization: `Bearer ${idToken}`
  }
});

const session = await response.json();
```

Create profile lalu kirim email verifikasi setelah register:

```ts
const csrfToken = decodeURIComponent(cookieValue("smart_journal_csrf"));

await fetch("http://127.0.0.1:8080/api/profile", {
  method: "POST",
  credentials: "include",
  headers: {
    "Content-Type": "application/json",
    "X-CSRF-Token": csrfToken
  },
  body: JSON.stringify({
    email: user.email,
    name: user.displayName ?? "",
    avatar_url: user.photoURL ?? "",
    bio: "",
    timezone: "Asia/Jakarta"
  })
});

await fetch("http://127.0.0.1:8080/api/auth/email-verification", {
  method: "POST",
  credentials: "include",
  headers: {
    "X-CSRF-Token": csrfToken
  }
});
```

Request mutasi setelah session aktif:

```ts
function cookieValue(name: string) {
  return document.cookie
    .split("; ")
    .find((item) => item.startsWith(`${name}=`))
    ?.split("=")[1] ?? "";
}

const csrfToken = decodeURIComponent(cookieValue("smart_journal_csrf"));

await fetch("http://127.0.0.1:8080/api/notes", {
  method: "POST",
  credentials: "include",
  headers: {
    "Content-Type": "application/json",
    "X-CSRF-Token": csrfToken
  },
  body: JSON.stringify({
    raw_text: "Catatan manual",
    title: "Manual"
  })
});
```

Request baca data setelah session aktif:

```ts
const response = await fetch("http://127.0.0.1:8080/api/notes?limit=50", {
  method: "GET",
  credentials: "include"
});

const data = await response.json();
```

Logout:

```ts
const csrfToken = decodeURIComponent(cookieValue("smart_journal_csrf"));

await fetch("http://127.0.0.1:8080/api/auth/session", {
  method: "DELETE",
  credentials: "include",
  headers: {
    "X-CSRF-Token": csrfToken
  }
});
```

### CORS

Karena auth memakai cookie, CORS harus memakai origin spesifik dan credentials:

```env
CORS_ALLOWED_ORIGINS=http://127.0.0.1:5173
```

Backend akan mengirim:

```http
Access-Control-Allow-Credentials: true
Access-Control-Allow-Headers: Content-Type, Authorization, X-CSRF-Token
```

Jangan pakai `Access-Control-Allow-Origin: *` untuk request yang membawa cookie.

## Firestore

Data disimpan per user:

```text
users/{userId}/profile/settings
users/{userId}/notes/{noteId}
users/{userId}/transactions/{transactionId}
users/{userId}/reminders/{reminderId}
email_verifications/{tokenHash}
```

Email reminder dikirim melalui Resend. Field tujuan memakai `NOTIFICATION_TO_EMAIL` dari `.env` bila tersedia; jika kosong, backend memakai `email` dari `users/{userId}/profile/settings`.

Query scheduler memakai `CollectionGroup("reminders")` dengan filter `status == "pending"` dan `remind_at <= now`, lalu `OrderBy("remind_at")`. Firestore dapat meminta composite index untuk query ini saat pertama dijalankan; ikuti link index yang diberikan Firebase di log/error.

## Deployment VPS

Build binary:

```bash
go build -o smart-journal ./cmd/server
```

Contoh systemd service:

```ini
[Unit]
Description=Smart Journal Backend
After=network.target

[Service]
WorkingDirectory=/opt/smart-journal
ExecStart=/opt/smart-journal/smart-journal
EnvironmentFile=/opt/smart-journal/.env
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

## Catatan Maintenance

- Handler hanya mengurus HTTP parsing dan response.
- Service menyimpan aturan bisnis: validasi intent, distribusi note, transaction, dan reminder.
- Repository menjadi satu-satunya layer yang mengenal struktur Firestore.
- Integrasi Gemini dan Resend API memakai `net/http`; mode SMTP memakai `net/smtp`.
- Scheduler disejajarkan ke menit berikutnya sebelum polling agar eksekusi reminder stabil di detik `00`.

## Prompt Injection Guard

Omni-input diperlakukan sebagai data tidak tepercaya:

- Prompt Gemini secara eksplisit melarang model mengikuti instruksi yang ada di teks jurnal.
- Teks user dikirim ke Gemini sebagai field JSON `untrusted_journal_entry`, bukan ditempel sebagai instruksi bebas.
- Response AI wajib JSON dan didecode dengan `DisallowUnknownFields`.
- Backend tetap memvalidasi intent, nominal, reminder timestamp, dan panjang field sebelum menyimpan data.
- Isi reminder yang masuk ke email HTML di-escape sebelum dikirim.

Catatan: prompt injection tidak bisa dijamin hilang 100% hanya dengan prompt. Proteksi utamanya adalah backend tidak pernah mengeksekusi instruksi dari output AI dan hanya menerima struktur data yang lolos validasi.
