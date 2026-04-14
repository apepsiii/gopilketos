Pilihan untuk beralih ke arsitektur *monolith* dengan Custom HTML sangat masuk akal, Pak Apep. Pendekatan ini akan membuat proses *deployment* dan pemeliharaan aplikasi di lingkungan sekolah menjadi jauh lebih ringkas, karena semuanya berjalan di dalam satu *service* Golang.

Berikut adalah versi *update* dari Product Requirements Document (PRD) yang telah disesuaikan dengan arsitektur *monolith* dan penggunaan Go HTML Templates, khusus untuk diterapkan di SMK NIBA Business School Bogor.

---

# Product Requirements Document (PRD): E-Voting OSIS App (Monolith Version)

## 1. Ringkasan Produk (Product Overview)
Aplikasi berbasis web *monolith* untuk memfasilitasi pemilihan Ketua dan Wakil Ketua OSIS secara digital di lingkungan sekolah. Sistem dirancang untuk menggantikan pencoblosan kertas dengan pemindaian QR Code/Barcode via kamera *smartphone* atau *webcam*, memastikan proses yang lebih cepat, efisien, dan transparan, namun tetap memegang teguh asas Luber Jurdil (Langsung, Umum, Bebas, Rahasia, Jujur, dan Adil).

## 2. Tech Stack (Updated)
* **Arsitektur:** Monolith (Server-Side Rendering)
* **Backend & Router:** Golang dengan framework Echo v5.1.0
* **Database:** SQLite (mudah di-*backup* dan dipindah-pindah tanpa instalasi *server database* terpisah).
* **Frontend / View Engine:** Go HTML Templates bawaan standar (`html/template`).
* **Frontend Interactivity:** Vanilla JavaScript & Custom CSS.
* **QR Scanner Library:** Pustaka JS klien (misal: `html5-qrcode`) untuk pemindaian di *browser*.
* **Third-Party API:** OneSender (WhatsApp Gateway) menggunakan *goroutine* untuk notifikasi *blasting*.

## 3. Aktor & Peran (User Roles)
1.  **Admin (Panitia/KPU OSIS):** Memiliki akses ke halaman *dashboard backend* untuk manajemen data kandidat, data pemilih (DPT), mencetak kartu *login* QR, dan memantau hasil pemilihan serta log audit.
2.  **Voter (Siswa/Pemilih):** Mengakses halaman publik untuk melihat profil kandidat dan melakukan proses *voting* melalui antarmuka pemindai QR.

## 4. Fitur Utama (Core Features)

### A. Modul Admin (Server-Rendered Pages)
* **Dashboard Analytics:** * Menampilkan metrik utama: Total DPT, Suara Masuk, dan Persentase Partisipasi.
    * Menampilkan hasil pemilihan sementara (Ketua & Wakil) yang di-*render* langsung atau di- *fetch* ringan via JS ke *endpoint* internal.
* **Manajemen Kandidat (CRUD):**
    * Halaman HTML dengan form standar untuk Input/Edit/Hapus Calon Ketua dan Wakil Ketua OSIS (Nama, Kelas, Foto, Visi, Misi, Program Kerja).
    * Foto kandidat disimpan di lokal direktori `public/uploads/`.
* **Manajemen Pemilih (DPT):**
    * Fasilitas input data pemilih (Nama, Kelas, Nomor HP WA, UUID).
    * Pembuatan UUID unik per siswa di *backend* saat data diinput.
* **Cetak Kartu Pemilih (Printable View):**
    * Halaman HTML khusus yang dioptimalkan untuk cetak kertas A4 (menggunakan *CSS Print Media Queries*), menampilkan Kartu Login berisi instruksi dan QR Code/Barcode.
* **Audit Log (Transparansi):**
    * Tabel *log* yang memuat data *read-only* berisi UUID yang disamarkan (misal: `123e4567-****-****-****-************`), waktu *voting*, dan paslon yang dipilih. 
* **Pengaturan Publikasi:** Pengaturan teks pengumuman untuk halaman *landing page*.

