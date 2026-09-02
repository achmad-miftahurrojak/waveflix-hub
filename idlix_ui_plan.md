# Goal: Replicate z2.idlixku.com UI/UX Exact Layout

This plan outlines the changes necessary to exactly replicate the homepage structure, `MovieCard` layout, and `HeroCarousel` design of `z2.idlixku.com` based on the visual audit and your detailed instructions.

## User Review Required

> [!IMPORTANT]
> **Dropdown Rows Strategy**: You requested filters within sections for Platforms (Netflix, Apple TV, etc.) and Asian Dramas (Korean, Chinese, etc.). The reference site uses a title that acts as a dropdown (e.g., clicking "Disney+ Originals" opens a menu to select "Netflix"). I will create two new components: `PlatformRow` and `AsianDramaRow` to handle this interactivity. Please confirm this matches your expectation!

## Open Questions

> [!WARNING]
> For the "Trending in Indonesia" row that includes KDramas, TMDB does not have a single API call to mix global trending with specific KDramas. I will fetch popular KDramas (`with_original_language=ko`) and use those for this row to satisfy the Indonesian market's preference for KDrama. Is this acceptable, or do you want a 50/50 mix of Indonesian content and KDrama?

## Proposed Changes

---

### Frontend Components (`website/frontend/components/`)

#### [MODIFY] `MovieCard.tsx`
- Remove the full hover overlay that contains the title.
- Move the title and year to be **underneath** the poster image.
- Add badges inside the poster at the top left (`TV` / `MOVIE`) and top right (`S1` / `WEB-DL`).
- Add the star rating (⭐) at the bottom left inside the poster, and the comment count (💬 using `vote_count` or mock) at the bottom right.
- Add support for a `rank` prop (1, 2, 3...) to render large white numbers on the top left of the card for the "Trending Now" row.

#### [MODIFY] `MovieRow.tsx`
- Add a `showRank` prop. When true, pass the index (1 to 20) to the `MovieCard`.
- Ensure the "View All >" link is always displayed next to the row title.
- Make the left/right scroll arrows more visible (similar to the reference site).

#### [NEW] `PlatformRow.tsx`
- Create a stateful wrapper around `MovieRow` that contains a dropdown menu in its header.
- The dropdown will allow users to switch between Netflix, Disney+, Apple TV+, Viu, etc.
- When a new platform is selected, the component will fetch the corresponding `/api/discover?provider=...` and update the row content.

#### [NEW] `AsianDramaRow.tsx`
- Create a stateful wrapper around `MovieRow` similar to `PlatformRow`.
- The dropdown will allow users to switch between Korean Drama, Chinese Drama, Japanese Drama, and Thai Drama.
- Will fetch `/api/discover?media=tv&with_original_language=ko` (or `zh`, `ja`, `th`).

#### [MODIFY] `HeroCarousel.tsx`
- Increase the height of the hero section to match the immersive look (~95vh).
- Fetch detailed TMDB data for the top 10 items to get their logos (`append_to_response=images`).
- Display the movie/show logo instead of plain text if a logo is available.
- Add a secondary "Trailer" button next to the "Watch Now" button.
- Ensure the pagination dots are visible at the bottom left.

---

### Frontend Pages (`website/frontend/app/`)

#### [MODIFY] `home/page.tsx`
- Reorganize the homepage to precisely match your requested order:
  1. `HeroCarousel` (Top 10 mixed ID/KDrama)
  2. `MovieRow` for **Trending Now** (Global Top 20) with `showRank={true}`
  3. `MovieRow` for **Trending in Indonesia** (KDrama focus)
  4. `PlatformRow` (Interactive dropdown for Netflix, Disney, etc.)
  5. `AsianDramaRow` (Interactive dropdown for Korean, Chinese, etc.)
  6. `MovieRow` for **Latest Movies** (`sort_by=primary_release_date.desc`)
  7. `MovieRow` for **Latest Series** (`sort_by=first_air_date.desc`)

## Verification Plan

### Automated Tests
- N/A for these visual UI changes.

### Manual Verification
- Review the application in the browser (`http://localhost:3001`).
- Verify that `MovieCard` text is strictly below the image and badges are placed correctly.
- Verify the "Trending Now" row has large rank numbers (1-20).
- Test the interactive dropdowns in the `PlatformRow` and `AsianDramaRow` to ensure they fetch new data.
- Verify the `HeroCarousel` displays logos and has the new "Trailer" button.
