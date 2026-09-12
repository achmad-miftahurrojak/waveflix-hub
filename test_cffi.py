from curl_cffi import requests
import sys

url = sys.argv[1]
headers = {
    'User-Agent': 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/137.0.0.0 Safari/537.36',
    'Origin': 'https://vidlink.pro',
    'Referer': 'https://vidlink.pro/'
}
try:
    r = requests.get(url, headers=headers, impersonate="chrome110", stream=True)
    print(r.status_code)
    print(r.headers)
except Exception as e:
    print(f"Error: {e}")
