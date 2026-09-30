# 🗺️ Roadmap & Rekomendasi Pengembangan Gopilketos

Dokumen ini merangkum inisiatif strategis dan roadmap pengembangan fitur **Gopilketos (Sistem E-Voting & Portal Pemilihan OSIS SMK NIBA Business School)** dengan fokus pada peningkatan penyampaian informasi, ketercapaian program sekolah, transparansi demokrasi, serta keterlibatan aktif siswa.

---

## 🎯 Tujuan Strategis

1. **Efektivitas Informasi**: Memastikan setiap siswa dan warga sekolah memahami visi, misi, serta program kerja calon pemimpin OSIS dan agenda resmi sekolah.
2. **Tingkat Partisipasi Maksimal**: Mendorong pemilih pemula hadir di TPS dan menyalurkan hak suara melalui pendekatan teknologi interaktif.
3. **Transparansi & Akuntabilitas**: Mewujudkan proses pemilihan yang LUBER JURDIL dengan keterbukaan data rekapitulasi yang tetap menjaga kerahasiaan pilihan individu.
4. **Keterlibatan Dua Arah (Two-Way Engagement)**: Memberikan ruang aspirasi bagi siswa untuk menyampaikan harapan dan masukan kepada sekolah dan pengurus OSIS.

---

## 📋 Daftar Rencana Peningkatan Fitur

### 1. 📅 Timeline Tahapan Pemilihan & Kalender Kegiatan Sekolah
* **Deskripsi**: Menampilkan infografis interaktif tahapan pemilihan serta kalender agenda besar sekolah di landing page dan portal pemilih.
* **Komponen Fitur**:
  - **Tahapan Pilketos**: Sosialisasi & Pendaftaran Calon $\rightarrow$ Verifikasi Berkas $\rightarrow$ Kampanye & Debat Terbuka $\rightarrow$ Hari Pemilihan (TPS) $\rightarrow$ Sidang Pleno $\rightarrow$ Pelantikan OSIS.
  - **Agenda Besar Sekolah**: Classmeeting, Peringatan Hari Besar Nasional/Keagamaan, Latihan Dasar Kepemimpinan (LDKS), dan Ujian.
* **Manfaat**: Siswa dan guru dapat memantau secara pasti progres pesta demokrasi dan jadwal penting sekolah.

---

### 2. 📢 Optimalisasi WhatsApp Gateway (GOWA Engine)
* **Deskripsi**: Memanfaatkan infrastruktur GOWA yang telah terintegrasi untuk sosialisasi massal dan asisten bot informasi.
* **Komponen Fitur**:
  - **Broadcast Pengingat H-1 & Hari-H**: Mengirimkan notifikasi pengingat otomatis ke nomor WhatsApp siswa yang belum hadir di TPS untuk memaksimalkan kehadiran.
  - **Ringkasan Visi-Misi ke WhatsApp**: Pengiriman kartu digital ringkas visi-misi paslon ke siswa/wali kelas sebelum masa pemungutan suara.
  - **Chatbot Interaktif Siswa**: Fitur auto-reply jika siswa mengirim pesan seperti `!kandidat`, `!jadwal`, atau `!lokasi-tps`.
* **Manfaat**: Informasi menjangkau saku siswa secara langsung tanpa harus menunggu mereka membuka web browser.

---

### 3. 🎬 Video Teaser & Orasi Debat Paslon
* **Deskripsi**: Integrasi konten video singkat (YouTube Shorts, Reels, atau link rekaman) pada profil kandidat.
* **Komponen Fitur**:
  - Kolom URL video profil/orasi pada dashboard admin kandidat.
  - Pemutar video modal atau kartu preview interaktif di halaman detail kandidat.
  - Unduhan dokumen PDF Program Kerja Lengkap per divisi paslon.
* **Manfaat**: Generasi Z lebih cepat mencerna gagasan melalui format audio visual dibandingkan teks panjang, sehingga kualitas pemilih meningkat.

---

### 4. 💬 Kotak Aspirasi & Suara Siswa (Student Voice Wall)
* **Deskripsi**: Saluran aspirasi digital bagi siswa untuk menyampaikan saran, ide kegiatan, atau evaluasi bagi sekolah dan kepengurusan OSIS baru.
* **Komponen Fitur**:
  - Form pengiriman aspirasi (bisa anonim atau dengan identitas kelas).
  - Panel moderasi di dashboard admin untuk menyetujui pesan yang layak tampil.
  - *Wall of Voice* / *Ticker Aspirasi* interaktif di halaman depan.
