import cloudscraper
scraper = cloudscraper.create_scraper()
html = scraper.get('https://open.spotify.com/embed/playlist/37i9dQZF1DXcBWIGoYBM5M').text
print(html[:500])
with open('embed.html', 'w', encoding='utf-8') as f:
    f.write(html)
import re
import urllib.parse
# Find track titles and artists in the HTML. 
# Usually, embed player contains a window.__INITIAL_STATE__ or __PRELOADED_STATE__
match = re.search(r'\"name\"\:\"(.*?)\"', html)
if match:
    print("Found name:", match.group(1))
