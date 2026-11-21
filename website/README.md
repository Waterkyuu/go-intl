# go-intl documentation site

The site uses Astro and Starlight for static documentation, React with TypeScript for the homepage, and Tailwind CSS for styling.

## Local development

Run these commands in `website/`.

```sh
npm ci
npm run dev
```

Edit the guides in `src/content/docs/guides/`. Each guide's `sidebar.order` controls its position in the documentation sidebar and on the homepage.

## Checks and production preview

```sh
npm run lint
npm run check
npm run format:check
npm run build
npm run preview
```

The production build writes the static site and Pagefind search index to `dist/`. Use the preview server to test full-text search. Run `npm run format` to apply Prettier or `npm run lint:fix` to apply ESLint fixes.
