# Graph Report - waveflix-hub  (2026-08-20)

## Corpus Check
- cluster-only mode — file stats not available

## Summary
- 492 nodes · 1087 edges · 29 communities (21 shown, 8 thin omitted)
- Extraction: 98% EXTRACTED · 2% INFERRED · 0% AMBIGUOUS · INFERRED: 24 edges (avg confidence: 0.78)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `6faebc6b`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- net/http.ResponseWriter
- helpers.ts
- types.ts
- compress.py
- tmdb.ts
- devDependencies
- validate.py
- Icons.tsx
- compilerOptions
- browse/page.tsx
- caveman-explore/package.json
- profileFonts.ts
- DetailActions.tsx
- useAuth
- AuthForm.tsx
- account/page.tsx
- AuthProvider.tsx
- caveman-explore/tests/skill-file.test.mjs
- route.ts
- caveman-learn/tests/skill-file.test.mjs
- __init__.py
- idea-refine.sh
- next.config.mjs
- postcss.config.mjs
- tailwind.config.ts
- summertide-backend

## God Nodes (most connected - your core abstractions)
1. `TmdbItem` - 35 edges
2. `itemTitle()` - 21 edges
3. `useAuth()` - 17 edges
4. `httpError()` - 17 edges
5. `isTv()` - 16 edges
6. `compress_file()` - 16 edges
7. `getHeroSlides()` - 16 edges
8. `compilerOptions` - 16 edges
9. `MediaType` - 15 edges
10. `mediaTypeOf()` - 14 edges

## Surprising Connections (you probably didn't know these)
- `ClientHeroSlide` --references--> `TmdbItem`  [EXTRACTED]
  website/frontend/components/UserLibrary.tsx → website/frontend/lib/types.ts
- `AuthContextValue` --references--> `TmdbItem`  [EXTRACTED]
  website/frontend/components/AuthProvider.tsx → website/frontend/lib/types.ts
- `Source` --references--> `MediaType`  [EXTRACTED]
  website/frontend/components/SwitchableCarousel.tsx → website/frontend/lib/types.ts
- `LatestEpisode` --references--> `TmdbItem`  [EXTRACTED]
  website/frontend/lib/tmdb.ts → website/frontend/lib/types.ts
- `Props` --references--> `TmdbItem`  [EXTRACTED]
  website/frontend/components/Carousel.tsx → website/frontend/lib/types.ts

## Import Cycles
- None detected.

## Communities (29 total, 8 thin omitted)

### Community 0 - "net/http.ResponseWriter"
Cohesion: 0.10
Nodes (49): authResponse, ctxKey, mediaItem, visitor, net/http.Handler, net/http.HandlerFunc, net/http.Request, net/http.ResponseWriter (+41 more)

### Community 1 - "helpers.ts"
Cohesion: 0.10
Nodes (37): DetailPage(), dynamicParams, dynamicParams, EpisodePage(), CastRow(), DetailActions(), EpisodesSection(), formatAirDate() (+29 more)

### Community 2 - "types.ts"
Cohesion: 0.09
Nodes (32): Carousel(), Props, EpisodePlayButton(), Props, Props, MovieCard(), Props, Props (+24 more)

### Community 3 - "compress.py"
Cohesion: 0.10
Nodes (37): main(), print_usage(), backup_dir_for(), build_compress_prompt(), build_fix_prompt(), call_claude(), compress_file(), first_nonblank_line() (+29 more)

### Community 4 - "tmdb.ts"
Cohesion: 0.12
Nodes (31): BrowsePage(), Home(), ORIGINALS, REGIONS, LandingPage(), dynamic, RealityPage(), dynamic (+23 more)

### Community 5 - "devDependencies"
Cohesion: 0.06
Nodes (34): autoprefixer, cheerio, framer-motion, postcss, react, react-dom, tailwindcss, @types/node (+26 more)

