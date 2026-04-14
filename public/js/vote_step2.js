document.addEventListener('DOMContentLoaded', async () => {
    const params = new URLSearchParams(window.location.search);
    const uuid = params.get('uuid');
    const chairmanId = params.get('chairman_id');
    if (!uuid || !chairmanId) {
        return window.location.href = '/vote?uuid=' + encodeURIComponent(uuid || '');
    }

    const scannerStatus = document.getElementById('vote-scan-status');
    const candidateScanner = document.getElementById('vote-scanner');

    const list = document.getElementById('candidates-list');
    list.innerHTML = '<p>Memuat kandidat wakil...</p>';

    try {
        const response = await fetch('/api/candidates?position=VICE_CHAIRMAN');
        const candidates = await response.json();
        if (!response.ok) {
            list.innerHTML = '<p class="error">Gagal memuat kandidat.</p>';
            return;
        }
        if (candidates.length === 0) {
            list.innerHTML = '<p>Tidak ada kandidat wakil tersedia.</p>';
            return;
        }
        const candidateMap = {};
        list.innerHTML = candidates.map(candidate => {
            candidateMap[candidate.id] = candidate;
            return `
            <div class="group relative bg-white rounded-2xl p-6 shadow-sm border border-slate-200 hover:shadow-md hover:border-secondary/30 transition-all duration-300 cursor-pointer"
                 onclick="window.location.href='/vote/confirm?uuid=${encodeURIComponent(uuid)}&chairman_id=${encodeURIComponent(chairmanId)}&vice_id=${candidate.id}'">
                <div class="flex flex-col items-center text-center gap-4">
                    <div class="relative">
                        <img src="${candidate.photo_url || '/static/images/default-profile.svg'}" alt="${candidate.name}" 
                             class="w-28 h-28 rounded-full object-cover border-4 border-slate-100 group-hover:border-secondary/50 transition-colors" />
                        <div class="absolute -bottom-2 -right-2 w-8 h-8 bg-secondary rounded-full flex items-center justify-center shadow-lg">
                            <span class="material-symbols-outlined text-white text-sm">how_to_vote</span>
                        </div>
                    </div>
                    <div class="space-y-1">
                        <h3 class="text-xl font-bold text-slate-900">${candidate.name}</h3>
                        <p class="text-sm font-medium text-slate-500">Kelas ${candidate.class_name}</p>
                    </div>
                    <div class="w-full pt-4 border-t border-slate-100 space-y-3">
                        <div class="text-left">
                            <p class="text-xs font-bold uppercase tracking-wider text-secondary mb-1">Visi</p>
                            <p class="text-sm text-slate-600 line-clamp-2">${candidate.vision}</p>
                        </div>
                        <div class="text-left">
                            <p class="text-xs font-bold uppercase tracking-wider text-secondary mb-1">Misi</p>
                            <p class="text-sm text-slate-600 line-clamp-2">${candidate.mission}</p>
                        </div>
                    </div>
                    <div class="w-full mt-2">
                        <span class="inline-flex items-center gap-2 px-4 py-2 bg-secondary text-white rounded-lg text-sm font-semibold group-hover:bg-secondary/90 transition-colors">
                            <span class="material-symbols-outlined text-sm">check_circle</span>
                            Pilih Wakil Ini
                        </span>
                    </div>
                </div>
            </div>
        `;
        }).join('');

        if (window.Html5QrcodeScanner && candidateScanner) {
            const scanner = new Html5QrcodeScanner('vote-scanner', { fps: 10, qrbox: 250 });
            scanner.render((decodedText) => {
                const id = parseInt(decodedText.replace(/[^0-9]/g, ''), 10);
                candidateMap[id] = candidateMap[id] || null;
                if (candidateMap[id]) {
                    scanner.clear().catch(() => {});
                    if (scannerStatus) {
                        scannerStatus.textContent = `Kandidat terdeteksi: ${candidateMap[id].name}. Mengarahkan...`;
                    }
                    window.location.href = `/vote/confirm?uuid=${encodeURIComponent(uuid)}&chairman_id=${encodeURIComponent(chairmanId)}&vice_id=${encodeURIComponent(id)}`;
                } else if (scannerStatus) {
                    scannerStatus.textContent = 'QR kandidat tidak valid. Pastikan Anda memindai QR kode wakil.';
                }
            }, () => {
                // ignore scan errors
            });
        } else if (scannerStatus) {
            scannerStatus.textContent = 'Scanner kamera tidak tersedia. Pilih kandidat secara manual.';
        }
    } catch (error) {
        list.innerHTML = '<p class="error">Terjadi kesalahan saat memuat kandidat.</p>';
    }
});
