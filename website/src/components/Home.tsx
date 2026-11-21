import { repository, sitePath, type GuideLink } from "./navigation";

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
        <img
          src={sitePath("hemisphere.webp")}
          srcSet={`${sitePath("hemisphere-864.webp")} 864w, ${sitePath("hemisphere.webp")} 2172w`}
          sizes="min(104vw, 1360px)"
          width="2172"
          height="724"
          fetchPriority="high"
          alt=""
          className="mt-8 block aspect-[6/1] h-auto w-[min(104vw,1360px)] max-w-none object-cover object-top md:mt-12"
        />
      </section>
      <section className="bg-tile text-white" aria-labelledby="bench-title">
        <div className="frame grid items-center gap-10 py-16 md:grid-cols-2 md:py-20">
          <div>
            <h2
              id="bench-title"
              className="text-[40px] leading-[1.1] font-semibold tracking-tight"
            >
              12.9 ns per lookup.
            </h2>
            <p className="mt-4 max-w-md text-[17px] leading-relaxed text-muted">
              Rendering one repeated static key with a bound translator takes
              zero heap allocations. Both libraries rendered the same English
              message.
            </p>
            <a
              href={sitePath("guides/performance/")}
              className="mt-5 inline-block text-action-light hover:underline"
            >
              Explore the results <span aria-hidden="true">›</span>
            </a>
          </div>
          <figure className="md:pl-8">
            <figcaption className="mb-5 text-sm text-muted">
              Repeated static key · lower is faster
            </figcaption>
            {(
              [
                ["go-intl", 12.9],
                ["go-i18n", 116.8],
              ] as const
            ).map(([label, ns]) => (
              <div key={label} className="mb-4">
                <div className="mb-2 flex justify-between gap-4 text-sm">
                  <span>{label}</span>
                  <span className="font-mono">{ns} ns/op</span>
                </div>
                <div
                  className={
                    label === "go-intl"
                      ? "h-2 rounded-full bg-action-light"
                      : "h-2 rounded-full bg-muted"
                  }
                  style={{ width: `${(ns / 116.8) * 100}%` }}
                />
              </div>
            ))}
            <p className="text-xs text-muted">
              Apple M3 Pro · Go 1.27.1 · median of three runs
            </p>
          </figure>
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
