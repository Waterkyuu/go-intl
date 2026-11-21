import react from "@astrojs/react";
import starlight from "@astrojs/starlight";
import tailwindcss from "@tailwindcss/vite";
import { defineConfig } from "astro/config";

export default defineConfig({
  integrations: [
    starlight({
      title: "go-intl Docs",
      favicon: "/favicon.png",
      components: { Head: "./src/components/DocsHead.astro" },
      head: [
        { tag: "meta", attrs: { name: "twitter:card", content: "summary" } },
      ],
      social: [
        {
          icon: "github",
          label: "GitHub",
          href: "https://github.com/Waterkyuu/go-intl",
        },
      ],
      sidebar: [
        {
          label: "Documentation",
          items: [{ autogenerate: { directory: "guides" } }],
        },
      ],
      customCss: ["./src/styles/docs.css"],
    }),
    react(),
  ],
  vite: { plugins: [tailwindcss()] },
  devToolbar: { enabled: false },
});
