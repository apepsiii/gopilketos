# Docker Deployment Guide - Pilketos E-Voting

## Prerequisites

- Docker Engine 20.10+
- Docker Compose v2.0+
- Portainer (untuk deployment di VPS)

---

## 1. Persiapan Project

### 1.1 File yang Dibutuhkan

Pastikan semua file ini ada di root project:

```
go_pilketos/
├── Dockerfile              # Build instructions
├── docker-compose.yml      # Container orchestration
├── config.docker.yaml      # Konfigurasi untuk Docker
├── Makefile                # Shortcut commands
├── .dockerignore           # Exclude files
├── go.mod                  # Go modules
├── main.go                 # Entry point
├── views/                  # HTML templates
├── public/                 # Static assets
├── database/               # SQLite database
└── handlers/               # HTTP handlers
```

### 1.2 Struktur Konfigurasi Docker

**config.docker.yaml** (sudah dibuat):

```yaml
app_name: "Pilketos E-Voting"
port: "8024"
domain: ""
install_dir: ""
admin_user: "admin"
admin_pass: "admin123"       # GANTI DI VPS!
db_path: "database/evoting.db"
```

**docker-compose.yml** (sudah dibuat):

```yaml
version: '3.8'

services:
  pilketos:
    build:
      context: .
      dockerfile: Dockerfile
    container_name: pilketos
    ports:
      - "8024:8024"
    volumes:
      - pilketos_data:/app/database
      - pilketos_uploads:/app/public/uploads
    environment:
      - TZ=Asia/Jakarta
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "wget", "-q", "--spider", "http://localhost:8024/"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 10s

volumes:
  pilketos_data:
  pilketos_uploads:
```

---

## 2. Build & Run Lokal

### 2.1 Build Docker Image

```bash
# Dari root project
docker build -t pilketos:latest .
```

### 2.2 Jalankan dengan Docker Compose

```bash
# Start container
docker-compose up -d

# Lihat status
docker-compose ps

# Lihat logs
docker-compose logs -f

# Stop container
docker-compose down
```

### 2.3 Menggunakan Makefile (Shortcut)

```bash
make docker-build    # Build image
make docker-up        # Start container
make docker-logs      # Lihat logs
make docker-down      # Stop container
make docker-shell     # Masuk ke container shell
make docker-restart   # Restart container
make docker-clean     # Hapus container + volumes
```

### 2.4 Verifikasi

Buka browser: `http://localhost:8024`

Login admin: `http://localhost:8024/admin/login`
- Username: `admin`
- Password: `admin123`

---

## 3. Deployment ke VPS dengan Portainer

### 3.1 Setup Portainer di VPS

Jalankan Portainer di VPS (satu kali saja):

```bash
# Install Portainer
docker volume create portainer_data

docker run -d \
  --name portainer \
  -p 9000:9000 \
  -p 8000:8000 \
  --restart always \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v portainer_data:/data \
  portainer/portainer-ce:latest
```

Akses Portainer: `http://IP-VPS:9000`

### 3.2 Buat Stack di Portainer

#### Langkah 1: Login ke Portainer

1. Buka browser: `http://IP-VPS:9000`
2. Buat admin user baru
3. Pilih "Local" environment

#### Langkah 2: Buat Stack Baru

1. Klik **Stacks** di sidebar kiri
2. Klik **Add stack**

#### Langkah 3: Pilih Build Method

Pilih **Web editor** atau **Repository** (terserah preferensi)

#### Langkah 4: Paste docker-compose.yml

**Option A: Web Editor**

Paste isi `docker-compose.yml` ke web editor:

```yaml
version: '3.8'

services:
  pilketos:
    build:
      context: .
      dockerfile: Dockerfile
    container_name: pilketos
    ports:
      - "8024:8024"
    volumes:
      - pilketos_data:/app/database
      - pilketos_uploads:/app/public/uploads
    environment:
      - TZ=Asia/Jakarta
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "wget", "-q", "--spider", "http://localhost:8024/"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 10s

volumes:
  pilketos_data:
  pilketos_uploads:
```

**Option B: Git Repository**

1. Push project ke GitHub/GitLab
2. Pilih **Repository**
3. Masukkan URL repository
4. Pilih branch `master` atau `main`
5. Build context: `/`

#### Langkah 5: Buat config.yaml di Portainer

Sebelum deploy, kita perlu buat config.yaml:

1. Di sidebar Portainer, klik **Configs**
2. Klik **Add config**
3. Nama: `pilketos-config`
4. Paste config berikut:

```yaml
app_name: "Pilketos E-Voting"
port: "8024"
domain: ""
install_dir: ""
admin_user: "admin"
admin_pass: "GANTI_PASSWORD_BARU"    # PENTING: GANTI!
db_path: "database/evoting.db"
```

5. Klik **Create config**