### Community 6 - "validate.py"
Cohesion: 0.11
Nodes (28): benchmark_pair(), count_tokens(), main(), print_table(), Path, count_bullets(), extract_code_blocks(), extract_fenced_spans() (+20 more)

### Community 7 - "Icons.tsx"
Cohesion: 0.12
Nodes (24): BookmarkIcon(), CalendarIcon(), ChevronRight(), CloseIcon(), FilmIcon(), GlobeIcon(), HomeIcon(), MaximizeIcon() (+16 more)

### Community 8 - "compilerOptions"
Cohesion: 0.07
Nodes (26): dom, dom.iterable, esnext, next-env.d.ts, .next/types/**/*.ts, node_modules, **/*.ts, **/*.tsx (+18 more)

### Community 9 - "browse/page.tsx"
Cohesion: 0.18
Nodes (13): dynamic, GenresPage(), CategoryTiles(), Tile, FilterBar(), COUNTRIES, genresFor(), MOVIE_GENRES (+5 more)

### Community 10 - "caveman-explore/package.json"
Cohesion: 0.10
Nodes (19): description, files, license, name, private, scripts, test, type (+11 more)

### Community 11 - "profileFonts.ts"
Cohesion: 0.12
Nodes (15): Msg, pillSpring, Tab, bebas, birds, caveat, floozy, lobster (+7 more)

### Community 12 - "DetailActions.tsx"
Cohesion: 0.22
Nodes (11): readList(), toggleLocal(), HeartIcon(), HeartSolidIcon(), PlayIcon(), PlusIcon(), FAV_KEY, LEGACY (+3 more)

### Community 13 - "useAuth"
Cohesion: 0.21
Nodes (9): SettingsPage(), logoFont, metadata, poppins, AuthProvider(), useAuth(), PUBLIC_ROUTES, RouteGuard() (+1 more)

### Community 14 - "AuthForm.tsx"
Cohesion: 0.32
Nodes (3): AuthForm(), PERKS, CheckIcon()

### Community 15 - "account/page.tsx"
Cohesion: 0.47
Nodes (5): AccountPage(), pillSpring, Tab, toItem(), nameFontCss()

### Community 16 - "AuthProvider.tsx"
Cohesion: 0.40
Nodes (4): AuthContext, AuthContextValue, AuthUser, ProfileFields

### Community 18 - "route.ts"
Cohesion: 0.67
Nodes (3): DRAMACOOL_MIRRORS, extractEmbedUrl(), GET()

## Knowledge Gaps
- **114 isolated node(s):** `Props`, `Msg`, `Tab`, `Tab`, `AuthUser` (+109 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **8 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `TmdbItem` connect `types.ts` to `helpers.ts`, `tmdb.ts`, `DetailActions.tsx`, `account/page.tsx`, `AuthProvider.tsx`?**
  _High betweenness centrality (0.032) - this node is a cross-community bridge._
- **Why does `useAuth()` connect `useAuth` to `helpers.ts`, `types.ts`, `Icons.tsx`, `profileFonts.ts`, `DetailActions.tsx`, `AuthForm.tsx`, `account/page.tsx`, `AuthProvider.tsx`?**
  _High betweenness centrality (0.016) - this node is a cross-community bridge._
- **Why does `itemTitle()` connect `helpers.ts` to `AuthProvider.tsx`, `types.ts`, `DetailActions.tsx`, `useAuth`?**
  _High betweenness centrality (0.008) - this node is a cross-community bridge._
- **What connects `Props`, `Msg`, `Tab` to the rest of the system?**
  _114 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `net/http.ResponseWriter` be split into smaller, more focused modules?**
  _Cohesion score 0.09935064935064936 - nodes in this community are weakly interconnected._
- **Should `helpers.ts` be split into smaller, more focused modules?**
  _Cohesion score 0.09643605870020965 - nodes in this community are weakly interconnected._
- **Should `types.ts` be split into smaller, more focused modules?**
  _Cohesion score 0.0898989898989899 - nodes in this community are weakly interconnected._