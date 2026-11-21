import { defineConfig } from "eslint/config";
import js from "@eslint/js";
import astro from "eslint-plugin-astro";
import react from "eslint-plugin-react";
import prettier from "eslint-config-prettier";
import tseslint from "typescript-eslint";

export default defineConfig(
  { ignores: ["dist/**", ".astro/**", "node_modules/**"] },
  js.configs.recommended,
  { files: ["**/*.{ts,tsx}"], extends: [tseslint.configs.recommended] },
  ...astro.configs.recommended,
  {
    files: ["**/*.astro"],
    languageOptions: {
      parserOptions: {
        parser: tseslint.parser,
        extraFileExtensions: [".astro"],
      },
    },
  },
  {
    files: ["**/*.tsx"],
    ...react.configs.flat.recommended,
    rules: {
      ...react.configs.flat.recommended.rules,
      ...react.configs.flat["jsx-runtime"].rules,
      "react/prop-types": "off",
    },
    settings: { react: { version: "detect" } },
  },
  prettier,
);
