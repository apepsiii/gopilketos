Untuk menghasilkan UI yang rapi, modern, dan sesuai dengan spesifikasi menggunakan AI UI Generator (seperti Google Project IDX terintegrasi Gemini, v0 by Vercel, atau Claude Artifacts), Anda perlu memberikan prompt yang sangat spesifik mengenai *framework*, tata letak (*layout*), dan komponen yang digunakan.

Karena Anda akan menggunakan **Shadcn UI** dan **Tailwind CSS**, berikut adalah panduan *prompt step-by-step* dalam bahasa Inggris yang bisa Anda *copy-paste* secara berurutan ke *tools* AI tersebut. 

Prompt ini sudah saya sesuaikan agar langsung menampilkan *placeholder* yang relevan dengan lingkungan sekolah Anda.

---

### Step 1: System Context & Admin Dashboard Layout
**Gunakan prompt ini pertama kali untuk membangun fondasi kerangka Admin dan menetapkan *rules* bagi AI.**

> **Prompt 1:**
> "Act as an expert Frontend Developer. We are building an E-Voting Web Application for a school election (OSIS). I want you to use React, Next.js, Tailwind CSS, and Shadcn UI components. 
> 
> First, generate the layout for the Admin Dashboard.
> - **Sidebar:** Include navigation links for 'Dashboard', 'Candidates', 'Voters (DPT)', and 'Audit Logs'.
> - **Header:** Display the title 'SMK NIBA E-Voting Admin' with an admin profile dropdown.
> - **Main Content Area (Dashboard):** Create a dashboard view with 3 summary cards: 'Total Voters', 'Votes Cast', and 'Participation Rate'. Below the cards, use a bar chart (you can use Recharts placeholder) showing the real-time voting results for 'Chairman Candidates'. Ensure the design is clean, enterprise-grade, and responsive."

### Step 2: Voter Landing Page
**Setelah dashboard terbentuk, buat halaman depan untuk para pemilih.**

> **Prompt 2:**
> "Now, generate the public Landing Page for the voters.
> - **Navbar:** Simple header with 'SMK NIBA Business School E-Voting' logo/text and a prominent 'Login to Vote' button.
> - **Hero Section:** An announcement banner with the text 'Welcome to the OSIS Election. Please review the candidates before casting your vote.'
> - **Content:** Create two sections side-by-side or stacked. 
>   1. 'Chairman Candidates': Display responsive cards for 3 candidates. Each card should have a photo placeholder, Name, Class, Vision, and Mission.
>   2. 'Vice-Chairman Candidates': Similar card layout for 3 candidates.
> Use Shadcn Card components to make it look elegant."

### Step 3: Passwordless Login Page (QR/Barcode)
**Halaman untuk siswa melakukan pemindaian.**

> **Prompt 3:**
> "Generate the Passwordless Login Page for voters. 
> This page should be highly minimalist and mobile-first. 
> - Center a large square placeholder meant for a QR/Barcode Camera Scanner. 
> - Below the scanner placeholder, add an animated scanning effect or a text stating 'Position your QR Code inside the frame to login securely.'
> - Provide a 'Back to Home' ghost button. 
> The UI should feel fast and secure."

### Step 4: The Voting Flow (Step 1 & 2)
**Halaman inti untuk proses pencoblosan digital.**

> **Prompt 4:**
> "Generate the interactive Voting Page. We need a multi-step form UI.
> - **Top Bar:** Show a progress indicator (Step 1 of 3: Choose Chairman).
> - **Main Content:** Display a grid of Shadcn Cards representing the candidates. 
> - **Interaction:** When a card is clicked, it should have a thick primary border or a checkmark indicating it is 'selected'. 
> - **Bottom Action:** A sticky bottom bar with a disabled 'Next' button that becomes active only after a candidate is selected.
> Make it look highly engaging, as this is the core experience for the students."

### Step 5: Confirmation & Submission Page
**Halaman final untuk validasi pilihan (Asas Luber Jurdil).**

> **Prompt 5:**
> "Finally, generate the Confirmation Page (Step 3).
> - Display a summary card: 'Your Selections'. Inside, show the chosen Chairman and Vice-Chairman with their small avatars and names.
> - Below the summary, add a warning text: 'Once submitted, your vote is final and cannot be changed.'
> - **Action Buttons:**
>   1. A large, prominent 'Submit Vote' button (primary color).
>   2. A 'Cancel and Restart' button (outline/destructive variant) that allows them to re-pick.
> Make sure the layout is centered, clear, and prevents accidental clicks."

---

Dengan memisahkan prompt menjadi 5 langkah di atas, AI akan memiliki fokus yang lebih baik pada setiap halamannya sehingga komponen Shadcn UI yang dihasilkan tidak bertabrakan dan kodenya lebih mudah Anda implementasikan ke dalam *frontend web framework* yang Anda pilih.

Apakah Anda juga ingin saya buatkan draf struktur folder proyek untuk *frontend* ini agar lebih mudah diintegrasikan dengan Golang Echo di *backend*?