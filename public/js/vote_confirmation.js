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
            <div class="p-6 rounded-xl bg-primary-container/30 border border-primary/20">
                <p class="text-xs font-bold uppercase tracking-widest text-primary mb-3">Ketua Terpilih</p>
                <div class="flex items-center gap-4">
                    <img src="${chairman.photo_url || '/static/images/default-profile.svg'}" alt="${chairman.name}" class="w-16 h-16 rounded-full object-cover border-2 border-primary/30" />
                    <div>
                        <p class="font-bold text-lg text-on-surface">${chairman.name}</p>
                        <p class="text-sm text-on-surface-variant">${chairman.class_name}</p>
                    </div>
                </div>
                <div class="mt-4 p-4 bg-white/50 rounded-lg">
                    <p class="text-xs font-semibold text-on-surface-variant uppercase tracking-wider mb-1">Visi</p>
                    <p class="text-sm text-on-surface">${chairman.vision}</p>
                </div>
            </div>
            <div class="p-6 rounded-xl bg-secondary-container/30 border border-secondary/20">
                <p class="text-xs font-bold uppercase tracking-widest text-secondary mb-3">Wakil Ketua Terpilih</p>
                <div class="flex items-center gap-4">
                    <img src="${vice.photo_url || '/static/images/default-profile.svg'}" alt="${vice.name}" class="w-16 h-16 rounded-full object-cover border-2 border-secondary/30" />
                    <div>
                        <p class="font-bold text-lg text-on-surface">${vice.name}</p>
                        <p class="text-sm text-on-surface-variant">${vice.class_name}</p>
                    </div>
                </div>
                <div class="mt-4 p-4 bg-white/50 rounded-lg">
                    <p class="text-xs font-semibold text-on-surface-variant uppercase tracking-wider mb-1">Visi</p>
                    <p class="text-sm text-on-surface">${vice.vision}</p>
                </div>
            </div>
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
