# Arsitektur Integrasi WhatsApp untuk E-Voting App

**Aplikasi:** E-Voting Standalone (Go)  
**Use Case:** Kirim ucapan terima kasih otomatis setelah user voting  
**Infrastruktur:** Server Armbian 192.168.1.101  
**Tanggal:** 29 September 2026

---

## 1. Arsitektur Overview

```
┌──────────────────────┐
│   E-Voting App       │  (Go binary, port :xxxx)
│   (Standalone)       │  - Terima POST vote
│                      │  - Simpan ke DB
│                      │  - Trigger WA notifikasi
└──────────┬───────────┘
           │
           │ HTTP POST /api/whatsapp/send
           │ Header: X-Device-Id: Pionir
           │ Body: {phone, message}
           ↓
┌──────────────────────┐
│  NIBA-Gowa Shim      │  (Python, :8054)
│  127.0.0.1:8054      │  - Translate API lama → baru
│                      │  - Map Pionir → samsung
└──────────┬───────────┘
           │
           │ HTTP POST /send/message
           │ Header: X-Device-Id: samsung
           ↓
┌──────────────────────┐
│  Gowa WhatsApp API   │  (Docker, :8053)
│  127.0.0.1:8053      │  - Kirim WA via device samsung
│                      │  - Return status
└──────────────────────┘
```

---

## 2. Komponen Detail

### A. E-Voting App (Go)

**Lokasi:** `/opt/evoting/` (contoh)  
**Port:** Misal `:8080`  
**Database:** SQLite/PostgreSQL (sesuai kebutuhan)

**Table Schema (minimal):**

```sql
CREATE TABLE votes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    voter_phone TEXT NOT NULL,
    voter_name TEXT,
    candidate_id INTEGER,
    voted_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    wa_notif_sent BOOLEAN DEFAULT 0,
    wa_notif_status TEXT
);
```

### B. WhatsApp Integration Layer

**Endpoint Shim:** `http://127.0.0.1:8054/api/whatsapp/send`  
**Auth:** Basic `admin:PutihAbu123!`  
**Device:** `Pionir` (akan dimapping ke samsung)

---

## 3. Flow Voting + Notifikasi

```
1. User submit voting
   └─> POST /vote {candidate_id, voter_phone, voter_name}

2. E-Voting App
   ├─> Validasi (belum voting sebelumnya)
   ├─> Simpan vote ke database
   └─> Trigger sendWhatsAppNotification()

3. sendWhatsAppNotification()
   ├─> Format pesan ucapan terima kasih
   ├─> POST ke http://127.0.0.1:8054/api/whatsapp/send
   │   Header: Authorization: Basic YWRtaW46UHV0aWhBYnUxMjMh
   │   Header: X-Device-Id: Pionir
   │   Body: {
   │     "phone": "6281234567890",
   │     "message": "Terima kasih {nama}, suara Anda telah tercatat..."
   │   }
   └─> Update votes.wa_notif_sent = 1

4. Shim (127.0.0.1:8054)
   ├─> Terima request
   ├─> Map Pionir → samsung
   ├─> Forward ke Gowa: POST 127.0.0.1:8053/send/message
   └─> Return response ke E-Voting

5. Gowa (127.0.0.1:8053)
   ├─> Kirim WA via device samsung (6285158250766)
   └─> Return status {success: true, message_id: ...}

6. E-Voting App
   └─> Log status ke votes.wa_notif_status
```

---

## 4. Code Implementation (Go)

### A. Struct & Config

```go
package main

import (
    "bytes"
    "encoding/base64"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "time"
)

// Config WA
const (
    ShimURL     = "http://127.0.0.1:8054/api/whatsapp/send"
    ShimUser    = "admin"
    ShimPass    = "PutihAbu123!"
    ShimDevice  = "Pionir"
    Timeout     = 30 * time.Second
)

// Request payload ke Shim
type WhatsAppRequest struct {
    Phone   string `json:"phone"`
    Message string `json:"message"`
}

// Response dari Shim/Gowa
type WhatsAppResponse struct {
    Success   bool   `json:"success"`
    MessageID string `json:"message_id,omitempty"`
    Error     string `json:"error,omitempty"`
}
```

### B. Fungsi Send WhatsApp

