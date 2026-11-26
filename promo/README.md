# hark promo

Promo videos for hark, made with [Remotion](https://www.remotion.dev): `Promo` runs 60 seconds at 1920×1080, and `PromoLinkedIn` runs 30 seconds at 1080×1350 for the LinkedIn feed.

```console
npm i
npm run dev
npx remotion render Promo out/hark-promo.mp4
npx remotion render PromoLinkedIn out/hark-promo-linkedin.mp4
```

The render lands in `out/`, which Git ignores.
