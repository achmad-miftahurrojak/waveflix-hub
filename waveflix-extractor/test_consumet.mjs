import { MOVIES } from "@consumet/extensions";

async function test() {
    const flixhq = new MOVIES.FlixHQ();
    try {
        const searchRes = await flixhq.search("Toy Story");
        if (searchRes.results.length === 0) {
            console.log("Not found");
            return;
        }
        const id = searchRes.results[0].id;
        console.log("Found ID:", id);
        
        const info = await flixhq.fetchMediaInfo(id);
        const epId = info.episodes[0].id;
        
        const stream = await flixhq.fetchEpisodeSources(epId, id);
        console.log(JSON.stringify(stream, null, 2));
    } catch(e) {
        console.error(e);
    }
}
test();