```go
// sendWhatsAppNotification kirim WA via Shim
func sendWhatsAppNotification(phone, message string) (*WhatsAppResponse, error) {
    // Format phone (pastikan format 62xxx)
    if phone[0] == '0' {
        phone = "62" + phone[1:]
    }

    // Payload
    payload := WhatsAppRequest{
        Phone:   phone,
        Message: message,
    }

    jsonData, err := json.Marshal(payload)
    if err != nil {
        return nil, fmt.Errorf("marshal JSON gagal: %w", err)
    }

    // HTTP Request
    req, err := http.NewRequest("POST", ShimURL, bytes.NewBuffer(jsonData))
    if err != nil {
        return nil, fmt.Errorf("create request gagal: %w", err)
    }

    // Headers
    auth := base64.StdEncoding.EncodeToString([]byte(ShimUser + ":" + ShimPass))
    req.Header.Set("Authorization", "Basic "+auth)
    req.Header.Set("X-Device-Id", ShimDevice)
    req.Header.Set("Content-Type", "application/json")

    // Send
    client := &http.Client{Timeout: Timeout}
    resp, err := client.Do(req)
    if err != nil {
        return nil, fmt.Errorf("HTTP request gagal: %w", err)
    }
    defer resp.Body.Close()

    // Parse response
    body, _ := io.ReadAll(resp.Body)

    var waResp WhatsAppResponse
    if err := json.Unmarshal(body, &waResp); err != nil {
        return nil, fmt.Errorf("parse response gagal (status %d): %s", resp.StatusCode, body)
    }

    if resp.StatusCode != 200 {
        return &waResp, fmt.Errorf("WA API error (status %d): %s", resp.StatusCode, waResp.Error)
    }

    return &waResp, nil
}
```

### C. Handler Vote dengan WA Notifikasi

```go
// voteHandler handle POST /vote
func voteHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    var req struct {
        CandidateID int    `json:"candidate_id"`
        VoterPhone  string `json:"voter_phone"`
        VoterName   string `json:"voter_name"`
    }

    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid JSON", http.StatusBadRequest)
        return
    }

    // 1. Validasi (cek sudah voting atau belum)
    // ... query database ...

    // 2. Simpan vote ke database
    result, err := db.Exec(`
        INSERT INTO votes (voter_phone, voter_name, candidate_id, voted_at)
        VALUES (?, ?, ?, ?)
    `, req.VoterPhone, req.VoterName, req.CandidateID, time.Now())

    if err != nil {
        http.Error(w, "Database error", http.StatusInternalServerError)
        return
    }

    voteID, _ := result.LastInsertId()

    // 3. Format pesan ucapan terima kasih
    message := fmt.Sprintf(
        "Terima kasih %s! 🙏\n\n"+
        "Suara Anda untuk Kandidat #%d telah tercatat pada %s.\n\n"+
        "Terima kasih atas partisipasi Anda dalam pemilihan ini.",
        req.VoterName,
        req.CandidateID,
        time.Now().Format("02 Jan 2006 15:04"),
    )

    // 4. Kirim WA (async agar tidak blocking)
    go func(id int64, phone, msg string) {
        waResp, err := sendWhatsAppNotification(phone, msg)

        status := "sent"
        if err != nil {
            status = "failed: " + err.Error()
        }

        // Update database
        db.Exec(`
            UPDATE votes
            SET wa_notif_sent = ?, wa_notif_status = ?
            WHERE id = ?
        `, err == nil, status, id)

        if err != nil {
            fmt.Printf("[WA] Vote ID %d error: %v\n", id, err)
        } else {
            fmt.Printf("[WA] Vote ID %d sent, msg_id: %s\n", id, waResp.MessageID)
        }
    }(voteID, req.VoterPhone, message)

    // 5. Response ke client
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]interface{}{
        "success": true,
        "message": "Vote berhasil tercatat",
        "vote_id": voteID,
    })
}
```

### D. Main App

```go
func main() {
    // Init database
    // db, _ = sql.Open("sqlite3", "./evoting.db")

    http.HandleFunc("/vote", voteHandler)
    http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte("OK"))
    })

    fmt.Println("E-Voting server running on :8080")
    http.ListenAndServe(":8080", nil)
}
```

---

## 5. API Contract

### Request (E-Voting → Shim)