#### Langkah 6: Modifikasi docker-compose.yml di Portainer

Karena config harus digunakan dari Portainer Configs, modifikasi compose:

```yaml
version: '3.8'

services:
  pilketos:
    build:
      context: .
      dockerfile: Dockerfile
    container_name: pilketos
    ports:
      - "8024:8024"
    volumes:
      - pilketos_data:/app/database
      - pilketos_uploads:/app/public/uploads
    configs:
      - source: pilketos-config
        target: /app/config.yaml
    environment:
      - TZ=Asia/Jakarta
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "wget", "-q", "--spider", "http://localhost:8024/"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 10s

configs:
  pilketos-config:
    external: true

volumes:
  pilketos_data:
  pilketos_uploads:
```

#### Langkah 7: Deploy

1. Beri nama stack, contoh: `pilketos`
2. Klik **Deploy the stack**

### 3.3 Troubleshooting Deploy di Portainer

**Error: Build Context**

Pastikan project structure benar. Portainer butuh akses ke:
- `Dockerfile`
- `docker-compose.yml`
- `main.go`
- `views/`
- `public/`
- `handlers/`
- `database/`
- `models/`
- `services/`

**Error: Config Not Found**

Pastikan config `pilketos-config` sudah dibuat di Portainer sebelum deploy stack.

---

## 4. Upload Project ke GitHub ( untuk Portainer Git Repository Method)

### 4.1 Inisialisasi Git (jika belum)

```bash
git init
git add .
git commit -m "Initial commit with Docker support"
```

### 4.2 Buat Repository di GitHub

1. Buka https://github.com/new
2. Nama repository: `pilketos`
3. Public/Private sesuai kebutuhan
4. Klik **Create repository**

### 4.3 Push ke GitHub

```bash
git remote add origin https://github.com/USERNAME/pilketos.git
git branch -M main
git push -u origin main
```

### 4.4 Update Portainer Stack

Setelah push, di Portainer:
1. Edit stack
2. Pilih **Repository**
3. Masukkan: `https://github.com/USERNAME/pilketos.git`
4. Branch: `main`
5. Klik **Update stack**

---

## 5. Maintenance

### 5.1 Backup Data

Data tersimpan di Docker volumes:

```bash
# Backup database
docker compose exec pilketos sh -c "cat database/evoting.db" > evoting_backup.db

# Atau backup整个 volume
docker run --rm -v pilketos_pilketos_data:/data -v $(pwd):/backup alpine tar czf /backup/pilketos_data.tar.gz -C /data .
```

### 5.2 Restore Data

```bash
# Stop container
docker-compose down

# Restore volume
docker run --rm -v pilketos_pilketos_data:/data -v $(pwd):/backup alpine tar xzf /backup/pilketos_data.tar.gz -C /data

# Start container
docker-compose up -d
```

### 5.3 Update Aplikasi

```bash
# Pull kode terbaru dari Git
git pull origin main

# Rebuild dan restart
docker-compose up -d --build
```

### 5.4 Reset Database

```bash
# Hapus volume (PERHATIAN: semua data hilang!)
docker-compose down -v

# Restart (database baru akan dibuat otomatis)
docker-compose up -d
```

---

## 6. Checklist Sebelum Production

- [ ] Ganti `admin_pass` di config.yaml
- [ ] Ganti `admin_user` jika perlu
- [ ] Set `domain` jika pakai subdomain
- [ ] Setup SSL/HTTPS (dengan Nginx/Caddy reverse proxy)
- [ ] Enable firewall (allow port 8024)
- [ ] Backup strategy terencana
- [ ] Monitor logs secara berkala

---

## 7. Quick Reference

| Command | Keterangan |
|---------|------------|
| `docker build -t pilketos .` | Build image |
| `docker-compose up -d` | Start container |
| `docker-compose down` | Stop container |
| `docker-compose logs -f` | Lihat logs |
| `docker-compose exec pilketos sh` | Shell ke container |
| `docker-compose restart` | Restart container |
| `docker-compose down -v` | Stop + hapus volumes |
| `make docker-*` | Use Makefile shortcuts |

---

## 8. Portainer Stack File (Ready to Use)

Simpan ini sebagai `stack.yml` untuk import langsung ke Portainer:

```yaml
version: '3.8'

services:
  pilketos:
    build:
      context: .
      dockerfile: Dockerfile
    container_name: pilketos
    ports:
      - "8024:8024"
    volumes:
      - pilketos_data:/app/database
      - pilketos_uploads:/app/public/uploads
    environment:
      - TZ=Asia/Jakarta
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "wget", "-q", "--spider", "http://localhost:8024/"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 10s

volumes:
  pilketos_data:
  pilketos_uploads:
```

**Catatan:** Untuk production, paste isi di atas ke Portainer Web Editor dan buat config `pilketos-config` terlebih dahulu.
