export const repository = "https://github.com/Waterkyuu/go-intl";

export function sitePath(path = "") {
  return `${import.meta.env.BASE_URL.replace(/\/$/, "")}/${path}`;
}

export interface GuideLink {
  href: string;
  title: string;
  summary: string;
}