```http
POST http://127.0.0.1:8054/api/whatsapp/send HTTP/1.1
Authorization: Basic YWRtaW46UHV0aWhBYnUxMjMh
X-Device-Id: Pionir
Content-Type: application/json

{
  "phone": "6281234567890",
  "message": "Terima kasih Budi! 🙏\n\nSuara Anda untuk Kandidat #2 telah tercatat..."
}
```

### Response (Shim → E-Voting)

**Success (200):**

```json
{
  "success": true,
  "message_id": "3EB0C6A5D83E123456"
}
```

**Error (4xx/5xx):**

```json
{
  "success": false,
  "error": "device not connected"
}
```

---

## 6. Deployment Steps

### A. Setup E-Voting App

```bash
# 1. Build binary (di laptop lokal)
GOOS=linux GOARCH=arm64 go build -o evoting_linux_arm64 .

# 2. Upload ke server
scp evoting_linux_arm64 armbian@192.168.1.101:/home/apep/migrasi/evoting/

# 3. Setup di server
ssh armbian@192.168.1.101
sudo mkdir -p /opt/evoting
sudo cp /home/apep/migrasi/evoting/evoting_linux_arm64 /opt/evoting/
sudo chmod +x /opt/evoting/evoting_linux_arm64

# 4. Create systemd service
sudo nano /etc/systemd/system/evoting.service
```

### B. Systemd Service File

```ini
[Unit]
Description=E-Voting Application with WhatsApp Integration
After=network.target niba-gowa-shim.service

[Service]
Type=simple
User=root
WorkingDirectory=/opt/evoting
ExecStart=/opt/evoting/evoting_linux_arm64
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
```

### C. Start Service

```bash
sudo systemctl daemon-reload
sudo systemctl enable evoting
sudo systemctl start evoting
sudo systemctl status evoting
```

---

## 7. Testing

### Test 1: Cek Shim Ready

```bash
curl -u admin:PutihAbu123! http://127.0.0.1:8054/health
# Expected: {"ok":true,"service":"niba-gowa-shim","gowa":"http://127.0.0.1:8053"}
```

### Test 2: Kirim WA Manual via Shim

```bash
curl -X POST http://127.0.0.1:8054/api/whatsapp/send \
  -u admin:PutihAbu123! \
  -H "X-Device-Id: Pionir" \
  -H "Content-Type: application/json" \
  -d '{
    "phone": "6285158250766",
    "message": "Test notifikasi e-voting: Terima kasih telah voting!"
  }'
```

### Test 3: Submit Vote

```bash
curl -X POST http://127.0.0.1:8080/vote \
  -H "Content-Type: application/json" \
  -d '{
    "candidate_id": 2,
    "voter_phone": "081234567890",
    "voter_name": "Budi Santoso"
  }'
```

### Test 4: Cek Log

```bash
# Log e-voting app
sudo journalctl -u evoting -f

# Log shim
sudo journalctl -u niba-gowa-shim -f

# Log Gowa (Docker)
docker logs -f whatsapp-api
```

---

## 8. Monitoring & Troubleshooting

### Cek Status Semua Layanan

```bash
# Status e-voting
systemctl status evoting

# Status shim
systemctl status niba-gowa-shim

# Status Gowa
docker ps | grep whatsapp-api
curl -u admin:PutihAbu123! http://192.168.1.101:8053/app/devices
```

### Database Check

```bash
# Cek votes yang notifikasinya gagal
sqlite3 /opt/evoting/evoting.db "
  SELECT id, voter_phone, wa_notif_sent, wa_notif_status
  FROM votes
  WHERE wa_notif_sent = 0
  ORDER BY id DESC
  LIMIT 10;
"
```

### Retry Failed Notifications (Optional)

Buat cron job atau endpoint `/retry-wa-notifications` yang query votes dengan `wa_notif_sent = 0` dan retry kirim.

---

## 9. Error Handling

| Error                    | Penyebab           | Solusi                                  |
| ------------------------ | ------------------ | --------------------------------------- |
| Connection refused :8054 | Shim mati          | `sudo systemctl restart niba-gowa-shim` |
| Connection refused :8053 | Gowa mati          | `docker restart whatsapp-api`           |
| 401 Unauthorized         | Auth salah         | Cek username/password di code           |
| Device not connected     | WA logout          | Scan QR di http://wa.gxa.my.id:8053     |
| Invalid phone format     | Format nomor salah | Pastikan format 62xxx (bukan 08xxx)     |
| Timeout                  | Network lambat     | Naikkan timeout di code (>30s)          |

