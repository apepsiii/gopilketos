document.addEventListener('DOMContentLoaded', function() {
    const statusDiv = document.getElementById('scan-status');
    const cameraSelectContainer = document.getElementById('camera-select-container');
    const cameraSelect = document.getElementById('camera-select');
    const uuidInput = document.getElementById('uuid-input');
    const validateBtn = document.getElementById('validate-btn');
    const qrReaderDiv = document.getElementById('qr-reader');

    console.log('[Scanner] Initializing...');

    if (location.protocol !== 'https:' && location.hostname !== 'localhost' && location.hostname !== '127.0.0.1') {
        statusDiv.textContent = 'Peringatan: Kamera memerlukan koneksi HTTPS';
        statusDiv.className = 'text-center text-sm text-yellow-600 mb-4';
        console.warn('[Scanner] Not using HTTPS - camera may not work');
    }

    if (typeof Html5Qrcode === 'undefined') {
        statusDiv.textContent = 'Library scanner tidak termuat. Gunakan input manual.';
        statusDiv.className = 'text-center text-sm text-red-600 mb-4';
        console.error('[Scanner] Html5Qrcode library not loaded');
        qrReaderDiv.innerHTML = '<div class="flex items-center justify-center h-full text-neutral-400"><p>Scanner tidak tersedia</p></div>';
        return;
    }

    console.log('[Scanner] Html5Qrcode library loaded');

    function normalizeUUID(value) {
        if (!value) return '';
        value = value.trim();
        if (value.toLowerCase().startsWith('uuid:')) {
            return value.split(':')[1].trim();
        }
        try {
            const url = new URL(value);
            return url.searchParams.get('uuid') || url.searchParams.get('id') || value;
        } catch (e) {
            return value;
        }
    }

    async function validateAndRedirect(uuid) {
        uuid = normalizeUUID(uuid);
        if (!uuid) {
            statusDiv.textContent = 'UUID wajib diisi';
            statusDiv.className = 'text-center text-sm text-red-600 mb-4';
            return;
        }
        statusDiv.textContent = 'Memvalidasi UUID...';
        statusDiv.className = 'text-center text-sm text-blue-600 mb-4';

        try {
            const res = await fetch('/validate-uuid', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ uuid })
            });
            const data = await res.json();
            if (res.ok && data.status === 'ok') {
                statusDiv.textContent = 'UUID valid, mengarahkan ke halaman voting...';
                statusDiv.className = 'text-center text-sm text-green-600 mb-4';
                setTimeout(() => window.location.href = '/vote?uuid=' + encodeURIComponent(uuid), 800);
            } else {
                statusDiv.textContent = data.error || 'UUID tidak valid';
                statusDiv.className = 'text-center text-sm text-red-600 mb-4';
            }
        } catch (err) {
            console.error('[Scanner] Validation error:', err);
            statusDiv.textContent = 'Terjadi kesalahan koneksi';
            statusDiv.className = 'text-center text-sm text-red-600 mb-4';
        }
    }

    validateBtn.addEventListener('click', function() {
        validateAndRedirect(uuidInput.value);
    });

    uuidInput.addEventListener('keypress', function(e) {
        if (e.key === 'Enter') {
            validateAndRedirect(uuidInput.value);
        }
    });

    let qrScanner = null;
    let isScanning = false;

    async function stopScanner() {
        if (qrScanner && isScanning) {
            try {
                await qrScanner.stop();
                qrScanner.clear();
                console.log('[Scanner] Scanner stopped');
            } catch (e) {
                console.warn('[Scanner] Error stopping scanner:', e);
            }
            isScanning = false;
        }
    }

    async function startScanner(deviceId) {
        console.log('[Scanner] Starting scanner with device:', deviceId);
        
        await stopScanner();

        qrReaderDiv.innerHTML = '';

        qrScanner = new Html5Qrcode('qr-reader');

        const config = {
            fps: 10,
            qrbox: { width: 250, height: 250 },
            aspectRatio: 1.0
        };

        try {
            statusDiv.textContent = 'Memulai kamera...';
            statusDiv.className = 'text-center text-sm text-blue-600 mb-4';

            await qrScanner.start(
                deviceId || { facingMode: 'environment' },
                config,
                async (decodedText, decodedResult) => {
                    console.log('[Scanner] QR Code detected:', decodedText);
                    statusDiv.textContent = 'QR Code terdeteksi, memvalidasi...';
                    statusDiv.className = 'text-center text-sm text-blue-600 mb-4';
                    await stopScanner();
                    await validateAndRedirect(decodedText);
                },
                (errorMessage) => {
                    // Silent ignore scan errors
                }
            );

            isScanning = true;
            statusDiv.textContent = 'Scanner aktif - arahkan ke QR Code';
            statusDiv.className = 'text-center text-sm text-green-600 mb-4';
            console.log('[Scanner] Scanner started successfully');
        } catch (err) {
            console.error('[Scanner] Failed to start scanner:', err);
            statusDiv.textContent = 'Gagal memulai kamera: ' + err.message;
            statusDiv.className = 'text-center text-sm text-red-600 mb-4';
            qrReaderDiv.innerHTML = '<div class="flex items-center justify-center h-full text-neutral-400 p-8 text-center"><p>Gagal mengakses kamera. Gunakan input manual di bawah.</p></div>';
        }
    }

    async function initCamera() {
        console.log('[Scanner] Requesting camera access...');
        
        try {
            const devices = await Html5Qrcode.getCameras();
            console.log('[Scanner] Available cameras:', devices);

            if (!devices || devices.length === 0) {
                statusDiv.textContent = 'Tidak ada kamera terdeteksi. Gunakan input manual.';
                statusDiv.className = 'text-center text-sm text-yellow-600 mb-4';
                qrReaderDiv.innerHTML = '<div class="flex items-center justify-center h-full text-neutral-400 p-8 text-center"><p>Kamera tidak tersedia</p></div>';
                return;
            }

            if (devices.length > 1) {
                cameraSelectContainer.classList.remove('hidden');
                cameraSelect.innerHTML = '';
                
                devices.forEach((camera, index) => {
                    const option = document.createElement('option');
                    option.value = camera.id;
                    option.textContent = camera.label || 'Kamera ' + (index + 1);
                    cameraSelect.appendChild(option);
                });

                cameraSelect.addEventListener('change', function() {
                    console.log('[Scanner] Camera changed to:', cameraSelect.value);
                    startScanner(cameraSelect.value);
                });
            }

            const backCamera = devices.find(d => 
                d.label.toLowerCase().includes('back') || 
                d.label.toLowerCase().includes('rear') ||
                d.label.toLowerCase().includes('environment')
            );
            
            const defaultCamera = backCamera ? backCamera.id : devices[devices.length - 1].id;
            console.log('[Scanner] Using default camera:', defaultCamera);
            
            await startScanner(defaultCamera);
        } catch (err) {
            console.error('[Scanner] Camera access error:', err);
            
            let errorMsg = 'Gagal mengakses kamera: ' + err.message;
            if (err.name === 'NotAllowedError' || err.message.includes('Permission')) {
                errorMsg = 'Izin kamera ditolak. Silakan izinkan akses kamera di browser Anda.';
            } else if (err.name === 'NotFoundError') {
                errorMsg = 'Kamera tidak ditemukan di perangkat ini.';
            } else if (err.name === 'NotReadableError') {
                errorMsg = 'Kamera sedang digunakan aplikasi lain.';
            } else if (err.name === 'OverconstrainedError') {
                errorMsg = 'Kamera tidak mendukung resolusi yang diperlukan.';
            }
            
            statusDiv.textContent = errorMsg;
            statusDiv.className = 'text-center text-sm text-red-600 mb-4';
            qrReaderDiv.innerHTML = '<div class="flex items-center justify-center h-full text-neutral-400 p-8 text-center"><p>' + errorMsg + '</p></div>';
        }
    }

    initCamera();

    window.addEventListener('beforeunload', function() {
        stopScanner();
    });
});