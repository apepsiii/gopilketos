function escapeHTML(str) {
    if (!str) return '';
    return String(str)
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;')
        .replace(/'/g, '&#39;');
}

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
    const mobileSubmitButton = document.getElementById('mobile-submit-btn');
    const mobileRestartButton = document.getElementById('mobile-restart-btn');

    summary.innerHTML = '<p class="text-slate-400 text-center py-6">Memuat ringkasan pilihan...</p>';

    try {
        const [chairmanResponse, viceResponse] = await Promise.all([
            fetch('/api/candidate?id=' + encodeURIComponent(chairmanId)),
            fetch('/api/candidate?id=' + encodeURIComponent(viceId)),
        ]);
        if (!chairmanResponse.ok || !viceResponse.ok) {
            summary.innerHTML = '<p class="text-red-500 text-center py-4 font-semibold">Tidak dapat memuat ringkasan pilihan.</p>';
            return;
        }

        const chairman = await chairmanResponse.json();
        const vice = await viceResponse.json();

        summary.innerHTML = `
            <div class="p-6 rounded-2xl bg-blue-50/70 border border-blue-200/80">
                <div class="flex items-center justify-between mb-3">
                    <p class="text-xs font-bold uppercase tracking-wider text-blue-700">Calon Ketua OSIS</p>
                    <span class="px-2 py-0.5 rounded-lg bg-blue-100 text-blue-800 font-mono text-xs font-bold">No. ${chairman.candidate_number || 1}</span>
                </div>
                <div class="flex items-center gap-4">
                    <img src="${chairman.photo_url ? encodeURI(chairman.photo_url) : '/static/images/default-profile.svg'}" 
                         alt="${escapeHTML(chairman.name)}" class="w-16 h-16 rounded-2xl object-cover border-2 border-blue-300 shadow-sm" />
                    <div>
                        <p class="font-bold text-lg text-slate-900">${escapeHTML(chairman.name)}</p>
                        <p class="text-xs text-slate-500">${escapeHTML(chairman.class_name)}</p>
                    </div>
                </div>
                <div class="mt-4 p-3.5 bg-white rounded-xl border border-blue-100 shadow-xs">
                    <p class="text-[10px] font-bold text-slate-400 uppercase tracking-wider mb-1">Visi</p>
                    <p class="text-xs text-slate-700 line-clamp-2">${escapeHTML(chairman.vision || '-')}</p>
                </div>
            </div>

            <div class="p-6 rounded-2xl bg-purple-50/70 border border-purple-200/80">
                <div class="flex items-center justify-between mb-3">
                    <p class="text-xs font-bold uppercase tracking-wider text-purple-700">Calon Wakil Ketua OSIS</p>
                    <span class="px-2 py-0.5 rounded-lg bg-purple-100 text-purple-800 font-mono text-xs font-bold">No. ${vice.candidate_number || 1}</span>
                </div>
                <div class="flex items-center gap-4">
                    <img src="${vice.photo_url ? encodeURI(vice.photo_url) : '/static/images/default-profile.svg'}" 
                         alt="${escapeHTML(vice.name)}" class="w-16 h-16 rounded-2xl object-cover border-2 border-purple-300 shadow-sm" />
                    <div>
                        <p class="font-bold text-lg text-slate-900">${escapeHTML(vice.name)}</p>
                        <p class="text-xs text-slate-500">${escapeHTML(vice.class_name)}</p>
                    </div>
                </div>
                <div class="mt-4 p-3.5 bg-white rounded-xl border border-purple-100 shadow-xs">
                    <p class="text-[10px] font-bold text-slate-400 uppercase tracking-wider mb-1">Visi</p>
                    <p class="text-xs text-slate-700 line-clamp-2">${escapeHTML(vice.vision || '-')}</p>
                </div>
            </div>
        `;

        async function submitVote() {
            if (submitButton) {
                submitButton.disabled = true;
                submitButton.textContent = 'Mengirim Suara...';
            }
            if (mobileSubmitButton) {
                mobileSubmitButton.disabled = true;
                mobileSubmitButton.querySelector('span:last-child').textContent = 'Mengirim...';
            }

            try {
                const response = await fetch('/submit-vote', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({
                        uuid,
                        chairman_id: parseInt(chairmanId, 10),
                        vice_chairman_id: parseInt(viceId, 10)
                    })
                });
                const result = await response.json();
                if (response.ok && result.status === 'success') {
                    window.location.href = '/vote/success';
                } else {
                    alert(result.message || 'Gagal menyimpan suara.');
                    if (submitButton) {
                        submitButton.disabled = false;
                        submitButton.textContent = 'Kirim Suara';
                    }
                    if (mobileSubmitButton) {
                        mobileSubmitButton.disabled = false;
                        mobileSubmitButton.querySelector('span:last-child').textContent = 'Kirim Suara';
                    }
                }
            } catch (err) {
                alert('Terjadi kesalahan jaringan.');
                if (submitButton) {
                    submitButton.disabled = false;
                    submitButton.textContent = 'Kirim Suara';
                }
                if (mobileSubmitButton) {
                    mobileSubmitButton.disabled = false;
                    mobileSubmitButton.querySelector('span:last-child').textContent = 'Kirim Suara';
                }
            }
        }

        function restartVote() {
            if (confirm('Apakah Anda yakin ingin membatalkan dan mengulang pemilihan?')) {
                window.location.href = '/vote?uuid=' + encodeURIComponent(uuid);
            }
        }

        if (submitButton) submitButton.onclick = submitVote;
        if (mobileSubmitButton) mobileSubmitButton.onclick = submitVote;
        if (restartButton) restartButton.onclick = restartVote;
        if (mobileRestartButton) mobileRestartButton.onclick = restartVote;

    } catch (error) {
        summary.innerHTML = '<p class="text-red-500 text-center py-4 font-semibold">Terjadi kesalahan saat memproses data kandidat.</p>';
    }
});
