document.addEventListener('DOMContentLoaded', async () => {
    const params = new URLSearchParams(window.location.search);
    const uuid = params.get('uuid');
    if (!uuid) {
        return window.location.href = '/scanner';
    }

    const list = document.getElementById('candidates-list');
    list.innerHTML = '<p>Memuat kandidat ketua...</p>';

    try {
        const response = await fetch('/api/candidates?position=CHAIRMAN');
        const candidates = await response.json();
        if (!response.ok) {
            list.innerHTML = '<p class="error">Gagal memuat kandidat.</p>';
            return;
        }
        if (candidates.length === 0) {
            list.innerHTML = '<p>Tidak ada kandidat ketua tersedia.</p>';
            return;
        }
        list.innerHTML = candidates.map(candidate => `
            <div class="candidate-card">
                <img src="${candidate.photo_url || '/static/images/default-profile.svg'}" alt="${candidate.name}" class="candidate-photo" />
                <h3>${candidate.name}</h3>
                <p>Kelas: ${candidate.class_name}</p>
                <p><strong>Visi:</strong> ${candidate.vision}</p>
                <p><strong>Misi:</strong> ${candidate.mission}</p>
                <p><strong>Program:</strong> ${candidate.program}</p>
                <a class="btn" href="/vote/step2?uuid=${encodeURIComponent(uuid)}&chairman_id=${candidate.id}">Pilih Ketua Ini</a>
            </div>
        `).join('');
    } catch (error) {
        list.innerHTML = '<p class="error">Terjadi kesalahan saat memuat kandidat.</p>';
    }
});