### B. Modul Voter (Public HTML Pages)
* **Landing Page (`/`):**
    * Halaman statis dinamis yang menampilkan teks pengumuman.
    * Daftar profil lengkap Calon Ketua dan Wakil Ketua OSIS.
* **Scanner Login (`/scanner`):**
    * Halaman yang memuat *script* Vanilla JS (seperti `html5-qrcode`) untuk mengaktifkan kamera *device*.
    * Sistem membaca QR Code, mengirimkan UUID ke *backend* via AJAX/Fetch API.
    * Jika UUID *valid* dan belum mencoblos, arahkan (*redirect*) ke Halaman Pemilihan.
* **Alur Pemilihan (`/vote`):**
    * Form HTML *multi-step* (dikendalikan dengan Vanilla JS untuk menyembunyikan/menampilkan *section* form tanpa *reload* halaman).
    * **Step 1:** Pilih Ketua OSIS (klik *card* kandidat).
    * **Step 2:** Pilih Wakil Ketua OSIS.
    * **Step 3:** Rangkuman pilihan dengan tombol konfirmasi ("Kirim Pilihan" atau "Batalkan").
* **Post-Voting Action:**
    * Saat form di-*submit*, *backend* Golang mencatat suara ke database SQLite (memisahkan identitas siswa dengan log coblosan).
    * Sistem menandai UUID siswa menjadi "Sudah Memilih".
    * *Backend* memicu *goroutine* untuk mengirim WA ucapan terima kasih via API OneSender.
    * Siswa di-*redirect* ke halaman sukses (`/success`).

## 5. Struktur Direktori Utama
* `cmd/server/main.go` (Entry point Echo)
* `handlers/` (Logika *controller* yang mengembalikan `c.Render()`)
* `views/` (File `*.html` terbagi atas folder `admin`, `voter`, dan `layouts`)
* `public/` (File CSS, Vanilla JS, Logo sekolah, dan Foto Kandidat)
* `database/` (File `.db` SQLite)

## 6. Keamanan & Aturan Bisnis
* **Satu Pemilih, Satu Suara:** Validasi *backend* menolak UUID yang status `has_voted`-nya sudah bernilai `1` atau `true`.
* **Integritas Data (Luber Jurdil):** Tidak ada *foreign key* yang menghubungkan tabel identitas siswa (`voters`) dengan tabel rekap suara (`votes`). 
* **Akses Admin:** Rute `/admin/*` dilindungi dengan *middleware* otentikasi sederhana (seperti *session* berbasis *cookie* atau Basic Auth) agar hanya panitia KPU OSIS yang bisa masuk.

---

Dengan PRD yang sudah disesuaikan ini, fokus *development* Anda sekarang sepenuhnya ada di lingkungan Golang Echo, mengelola logika *routing*, *rendering template*, dan meracik antarmuka HTML/CSS yang ringan.


Pilihan yang sangat tepat. Menggunakan goroutine adalah salah satu keunggulan utama Golang untuk menangani task asinkron atau "fire-and-forget" semacam ini.

Dengan goroutine, siswa tidak perlu menatap layar loading selagi sistem menunggu balasan dari server OneSender. Pengalaman pengguna (user experience) di aplikasi frontend akan terasa sangat responsif dan instan.

Secara teknis di controller Echo Anda nanti, implementasinya akan terlihat sangat bersih, kurang lebih seperti ini:

func (h *VoteHandler) SubmitVote(c echo.Context) error {
    // 1. Parsing payload & Validasi UUID pemilih
    // 2. Simpan record pilihan ke database SQLite
    
    // 3. Jalankan proses kirim WA di background tanpa memblokir thread utama
    go func(phoneNumber string) {
        err := h.OneSenderClient.SendMessage(phoneNumber, "Terima kasih atas partisipasi Anda dalam pemilihan OSIS...")
        if err != nil {
            // Log error jika WA gagal terkirim, tapi tidak membatalkan vote
            log.Printf("Gagal mengirim WA ke %s: %v", phoneNumber, err)
        }
    }(voter.PhoneNumber)

    // 4. Langsung kembalikan response sukses ke Frontend
    return c.JSON(http.StatusOK, map[string]string{
        "status": "success",
        "message": "Pilihan berhasil disimpan",
    })
}