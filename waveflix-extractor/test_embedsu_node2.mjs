import fetch from "node-fetch";

async function embedSuGetVideo(id) {
  const url = "https://embed.su/embed/movie/" + id;
  console.log("Fetching", url);
  const res = await fetch(url, {
    headers: { "User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36" },
  });
  if (!res.ok) return console.log("Not ok:", res.status);
  const html = await res.text();

  const match = html.match(/window\.vConfig = JSON\.parse\(atob\(\(.+?)\\)\)/);
  if (!match) return console.log("No config match");

  const decodedData = JSON.parse(Buffer.from(match[1], "base64").toString());
  const firstDecode = atob(decodedData.hash).split(".").map((item) => item.split("").reverse().join(""));
  const servers = JSON.parse(atob(firstDecode.join("").split("").reverse().join("")));
  console.log("Servers:", servers);
  
  if (servers.length > 0) {
      const s_res = await fetch("https://embed.su/api/e/" + servers[0].hash, {
          headers: { "User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36" },
      });
      console.log("Stream status:", s_res.status);
      console.log("Stream:", await s_res.json());
  }
}

embedSuGetVideo(862);
