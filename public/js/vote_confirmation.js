document.addEventListener('DOMContentLoaded', async () => {
    const params = new URLSearchParams(window.location.search);
    const uuid = params.get('uuid');
    const chairmanId = params.get('chairman_id');
    const viceId = params.get('vice_id');

    if (!uuid || !chairmanId || !viceId) {
        return window.location.href = '/vote?uuid=' + encodeURIComponent(uuid || '');
    }

    const summary = document.getElementById('summary');
    const submitButton = document.getElementById('submit-btn');
    const restartButton = document.getElementById('restart-btn');

    summary.innerHTML = '<p>Memuat ringkasan pilihan...</p>';

    try {
        const [chairmanResponse, viceResponse] = await Promise.all([
            fetch('/api/candidate?id=' + encodeURIComponent(chairmanId)),
            fetch('/api/candidate?id=' + encodeURIComponent(viceId)),
        ]);
        if (!chairmanResponse.ok || !viceResponse.ok) {
            summary.innerHTML = '<p class="error">Tidak dapat memuat ringkasan pilihan.</p>';
            return;
        }

        const chairman = await chairmanResponse.json();
        const vice = await viceResponse.json();

        summary.innerHTML = `
            <div class="candidate-card selected">
                <h3>Ketua Terpilih</h3>
                <p><strong>${chairman.name}</strong> (${chairman.class_name})</p>
                <p>${chairman.vision}</p>
            </div>
            <div class="candidate-card selected">
                <h3>Wakil Terpilih</h3>
                <p><strong>${vice.name}</strong> (${vice.class_name})</p>
                <p>${vice.vision}</p>
            </div>
            <p class="warning">Sekali dikirim, pilihan tidak bisa diubah.</p>
        `;

        submitButton.onclick = async () => {
            submitButton.disabled = true;
            submitButton.textContent = 'Mengirim...';
            const response = await fetch('/submit-vote', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ uuid, chairman_id: parseInt(chairmanId, 10), vice_chairman_id: parseInt(viceId, 10) })
            });
            const result = await response.json();
            if (response.ok && result.status === 'success') {
                window.location.href = '/vote/success';
            } else {
                summary.innerHTML += `<p class="error">${result.message || 'Gagal menyimpan vote.'}</p>`;
                submitButton.disabled = false;
                submitButton.textContent = 'Kirim Pilihan';
            }
        };

        restartButton.onclick = () => {
            window.location.href = '/vote?uuid=' + encodeURIComponent(uuid);
        };
    } catch (error) {
        summary.innerHTML = '<p class="error">Terjadi kesalahan saat memproses ringkasan.</p>';
    }
});
