import json
import os
import shutil
import tempfile
from datetime import datetime

def baca(jalur, bawaan=None):
    """Read JSON file. Return (data, aman). aman=False jika file corrupt."""
    if bawaan is None:
        bawaan = {}
    try:
        with open(jalur, 'r', encoding='utf-8') as f:
            isi = f.read()
    except FileNotFoundError:
        return (bawaan, True)
    except OSError as e:
        print(f'[simpan] {jalur} ga kebuka: {e}')
        return (bawaan, False)
    if not isi.strip():
        return (bawaan, True)
    try:
        return (json.loads(isi), True)
    except (json.JSONDecodeError, ValueError) as e:
        selamat = f"{jalur}.rusak-{datetime.now().strftime('%Y%m%d-%H%M%S')}"
        try:
            shutil.copy2(jalur, selamat)
            print(f'[simpan] {jalur} RUSAK: {e}')
            print(f'[simpan] aslinya diselamatin ke {selamat}, JANGAN dihapus')
        except OSError as e2:
            print(f'[simpan] {jalur} RUSAK ({e}) dan gagal diselamatin: {e2}')
        return (bawaan, False)

def tulis(jalur, data, rapi=1):
    """Atomic write JSON pakai temp file + os.replace()."""
    folder = os.path.dirname(os.path.abspath(jalur)) or '.'
    sementara = None
    try:
        os.makedirs(folder, exist_ok=True)
        fd, sementara = tempfile.mkstemp(dir=folder, prefix='.tulis-', suffix='.tmp')
        with os.fdopen(fd, 'w', encoding='utf-8') as f:
            json.dump(data, f, ensure_ascii=False, indent=rapi)
            f.flush()
            os.fsync(f.fileno())
        os.replace(sementara, jalur)
        sementara = None
        return True
    except Exception as e:
        print(f'[simpan] gagal nulis {jalur}: {e}')
        return False
    finally:
        if sementara and os.path.exists(sementara):
            try:
                os.remove(sementara)
            except OSError:
                pass

def sapu_yang_rusak(folder=None, simpan_berapa=3, umur_hari=30):
    """Hapus file .rusak-* yang kelebihan atau ketuaan."""
    import glob
    import time
    folder = folder or os.path.dirname(os.path.abspath(__file__))
    semua = glob.glob(os.path.join(folder, '*.rusak-*'))
    if not semua:
        return 0
    per_asal = {}
    for jalur in semua:
        asal = jalur.split('.rusak-')[0]
        per_asal.setdefault(asal, []).append(jalur)
    batas_umur = time.time() - umur_hari * 86400
    dibuang = 0
    for _asal, daftar in per_asal.items():
        daftar.sort(key=lambda p: os.path.getmtime(p), reverse=True)
        for i, jalur in enumerate(daftar):
            terlalu_banyak = i >= simpan_berapa
            terlalu_tua = os.path.getmtime(jalur) < batas_umur
            if not (terlalu_banyak or terlalu_tua):
                continue
            try:
                os.remove(jalur)
                dibuang += 1
            except OSError as e:
                print(f'[simpan] gagal hapus {jalur}: {e}')
    if dibuang:
        print(f'[simpan] {dibuang} salinan .rusak lama dibuang')
    return dibuang

def baca_atau_berhenti(jalur, bawaan=None, nama=None):
    """Baca JSON, raise SystemExit jika file corrupt. Untuk data kritis."""
    isi, aman = baca(jalur, bawaan)
    if not aman:
        label = nama or jalur
        raise SystemExit(f'\n[simpan] BERHENTI. {label} rusak dan ga bisa dibaca.\n[simpan] Salinan aslinya ada di sebelahnya, berakhiran .rusak-*\n[simpan] Benerin dulu filenya, atau kalau emang mau mulai dari\n[simpan] nol, hapus {jalur} terus nyalain lagi.\n')
    return isi