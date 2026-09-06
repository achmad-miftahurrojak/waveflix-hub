const https = require('https');

https.get('https://yts.mx/api/v2/list_movies.json?query_term=tt0111161', (res) => {
  let data = '';
  res.on('data', chunk => data += chunk);
  res.on('end', () => {
    const json = JSON.parse(data);
    if(json.data && json.data.movies && json.data.movies.length > 0) {
      console.log(json.data.movies[0].torrents[0].url);
    } else {
      console.log('No movies found');
    }
  });
});
