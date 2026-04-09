Berikut adalah rancangan struktur database (Schema) menggunakan SQLite. Fokus utama dari desain ini adalah memisahkan data identitas pemilih dengan data hasil *voting* untuk menjamin **asas kerahasiaan (Luber Jurdil)**, sambil tetap menyimpan rekam jejak (*audit log*) yang bisa divalidasi.

### 1. Entity Relationship Design (SQL DDL)

Karena menggunakan SQLite, tipe datanya disesuaikan dengan standar SQLite (menggunakan `INTEGER` untuk *boolean* dan `TEXT` untuk *enum*).

```sql
-- Tabel untuk Admin Panel Shadcn
CREATE TABLE admins (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Tabel Pengaturan (Untuk teks pengumuman di Landing Page)
CREATE TABLE settings (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    announcement_text TEXT,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Tabel Kandidat (Menggabungkan Ketua dan Wakil dengan penanda 'position')
CREATE TABLE candidates (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    class_name TEXT NOT NULL,
    photo_url TEXT,
    vision TEXT,
    mission TEXT,
    program TEXT,
    position TEXT NOT NULL CHECK(position IN ('CHAIRMAN', 'VICE_CHAIRMAN')),
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Tabel Data Pemilih (DPT)
CREATE TABLE voters (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    uuid TEXT NOT NULL UNIQUE, -- Digunakan untuk generate QR/Barcode
    name TEXT NOT NULL,
    class_name TEXT NOT NULL,
    phone_number TEXT NOT NULL, -- Untuk integrasi OneSender WA
    has_voted INTEGER DEFAULT 0, -- 0 = Belum, 1 = Sudah
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Tabel Hasil Pemilihan & Audit Log (Terpisah dari identitas asli)
CREATE TABLE votes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    masked_uuid TEXT NOT NULL, -- Contoh format: "123e4567-****-****-************"
    chairman_id INTEGER NOT NULL,
    vice_chairman_id INTEGER NOT NULL,
    voted_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (chairman_id) REFERENCES candidates(id),
    FOREIGN KEY (vice_chairman_id) REFERENCES candidates(id)
);
```

### 2. Implementasi Golang Structs

Jika Anda menggunakan ORM seperti GORM (sangat disarankan untuk mempercepat *development*) atau standar `database/sql`, representasi *struct* di Golang akan terlihat seperti ini:

```go
package models

import (
	"time"
)

type Admin struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	Username     string    `json:"username" gorm:"unique;not null"`
	PasswordHash string    `json:"-" gorm:"not null"` // Disembunyikan dari response JSON
	CreatedAt    time.Time `json:"created_at"`
}

type Setting struct {
	ID               uint      `json:"id" gorm:"primaryKey"`
	AnnouncementText string    `json:"announcement_text" gorm:"type:text"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type Candidate struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Name      string    `json:"name" gorm:"not null"`
	ClassName string    `json:"class_name" gorm:"not null"`
	PhotoURL  string    `json:"photo_url"`
	Vision    string    `json:"vision" gorm:"type:text"`
	Mission   string    `json:"mission" gorm:"type:text"`
	Program   string    `json:"program" gorm:"type:text"`
	Position  string    `json:"position" gorm:"not null"` // "CHAIRMAN" atau "VICE_CHAIRMAN"
	CreatedAt time.Time `json:"created_at"`
}

type Voter struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	UUID        string    `json:"uuid" gorm:"uniqueIndex;not null"`
	Name        string    `json:"name" gorm:"not null"`
	ClassName   string    `json:"class_name" gorm:"not null"`
	PhoneNumber string    `json:"phone_number" gorm:"not null"`
	HasVoted    bool      `json:"has_voted" gorm:"default:false"`
	CreatedAt   time.Time `json:"created_at"`
}

type Vote struct {
	ID             uint      `json:"id" gorm:"primaryKey"`
	MaskedUUID     string    `json:"masked_uuid" gorm:"not null"`
	ChairmanID     uint      `json:"chairman_id" gorm:"not null"`
	ViceChairmanID uint      `json:"vice_chairman_id" gorm:"not null"`
	VotedAt        time.Time `json:"voted_at" gorm:"autoCreateTime"`
    
    // Relasi (Opsional, untuk mempermudah query join)
	Chairman       Candidate `json:"chairman" gorm:"foreignKey:ChairmanID"`
	ViceChairman   Candidate `json:"vice_chairman" gorm:"foreignKey:ViceChairmanID"`
}
```

### 3. Penjelasan Logika Kerahasiaan (Data Flow)

Struktur di atas mengamankan proses pemilihan dengan alur logika *database* berikut:

1. **Validasi Scanner:** Saat siswa *scan* QR, sistem melakukan *query* ke tabel `voters` berdasarkan `uuid`. Jika `has_voted == 0`, izinkan masuk ke halaman pemilihan.
2. **Proses Submit:** Saat siswa klik "Kirim Pilihan", *backend* melakukan *database transaction* yang berisi dua perintah:
   * **UPDATE** tabel `voters`: Ubah `has_voted = 1` berdasarkan `uuid` yang sedang *login*.
   * **INSERT** ke tabel `votes`: Masukkan `chairman_id`, `vice_chairman_id`, dan *generate* string baru ke kolom `masked_uuid` (dengan me-*replace* karakter tengah UUID asli dengan asterisk `*`).
3. **Kerahasiaan Terjamin:** Karena tidak ada *Foreign Key* (relasi langsung) antara tabel `voters` dan `votes`, admin *database* paling mahir sekalipun tidak akan bisa mengetahui siswa A memilih kandidat yang mana. Tabel `votes` murni hanya berisi rekap log suara masuk yang valid.