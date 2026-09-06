const res = await fetch('https://yts.mx/api/v2/list_movies.json?query_term=tt0111161');
const json = await res.json();
console.log(json.data.movies[0].torrents[0].url);