* **Manfaat**: Membangun budaya kepedulian dan rasa memiliki (*sense of belonging*) siswa terhadap kemajuan sekolah.

---

### 5. 📊 Live Quick Count Terbuka (Pasca Penutupan TPS)
* **Deskripsi**: Visualisasi hasil rekapitulasi suara terbuka yang dapat diaktifkan oleh admin setelah TPS resmi ditutup.
* **Komponen Fitur**:
  - Sakelar kontrol *Publish Live Count* di menu Pengaturan Admin.
  - Grafik batang dan donat perolehan suara paslon dengan animasi real-time.
  - Total partisipasi per kelas dan persentase suara sah.
* **Manfaat**: Menghilangkan kecurigaan manipulasi hasil dan menciptakan euforia perayaan pesta demokrasi sekolah yang sehat.

---

### 6. 🤳 E-Twibbon & Kartu Bukti "Saya Sudah Memilih"
* **Deskripsi**: Generator kartu digital atau Twibbon otomatis setelah siswa selesai menyalurkan suara di bilik suara.
* **Komponen Fitur**:
  - Pada halaman sukses pemungutan suara (`/vote/success`), disediakan tombol *Download Kartu Digital*.
  - Generator canvas gambar bertuliskan *"Saya Sudah Menggunakan Hak Suara di Pilketos SMK NIBA"*.
  - Tombol *Share to WhatsApp Story / Instagram*.
* **Manfaat**: Menularkan antusiasme positif ke siswa lain (*viral peer-effect*) agar berbondong-bondong datang ke TPS.

---

### 7. 🏛️ Direktori Ekstrakurikuler & Organisasi (Student Hub)
* **Deskripsi**: Halaman direktori seluruh ekstrakurikuler dan organisasi binaan OSIS SMK NIBA.
* **Komponen Fitur**:
  - Profil ekskul (Pramuka, Paskibra, PMR, IT Club, Futsal, Seni Musik, Rohis, dll).
  - Nama pembina, ketua ekskul, jadwal latihan rutin, dan foto kegiatan prestasi.
  - Link formulir pendaftaran ekskul bagi siswa baru.
* **Manfaat**: Membantu siswa menyalurkan minat bakat dan memastikan program pengembangan karakter sekolah tercapai maksimal.

---

## 🗓️ Rencana Eksekusi Bertahap (Phased Milestones)

| Fase | Fokus Utama | Fitur yang Diterapkan | Estimasi Efek |
|---|---|---|---|
| **Fase 1** | *Awareness & Schedule* | • Timeline Tahapan Pemilihan di Beranda<br>• Link Video Orasi Kandidat (YouTube Embed)<br>• Broadcast Pengingat H-1 via GOWA | Siswa memahami jadwal dan mengenal kandidat lebih dini. |
| **Fase 2** | *Engagement & Partisipasi* | • E-Twibbon / Kartu Digital "Saya Sudah Memilih"<br>• Kotak Aspirasi Siswa & Moderasi Admin<br>• Filter Lanjutan Demisioner & Alumni | Partisipasi meningkat dan keterlibatan aktif siswa terbangun. |
| **Fase 3** | *Transparansi & Student Hub* | • Sakelar Live Quick Count Terbuka Pasca TPS Tutup<br>• Direktori Ekstrakurikuler & Organisasi Sekolah<br>• Asisten Chatbot WA Otomatis | Transparansi 100% dan wadah terpadu kegiatan kesiswaan. |

---

## 🛠️ Pedoman Teknis & Kompatibilitas
- **Arsitektur**: Semua fitur baru dibangun di atas arsitektur Go (Echo framework) dengan query SQLite yang dioptimalkan (`WAL mode`).
- **Antarmuka**: Mempertahankan standar desain bersih, minimalis, dan modern menggunakan Tailwind CSS serta iconografi Google Material Symbols.
- **Keamanan Data**: Fitur interaktif (kotak aspirasi / twibbon) dirancang tanpa melanggar prinsip kerahasiaan suara pemilih (*Zero Traceable Voting*).
