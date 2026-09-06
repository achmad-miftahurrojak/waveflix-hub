import { MOVIES } from '@consumet/extensions';
process.env.NODE_TLS_REJECT_UNAUTHORIZED = "0";
(async () => {
  try {
    const flixhq = new MOVIES.FlixHQ();
    const searchRes = await flixhq.search('Deadpool');
    console.log('Search Results:', searchRes.results.map(r => r.id));
    if (searchRes.results.length > 0) {
      const info = await flixhq.fetchMediaInfo(searchRes.results[0].id);
      console.log('Episodes:', info.episodes.length);
      const stream = await flixhq.fetchEpisodeSources(info.episodes[0].id, info.id);
      console.log('Stream Sources:', stream.sources);
    }
  } catch(e) {
    console.error(e);
  }
})();
