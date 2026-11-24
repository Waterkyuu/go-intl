import { repository, sitePath } from "./navigation";

export function GlobeMark() {
  return (
    <svg
      aria-hidden="true"
      width="23"
      height="23"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.5"
    >
      <circle cx="12" cy="12" r="9.5" />
      <ellipse cx="12" cy="12" rx="4.4" ry="9.5" />
      <path d="M2.5 12h19M4.5 6.6q7.5 2.7 15 0M4.5 17.4q7.5-2.7 15 0" />
    </svg>
  );
}

export function Header() {
  const tabs = [
    ["Overview", "", "overview"],
    ["Documentation", "guides/get-started/", "guides"],
    ["Benchmarks", "guides/performance/", "performance"],
  ];
  return (
    <header>
      <div className="bg-black text-white">
        <div className="frame flex h-11 items-center justify-between text-xs">
          <a
            className="flex items-center gap-2 font-semibold"
            href={sitePath()}
          >
            <GlobeMark />
            <span className="text-base">go-intl</span>
          </a>
          <a href={repository} className="hover:underline">
            GitHub <span aria-hidden="true">↗</span>
          </a>
        </div>
      </div>
      <nav aria-label="Main" className="border-b border-hairline bg-pearl">
        <div className="frame flex min-h-[52px] flex-wrap items-center justify-between gap-x-5 gap-y-2 py-2">
          <a
            href={sitePath()}
            className="text-[21px] font-semibold tracking-tight"
          >
            go-intl Docs
          </a>
          <div className="flex flex-wrap gap-x-5 gap-y-1 text-sm">
            {tabs.map(([title, href, group]) => (
              <a
                key={href}
                href={sitePath(href)}
                aria-current={group === "overview" ? "page" : undefined}
                className={
                  group === "overview" ? "text-primary" : "hover:text-primary"
                }
              >
                {title}
              </a>
            ))}
          </div>
        </div>
      </nav>
    </header>
  );
}

export function Footer() {
  return (
    <footer className="border-t border-hairline bg-parchment py-8 text-sm">
      <div className="frame flex flex-wrap items-center justify-between gap-4">
        <a href={sitePath()} className="font-semibold">
          go-intl
        </a>
        <div className="flex flex-wrap gap-5">
          <a href={sitePath("guides/get-started/")}>Documentation</a>
          <a href={`${repository}/releases/latest`}>Release notes</a>
          <a href={repository}>GitHub ↗</a>
        </div>
      </div>
    </footer>
  );
}