---

## 10. Template Pesan (Customizable)

```go
// Di code e-voting, sesuaikan template pesan
const MessageTemplate = `Terima kasih %s! 🙏

Suara Anda untuk Kandidat #%d telah tercatat pada %s.

Partisipasi Anda sangat berarti untuk keberhasilan pemilihan ini.

Salam,
Panitia E-Voting`
```

Atau pakai template dengan data kandidat:

```go
message := fmt.Sprintf(
    "Terima kasih %s! 🙏\n\n"+
    "✅ Vote Anda untuk:\n"+
    "   Kandidat: %s\n"+
    "   Nomor Urut: %d\n"+
    "   Waktu: %s\n\n"+
    "Terima kasih atas partisipasi Anda!",
    voterName,
    candidateName,
    candidateNumber,
    time.Now().Format("02 Jan 2006 15:04 WIB"),
)
```

---

## 11. Scalability & Best Practices

### A. Queue System (Optional untuk High Traffic)

Jika voting ramai (ribuan user), pertimbangkan queue system:

```
E-Voting App → Redis Queue → Worker → Shim → Gowa
```

Worker bisa pakai library `go-workers` atau simple goroutine pool.

### B. Rate Limiting

Gowa punya limit kirim WA per detik. Jika perlu kirim massal:

- Tambahkan delay antar request (100-200ms)
- Batch processing dengan retry logic

### C. Logging

Simpan semua aktivitas WA:

```sql
CREATE TABLE wa_logs (
    id INTEGER PRIMARY KEY,
    vote_id INTEGER,
    phone TEXT,
    message TEXT,
    status TEXT,
    response TEXT,
    sent_at DATETIME
);
```

---

## 12. Security Checklist

- [ ] Auth credentials (admin:PutihAbu123!) di environment variable, bukan hardcode
- [ ] Validasi input phone number (regex 628xxx, max 15 digit)
- [ ] Rate limiting endpoint /vote (max 1 vote per phone)
- [ ] HTTPS jika app public (reverse proxy nginx + Let's Encrypt)
- [ ] Sanitasi message content (escape special chars)
- [ ] Log sensitive data (phone, message) dengan masking

---

## 13. Infrastruktur Existing

**Server:** 192.168.1.101 (Armbian, 2GB RAM)  
**Services Running:**

- Gowa (Docker :8053) ✅
- Shim (systemd :8054) ✅
- NIBA (:8073) ✅
- Portainer, UptimeKuma, Cloudflare Tunnel ✅

**Available Ports:** 8080-8090 (untuk e-voting app)

---

## 14. Next Steps

1. ✅ Baca dokumentasi ini
2. ⬜ Setup database schema e-voting
3. ⬜ Implement code Go sesuai contoh
4. ⬜ Build binary ARM64
5. ⬜ Deploy ke server (systemd)
6. ⬜ Testing flow lengkap
7. ⬜ Setup monitoring/alerting
8. ⬜ Production deployment

---

**Dibuat:** 29 September 2026  
**Untuk:** Mas Sae (Muhamad Sae)  
**Oleh:** Kiro Assistant

---

## FAQ

**Q: Bisa pakai device wa-broadcast (smk niba) daripada samsung (Pionir)?**  
A: Bisa. Ganti `X-Device-Id: smk niba` di request. Shim akan mapping ke `wa-broadcast`.

**Q: Berapa lama delay kirim WA?**  
A: ~1-3 detik dari trigger sampai WA terkirim (tergantung network).

**Q: Bisa kirim gambar/media?**  
A: Bisa. Ganti endpoint ke `/api/whatsapp/send-image`, tambahkan field `image_url` di payload.

**Q: Kalau Gowa mati saat voting?**  
A: Notifikasi gagal, tersimpan di DB (wa_notif_sent=0). Bisa retry manual/otomatis nanti.

**Q: Limit kirim WA per hari?**  
A: Tidak ada limit hard dari Gowa. WhatsApp sendiri bisa ban akun jika spam (>500 msg/hari ke nomor beda). Untuk voting sekolah (<1000 voters) aman.
