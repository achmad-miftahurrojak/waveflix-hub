from curl_cffi import requests

try:

    r_stream = requests.get("http://localhost:8000/stream?media=movie&id=278")
    data = r_stream.json()
    url = data['sources'][0]['file']
    print("Fresh URL:", url)

    headers = {
        'User-Agent': 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/137.0.0.0 Safari/537.36',
        'Origin': 'https://vidlink.pro',
        'Referer': 'https://vidlink.pro/'
    }
    r = requests.get(url, headers=headers, impersonate="chrome110", stream=True)
    print("Status:", r.status_code)
except Exception as e:
    print("Error:", e)
