# Pilketos - OSIS E-Voting System

![Go Version](https://img.shields.io/badge/Go-1.25%2B-00ADD8?style=flat&logo=go)
![License](https://img.shields.io/badge/License-MIT-green?style=flat)

Aplikasi e-voting OSIS untuk SMK NIBA, dibangun dengan Golang, Echo v5, SQLite, dan Go HTML Templates. Mendukung dashboard admin, QR-based login pemilih, multi-step voting flow, dan notifikasi WhatsApp via OneSender.

---

## Fitur

### Admin Module
- Dashboard dengan metrics real-time
- Manajemen kandidat (CRUD dengan foto upload)
- Manajemen pemilih (DPT, UUID generation, import CSV)
- Kartu pemilih printable
- Scanner kehadiran (attendance)
- Audit logs dengan masked UUIDs
- Settings (announcement, OneSender config)
- Backup database & export report

### Voter Module
- Landing page dengan profil kandidat
- QR Code scanner untuk login passwordless
- Multi-step voting flow (Step 1 → Step 2 → Confirm → Success)
- Notifikasi WhatsApp setelah voting

### Technical
- Server-side rendering dengan Go HTML Templates
- SQLite database dengan auto-migration
- Docker-ready untuk deployment
- Responsive UI dengan TailwindCSS

---

## Quick Start

### Prerequisites

- Go 1.25+
- Docker (optional, untuk deployment)
- GCC toolchain (untuk SQLite CGO)

### Local Development

```bash
# Clone repository
git clone https://github.com/apepsiii/gopilketos.git
cd gopilketos

# Install dependencies
go mod tidy

# Run dengan air (hot reload)
go install github.com/cosmtrek/air@latest
air

# atau run langsung
go run main.go
```

Akses:
- Voter: `http://localhost:8024`
- Admin: `http://localhost:8024/admin/login`

Default login:
- Username: `admin`
- Password: `admin123`

---

## Docker Deployment

### Build & Run dengan Docker

```bash
# Build image
docker build -t pilketos:latest .

# Run container
docker run -d --name pilketos -p 8024:8024 -v pilketos_data:/app/database pilketos:latest

# atau dengan docker-compose
docker-compose up -d
```

### Deploy ke VPS dengan Portainer

1. Setup DNS A Record:
   ```
   Type: A
   Name: vote
   Value: IP_VPS
   ```

2. Clone di VPS:
   ```bash
   git clone https://github.com/apepsiii/gopilketos.git
   cd gopilketos
   ```

3. Generate SSL Certificate:
   ```bash
   # Stop container untuk port 80
   docker-compose down
   
   # Generate cert
   docker run --rm -v $(pwd)/data/certbot/conf:/etc/letsencrypt \
     -v $(pwd)/data/certbot/www:/var/www/certbot \
     -p 80:80 certbot/certbot certonly --standalone \
     -d vote.yourdomain.app --agree-tos --email your@email.com --no-eff-email
   ```

4. Deploy dengan SSL:
   ```bash
   docker-compose up -d
   ```

5. Akses: `https://vote.yourdomain.app`

---

## Folder Structure

```
gopilketos/
├── main.go                 # Entry point
├── handlers/               # HTTP request handlers
│   ├── admin.go            # Admin handlers
│   ├── vote.go             # Voting flow handlers
│   ├── scanner.go          # QR scanner handler
│   └── landing.go          # Landing page handler
│   └── submit_vote.go      # Vote submission handler
│   └── candidate_api.go    # Candidate API
├── database/               # SQLite connection & migration
│   ├── sqlite.go           # DB connection
│   └── migrate.go          # Schema migration
├── models/                 # Data models
├── services/               # External services
│   └── onesender.go        # WhatsApp notification
├── views/                  # HTML templates
│   ├── admin/              # Admin pages
│   ├── voter/              # Voter pages
│   └── layouts/            # Layout templates
├── public/                 # Static assets
│   ├── css/                # Stylesheets
│   ├── js/                 # JavaScript
│   ├── images/             # Static images
│   └── uploads/            # Uploaded files (candidate photos)
├── installer/              # Installer binary
├── docs/                   # Documentation
├── Dockerfile              # Docker build config
├── docker-compose.yml      # Docker orchestration
├── config.docker.yaml      # Docker config template
├── .air.toml               # Air hot reload config
├── Makefile                # Build shortcuts
└── go.mod                  # Go modules
```

---

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8024` | Server port |
| `DB_PATH` | `database/evoting.db` | SQLite database path |
| `ADMIN_USER` | `admin` | Admin username |
| `ADMIN_PASS` | `admin123` | Admin password |

### config.yaml (Optional)

Buat file `config.yaml` di root project atau `/opt/pilketos/`:

```yaml
app_name: "Pilketos E-Voting"
port: "8024"
domain: "vote.yourdomain.app"
admin_user: "admin"
admin_pass: "securepassword"
db_path: "database/evoting.db"
```

---

## API Endpoints

### Voter Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/` | Landing page |
| GET | `/scanner` | QR scanner page |
| POST | `/validate-uuid` | Validate UUID for login |
| GET | `/vote` | Vote step 1 |
| GET | `/vote/step2` | Vote step 2 |
| GET | `/vote/confirm` | Vote confirmation |
| POST | `/submit-vote` | Submit vote |
| GET | `/vote/success` | Success page |
| GET | `/api/candidates` | List candidates (JSON) |
| GET | `/api/candidate?id=X` | Get candidate detail (JSON) |

### Admin Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/admin/login` | Admin login page |
| POST | `/admin/login` | Admin login action |
| GET | `/admin` | Dashboard |
| GET | `/admin/candidates` | Candidates list |
| POST | `/admin/candidates` | Create candidate |
| POST | `/admin/candidates/:id` | Update candidate |
| POST | `/admin/candidates/:id/delete` | Delete candidate |
| GET | `/admin/voters` | Voters list (DPT) |
| POST | `/admin/voters/save` | Create voter |
| POST | `/admin/voters/import` | Import voters (CSV) |
| GET | `/admin/voters/cards` | Printable voter cards |
| GET | `/admin/attendance` | Attendance list |
| GET | `/admin/attendance/scanner` | Attendance scanner |
| POST | `/admin/attendance/mark` | Mark attendance |
| GET | `/admin/logs` | Audit logs |
| GET | `/admin/settings` | Settings page |
| POST | `/admin/settings/save` | Save settings |
| POST | `/admin/settings/reset-votes` | Reset all votes |
| GET | `/admin/settings/backup` | Download backup |
| GET | `/admin/settings/report` | Download report |
| GET | `/admin/logout` | Logout |

---

## Database Schema

### Tables

- `candidates` - Kandidat OSIS
- `voters` - Daftar Pemilih Tetap (DPT)
- `votes` - Record voting
- `audit_logs` - Audit trail
- `settings` - Application settings
- `attendance` - Attendance records

---

## Maintenance

### Backup Database

```bash
# Manual backup
docker exec pilketos cat database/evoting.db > backup_$(date +%Y%m%d).db

# Via admin panel
GET /admin/settings/backup
```

### Reset Votes

Via admin panel: `/admin/settings` → "Reset Votes"

### Update Application

```bash
git pull origin master
docker-compose down
docker-compose build --no-cache
docker-compose up -d
```

---

## Security Notes

- Ganti default password admin sebelum production
- Gunakan HTTPS untuk QR scanner (getUserMedia API)
- Backup database secara berkala
- Audit logs menyimpan masked UUID untuk anonymity

---

## License

MIT License

---

## Author

Developed for SMK NIBA E-Voting System

GitHub: [https://github.com/apepsiii/gopilketos](https://github.com/apepsiii/gopilketos)