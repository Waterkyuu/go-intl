import react from "@astrojs/react";
import starlight from "@astrojs/starlight";
import tailwindcss from "@tailwindcss/vite";
import { defineConfig } from "astro/config";

export default defineConfig({
  integrations: [
    starlight({
      title: "go-intl Docs",
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
