import asyncio
import inspect
from collections import defaultdict
_pendengar = defaultdict(list)
BERISIK = False

def dengar(event_name, fungsi):
    if fungsi not in _pendengar[event_name]:
        _pendengar[event_name].append(fungsi)
    return fungsi

def lupakan(event_name, fungsi):
    if fungsi in _pendengar.get(event_name, []):
        _pendengar[event_name].remove(fungsi)

async def umumkan(event_name, **isi):
    """Publish event ke semua listeners. Return (total, errors)."""
    daftar = list(_pendengar.get(event_name, []))
    errors = 0
    if BERISIK:
        print(f'[acara] {event_name} -> {len(daftar)} pendengar | {isi}')
    for fungsi in daftar:
        try:
            hasil = fungsi(isi)
            if inspect.isawaitable(hasil):
                await hasil
        except asyncio.CancelledError:
            raise
        except Exception as e:
            errors += 1
            asal = getattr(fungsi, '__module__', '?')
            nama = getattr(fungsi, '__name__', '?')
            print(f"[acara] pendengar '{event_name}' ({nama} dari {asal}) error: {e}")
    return len(daftar), errors

def umumkan_nanti(event_name, **isi):
    try:
        asyncio.get_running_loop().create_task(umumkan(event_name, **isi))
    except RuntimeError:
        pass

def siapa_dengerin():
    return {event_name: [getattr(f, '__module__', '?') for f in daftar] for event_name, daftar in _pendengar.items() if daftar}