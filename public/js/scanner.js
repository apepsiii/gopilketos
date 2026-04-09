// scanner.js
// Anda bisa integrasikan html5-qrcode atau library lain sesuai kebutuhan
// Contoh simulasi input manual untuk demo

document.addEventListener('DOMContentLoaded', function() {
    const statusDiv = document.getElementById('scan-status');
    const qrDiv = document.getElementById('qr-reader');

    // Simulasi input manual UUID
    const input = document.createElement('input');
    input.type = 'text';
    input.placeholder = 'Tempel UUID di sini untuk simulasi';
    qrDiv.appendChild(input);

    const btn = document.createElement('button');
    btn.textContent = 'Validasi UUID';
    qrDiv.appendChild(btn);

    btn.onclick = async function() {
        const uuid = input.value.trim();
        if (!uuid) {
            statusDiv.textContent = 'UUID wajib diisi';
            return;
        }
        statusDiv.textContent = 'Memvalidasi UUID...';
        const res = await fetch('/validate-uuid', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ uuid })
        });
        const data = await res.json();
        if (data.status === 'ok') {
            statusDiv.textContent = 'UUID valid, mengarahkan ke halaman voting...';
            setTimeout(() => window.location.href = '/vote?uuid=' + encodeURIComponent(uuid), 1000);
        } else {
            statusDiv.textContent = data.error || 'UUID tidak valid';
        }
    };
});
