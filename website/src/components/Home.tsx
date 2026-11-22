import Globe from "./Globe";
import { repository, sitePath, type GuideLink } from "./navigation";

const benchmarks = [
  {
    label: "Repeated static key",
    results: [
      ["go-intl", 12.9, 0],
      ["go-i18n", 116.8, 0],
    ],
  },
  {
    label: "Rotating static keys (1,024 keys)",
    results: [
      ["go-intl", 15.8, 0],
      ["go-i18n", 125.8, 0],
    ],
  },
  {
    label: "Greeting with a named argument",
    results: [
      ["go-intl", 66.2, 3],
      ["go-i18n", 288.2, 7],
    ],
  },
  {
    label: "English plural (count 7)",
    results: [
      ["go-intl", 81.2, 2],
      ["go-i18n", 336.5, 8],
    ],
  },
] as const;

const maxTime = Math.max(
  ...benchmarks.flatMap(({ results }) => results.map(([, time]) => time)),
);

export default function Home({ guides }: { guides: GuideLink[] }) {
  return (
    <>
      <section
        className="hero flex flex-col items-center overflow-hidden text-center"
        aria-labelledby="hero-title"
      >
        <div className="frame flex flex-col items-center">
          <a
            className="mb-5 inline-flex items-center gap-2 text-sm font-semibold text-primary hover:underline"
            href={`${repository}/releases/tag/v0.1.0`}
          >
            go-intl v0.1.0 <span aria-hidden="true">↗</span>
          </a>
          <h1 id="hero-title" className="hero-title">
            Internationalization for Go.
          </h1>
          <p className="mt-5 max-w-[580px] text-[21px] leading-[1.35] text-balance text-secondary">
            One translation catalog for your terminal, desktop, and server
            applications.
          </p>
          <div className="mt-7 flex flex-wrap items-center justify-center gap-x-6 gap-y-3">
            <a href={sitePath("guides/get-started/")} className="button">
              Get started
            </a>
            <a
              href={sitePath("guides/performance/")}
              className="text-primary hover:underline"
            >
              View benchmarks <span aria-hidden="true">›</span>
            </a>
          </div>
        </div>
        <Globe />
      </section>
      <section className="bg-tile text-white" aria-labelledby="bench-title">
        <div className="frame py-16 md:py-20">
          <h2
            id="bench-title"
            className="text-[40px] leading-[1.1] font-semibold tracking-tight"
          >
            Translation benchmarks
          </h2>
          <p className="mt-4 max-w-2xl text-[17px] leading-relaxed text-muted">
            Time per English translation with a bound translator or warmed
            localizer and reused arguments. Shorter bars mean faster rendering.
          </p>
          <div className="mt-10 grid gap-x-16 gap-y-10 md:grid-cols-2">
            {benchmarks.map(({ label, results }) => (
              <figure key={label}>
                <figcaption className="mb-4 text-xl font-semibold">
                  {label}
                </figcaption>
                {results.map(([library, time, allocs]) => (
                  <div key={library} className="mt-3">
                    <div className="mb-2 flex flex-wrap items-baseline justify-between gap-x-2 text-sm">
                      <span>{library}</span>
                      <span className="font-mono tabular-nums">
                        {time} ns/op{" "}
                        <span className="text-muted">· {allocs} allocs/op</span>
                      </span>
                    </div>
                    <div
                      aria-hidden="true"
                      className={
                        library === "go-intl"
                          ? "h-2 rounded-full bg-action-light"
                          : "h-2 rounded-full bg-muted"
                      }
                      style={{ width: `${(time / maxTime) * 100}%` }}
                    />
                  </div>
                ))}
              </figure>
            ))}
          </div>
          <div className="mt-10 flex flex-wrap items-center justify-between gap-4">
            <p className="text-sm text-muted">
              Apple M3 Pro · Go 1.27.1 · median of three runs
            </p>
            <a
              href={sitePath("guides/performance/")}
              className="text-action-light hover:underline"
            >
              Benchmark methodology <span aria-hidden="true">›</span>
            </a>
          </div>
        </div>
      </section>
      <section aria-labelledby="guide-title" className="frame py-16 md:py-20">
        <h2
          id="guide-title"
          className="mb-8 text-[40px] font-semibold tracking-tight"
        >
          Documentation
        </h2>
        <div className="grid gap-x-14 md:grid-cols-2">
          {guides.map(({ href, title, summary }) => (
            <a
              key={href}
              href={href}
              className="group flex items-center justify-between gap-4 border-t border-hairline py-6"
            >
              <div>
                <h3 className="text-xl font-semibold group-hover:text-primary">
                  {title}
                </h3>
                <p className="mt-1 text-sm text-secondary">{summary}</p>
              </div>
              <span aria-hidden="true" className="text-xl text-primary">
                ›
              </span>
            </a>
          ))}
        </div>
      </section>
    </>
  );
}
