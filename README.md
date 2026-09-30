# Gopilketos - Modern OSIS E-Voting System

[![Go Version](https://img.shields.io/badge/Go-1.25%2B-00ADD8?style=flat&logo=go)](https://golang.org)
[![Echo Framework](https://img.shields.io/badge/Echo-v5.1.0-00ADD8?style=flat)](https://echo.labstack.com)
[![Platform](https://img.shields.io/badge/Platform-Linux%20%7C%20ARM64%20%7C%20Windows-blue?style=flat)](https://github.com/apepsiii/gopilketos)
[![License](https://img.shields.io/badge/License-MIT-green?style=flat)](LICENSE)

**Gopilketos** adalah aplikasi pemilihan umum ketua & wakil ketua OSIS berbasis web yang dirancang khusus untuk institusi pendidikan (seperti SMK NIBA Business School). Dibangun menggunakan **Golang (Echo v5)**, **SQLite**, dan **Go HTML Templates** dengan antarmuka UI/UX modern, clean, dan minimalis.

Aplikasi ini menjamin asas **LUBER (Langsung, Umum, Bebas, Rahasia)** dengan sistem *anonymous ballot receipt*, verifikasi identitas instan via QR Scanner, presensi TPS real-time, serta notifikasi bukti voting otomatis via **WhatsApp Gateway (GOWA Engine)**.

---

## 🌟 Fitur Utama

### 🗳️ Bilik Suara Digital (Voter Module)
- **UI/UX Modern & Minimalis**: Tipografi *Plus Jakarta Sans*, desain responsif, ramah perangkat mobile, tablet, maupun layar kiosk.
- **Login Passwordless via QR Code**: Pemilih cukup memindai kartu pemilih menggunakan kamera perangkat atau memasukkan UUID manual.
- **Multi-Step Voting Flow**:
  1. **Langkah 1**: Penentuan Calon Ketua OSIS (Kandidat No. 1, 2, atau 3).
  2. **Langkah 2**: Penentuan Calon Wakil Ketua OSIS (Kandidat No. 1, 2, atau 3).
  3. **Langkah 3**: Konfirmasi pilihan pasangan terpilih sebelum surat suara dikirim.
  4. **Sukses**: Tanda terima suara digital.
- **Asas Rahasia (LUBER)**: Setiap surat suara yang masuk diberi kode anonim (*Masked Ballot Receipt* contoh: `BLT-A1B2C3D4`) yang memutus kaitan antara identitas pemilih dan pilihan politiknya di database.
- **Notifikasi WhatsApp Otomatis**: Integrasi pengiriman ucapan terima kasih dan konfirmasi partisipasi langsung ke nomor WhatsApp pemilih secara *asynchronous* (non-blocking).

### ⚙️ Panel Administrasi (Admin Module)
- **Dashboard Metrik Real-Time**:
  - Total DPT, Suara Masuk, Persentase Partisipasi Pemilih.
  - Grafik distribusi perolehan suara Calon Ketua & Wakil Ketua OSIS (Chart.js).
- **Manajemen Kandidat**:
  - Penomoran kandidat independen (Nomor Urut 01, 02, dan 03) untuk Ketua dan Wakil Ketua.
  - Upload foto profil calon dengan crop rasio seragam.
  - Penyuntingan visi, misi, dan program kerja unggulan.
- **Manajemen Daftar Pemilih Tetap (DPT)**:
  - Tambah, edit, dan hapus data pemilih perorangan.
  - **Import Massal**: Dukungan impor file **CSV** dan **Excel (.xlsx)** lengkap dengan validasi duplikasi.
  - Auto-generate UUID standar V4 untuk autentikasi unik.
- **Cetak Kartu Pemilih Siap Pakai**:
  - Format grid standar A4 (8 kartu presisi per lembar cetak).
  - Dilengkapi QR Code dinamis dan Barcode Code128 untuk scanner fisik/optik.
  - Filter cetak berdasarkan kelas/jurusan.
- **Sistem Presensi / Kehadiran TPS**:
  - Monitoring kehadiran DPT di bilik suara secara langsung.
  - Scanner presensi mandiri khusus panitia di meja registrasi (`/admin/attendance/scanner`).
- **Gateway Notifikasi WhatsApp (GOWA)**:
  - Form konfigurasi fleksibel: Endpoint Shim/Gowa, Device ID, Basic Auth credentials.
  - Live Testing: Uji coba pengiriman pesan test langsung dari dashboard admin.
  - Template pesan dinamis dengan variabel `{nama}`, `{kandidat_ketua}`, `{kandidat_wakil}`, `{waktu}`.
- **Audit Ledger & Maintenance**:
  - Audit log surat suara masuk (*immutable ledger*).
  - Backup data lengkap dalam format JSON.
  - Ekspor laporan rekapitulasi pemilihan dalam format CSV.
  - Fitur darurat: Reset data suara dengan verifikasi proteksi ketik teks *RESET*.

---

## 🏗️ Arsitektur Integrasi WhatsApp (GOWA)

Aplikasi terintegrasi dengan arsitektur **GOWA** (*Go WhatsApp API*) melalui layer **NIBA-Gowa Shim** pada server lokal:

```
┌──────────────────────┐
│   E-Voting App       │  (Go binary :8024)
│   (Gopilketos)       │  - Terima submit suara
│                      │  - Background goroutine notif
└──────────┬───────────┘
           │
           │ HTTP POST /api/whatsapp/send
           │ Header: Authorization: Basic <base64>
           │ Header: X-Device-Id: Pionir
           │ Body: {"phone": "628...", "message": "..."}
           ↓
┌──────────────────────┐
│   NIBA-Gowa Shim     │  (Python systemd :8054)
│   127.0.0.1:8054     │  - Translate header & endpoint
│                      │  - Map device Pionir → samsung
└──────────┬───────────┘
           │
           │ HTTP POST /send/message
           │ Header: X-Device-Id: samsung
           ↓
┌──────────────────────┐
│   Gowa Docker API    │  (Docker container :8053)
│   127.0.0.1:8053     │  - Kirim WA via socket WhatsApp
│                      │  - Return status & Message ID
└──────────────────────┘
```

---

## 🚀 Panduan Memulai Cepat (Quick Start)

### Kebutuhan Sistem
- **Go**: Versi 1.25 atau lebih baru
- **C Compiler**: GCC / MinGW (diperlukan untuk driver `go-sqlite3` CGO)
- **Git**

### 1. Menjalankan di Lingkungan Lokal (Windows / Linux)

```bash
# Clone repository
git clone https://github.com/apepsiii/gopilketos.git
cd gopilketos

# Download dependensi
go mod tidy

# Jalankan server
go run main.go
```

Setelah server aktif, akses melalui browser:
- **Halaman Pemilih**: [http://localhost:8024](http://localhost:8024)
- **Panel Admin**: [http://localhost:8024/admin/login](http://localhost:8024/admin/login)
  - **Username Default**: `admin`
  - **Password Default**: `admin123`

---

## 🐧 Kompilasi & Deploy untuk Platform ARM (Armbian 64-bit)

Aplikasi telah dioptimalkan untuk berjalan di mini-PC / Single Board Computer (SBC) seperti **Orange Pi / Raspberry Pi / Armbian Server** (`192.168.1.101`).

### Opsi A: Build Langsung di Mesin Armbian (Paling Stabil)
```bash
# 1. SSH ke server Armbian
ssh armbian@192.168.1.101

# 2. Clone atau tarik repository
cd /opt/evoting
git pull origin master

# 3. Jalankan script build ARM64 otomatis
chmod +x build_arm.sh
./build_arm.sh
```

### Opsi B: Cross-Compile dari Mesin Linux/WSL x86_64
Script `build_arm.sh` otomatis mendeteksi arsitektur host dan menggunakan toolchain `aarch64-linux-gnu-gcc`:
```bash
sudo apt update && sudo apt install -y gcc-aarch64-linux-gnu
./build_arm.sh
```

### Konfigurasi Systemd Service (Linux/Armbian)
Buat file service systemd di `/etc/systemd/system/evoting.service`:
```ini
[Unit]
Description=Gopilketos E-Voting Application
After=network.target niba-gowa-shim.service

[Service]
Type=simple
User=root
WorkingDirectory=/opt/evoting
ExecStart=/opt/evoting/pilketos
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
```

Aktifkan service:
```bash
sudo systemctl daemon-reload
sudo systemctl enable evoting
sudo systemctl restart evoting
sudo systemctl status evoting
```

---

## 🐳 Docker Deployment

Untuk deployment menggunakan container Docker:

```bash
# Build & jalankan dengan docker-compose
docker-compose up -d --build

# Periksa log aplikasi
docker-compose logs -f pilketos
```

---

## 📁 Struktur Direktori

```
gopilketos/
├── main.go                     # Entry point & inisialisasi route Echo
├── database/                   # Manajemen database SQLite & migration
│   ├── sqlite.go               # Inisialisasi koneksi WAL mode & pool
│   └── migrate.go              # Auto-migrasi skema & data initialization
├── handlers/                   # Controller / HTTP Request Handlers
│   ├── admin.go                # Manajemen dashboard, kandidat, settings, logs
│   ├── candidate_api.go        # JSON REST API profil kandidat
│   ├── landing.go              # Halaman depan informasi pemilihan
│   ├── scanner.go              # Bilik scanner QR Code voter
│   ├── submit_vote.go          # Validasi & penyimpanan suara pemilih
│   └── vote.go                 # Step 1, Step 2, & Konfirmasi bilik suara
├── models/                     # Deklarasi tipe data & entitas model Go
├── services/                   # Service layer eksternal
│   ├── whatsapp.go             # Client GOWA / Shim WhatsApp Gateway
│   └── onesender.go            # Backward-compatibility wrapper
├── views/                      # Template Go HTML (Tailwind CSS)
│   ├── admin/                  # Tampilan dashboard, kandidat, voters, DPT, settings
│   ├── layouts/                # Master layout, navbar, dan sidebar admin
│   └── voter/                  # Landing, scanner, bilik vote step 1 & 2, success
├── public/                     # Aset statis (CSS, JS, SVG, foto kandidat)
│   ├── css/
│   ├── js/
│   └── uploads/                # Direktori penyimpanan foto kandidat
├── docs/                       # Dokumentasi spesifikasi & integrasi
│   ├── gowa-integration.md     # Arsitektur & kontrak API WhatsApp
│   └── dbScheme.md             # Dokumentasi skema tabel database
├── build_arm.sh                # Skrip kompilasi otomatis target ARM64 Linux
├── config.docker.yaml          # Template konfigurasi container Docker
├── docker-compose.yml          # Konfigurasi orkestrasi container Docker
├── Dockerfile                  # Multi-stage Docker build file
└── go.mod                      # Dependensi module Go
```

---

## ⚙️ Variabel Lingkungan & Konfigurasi

Aplikasi dapat dikonfigurasi melalui file `config.yaml` atau Environment Variables:

| Variabel | Default | Keterangan |
|---|---|---|
| `PORT` | `8024` | Port HTTP server |
| `DB_PATH` | `database/evoting.db` | Lokasi file SQLite |
| `ADMIN_USER` | `admin` | Username panel admin |
| `ADMIN_PASS` | `admin123` | Password panel admin |

Contoh `config.yaml`:
```yaml
app_name: "Pilketos E-Voting SMK NIBA"
port: "8024"
domain: "vote.smkniba.sch.id"
admin_user: "admin"
admin_pass: "KataSandiKuat2026!"
db_path: "database/evoting.db"
```

---

## 🔒 Keamanan & Prinsip Kerahasiaan (LUBER)

1. **Anonymous Ballots**: Data pemilih (`voters.uuid`) tidak disimpan berelasi langsung dengan tabel `votes`. Tabel suara hanya menyimpan token acak `BLT-XXXXXX` sehingga siapa memilih siapa tidak dapat ditelusuri kembali (*Zero Traceable Voting*).
2. **Double Vote Prevention**: Sistem validasi transaksi tingkat database (`BEGIN TRANSACTION`) memastikan pemilih berstatus `has_voted = 1` ditolak seketika jika mencoba memilih kembali.
3. **Session Hardening**: Cookie sesi admin dilindungi flag `HttpOnly`, `SameSite=Strict`, dan ditandatangani dengan *HMAC-SHA256*.
4. **Rate Limiting**: Proteksi *brute-force* login panel admin dengan penguncian otomatis (*lockout*) setelah 5 kali percobaan gagal.

---

## 📄 Lisensi

Proyek ini dirilis di bawah lisensi [MIT License](LICENSE).

---

## 👥 Kontributor & Pengembang

Dikembangkan untuk **SMK NIBA Business School E-Voting System**.  
Repository: [https://github.com/apepsiii/gopilketos](https://github.com/apepsiii/gopilketos)