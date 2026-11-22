import { landBands } from "./land_bands";

const radians = Math.PI / 180;
const radius = 540;
const tilt = -13 * radians;

// Project latitude rings at build time. A repeatable offset varies the spacing
// without changing the coastline or requiring a browser animation loop.
function globePaths() {
  const paths = { sea: "", land: "" };
  let seed = 42;
  function random() {
    seed = (Math.imul(seed, 1664525) + 1013904223) >>> 0;
    return seed / 2 ** 32;
  }

  for (const [latitude, crossings] of landBands.entries()) {
    const lat = latitude * radians;
    const count = Math.round(360 * Math.cos(lat));
    const offset = random();
    let crossing = 0;

    for (let i = 0; i < count; i++) {
      const longitude = ((i + offset) * 360) / count - 180;
      while (crossing < crossings.length && longitude >= crossings[crossing]) {
        crossing++;
      }
      const land = crossing % 2 === 1;
      if (random() > (land ? 0.96 : 0.38)) continue;

      const phi = lat + (random() - 0.5) * 0.35 * radians;
      const lambda = (longitude + 18 + (random() - 0.5) * 0.35) * radians;
      const forward = Math.cos(phi) * Math.cos(lambda);
      const depth = forward * Math.cos(tilt) + Math.sin(phi) * Math.sin(tilt);
      if (depth < 0) continue;

      const x = Math.round(radius * Math.cos(phi) * Math.sin(lambda));
      const y = Math.round(
        radius *
          (1 - Math.sin(phi) * Math.cos(tilt) + forward * Math.sin(tilt)),
      );
      if (y > radius / 2 + 2) continue;
      paths[land ? "land" : "sea"] += `M${x} ${y}h0`;
    }
  }
  return paths;
}

const paths = globePaths();

export default function Globe() {
  return (
    <svg
      aria-hidden="true"
      focusable="false"
      viewBox="-475 -2 950 272"
      className="mt-8 block h-auto w-[clamp(460px,94vw,1120px)] max-w-none shrink-0 md:mt-12"
      fill="none"
      strokeLinecap="round"
    >
      <path d={paths.sea} className="stroke-primary/30" strokeWidth="2.6" />
      <path d={paths.land} className="stroke-primary" strokeWidth="4.4" />
    </svg>
  );
}
