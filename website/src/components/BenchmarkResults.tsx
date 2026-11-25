import { Fragment } from "react";
import "./benchmark-results.css";

type Measurement = [time: number, bytes: number, allocs: number];
type Result = [scenario: string, intl: Measurement, i18n: Measurement];

const groups: { title: string; decimals: number; cases: Result[] }[] = [
  {
    title: "Interpolation · median of 6 runs",
    decimals: 2,
    cases: [
      ["Bare argument {first}", [27.54, 8, 1], [284.6, 275, 7]],
      ["Hello, {first}!", [52.34, 24, 2], [300.7, 288, 7]],
      ["Item {id}. (42)", [88.86, 24, 3], [307.6, 280, 7]],
      ["{first} {last}", [70.28, 24, 2], [420.15, 320, 9]],
      ["{first}{last}", [68.54, 24, 2], [410.2, 320, 9]],
      ["4 distinct arguments", [97.93, 24, 2], [670.65, 384, 13]],
      ["8 distinct arguments", [156.7, 40, 2], [1152.5, 528, 21]],
      ["16 distinct arguments", [257.1, 136, 3], [2015.5, 816, 37]],
      ["32 distinct arguments", [470.55, 264, 3], [3877.0, 1520, 70]],
      ["One argument repeated 16 times", [265.55, 136, 3], [2004.0, 816, 37]],
    ],
  },
  {
    title: "Static lookup and plural · median of 3 runs",
    decimals: 1,
    cases: [
      ["Repeated static key", [12.9, 0, 0], [116.8, 0, 0]],
      ["Rotating static keys (1,024 keys)", [15.8, 0, 0], [125.8, 0, 0]],
      ["English plural (count 7)", [81.2, 16, 2], [336.5, 344, 8]],
    ],
  },
];

const maxSpeedup = Math.max(
  ...groups.flatMap(({ cases }) =>
    cases.map(([, [intl], [i18n]]) => i18n / intl),
  ),
);

export default function BenchmarkResults() {
  return (
    <div className="benchmark-results">
      {groups.map(({ title, decimals, cases }) => {
        const time = new Intl.NumberFormat("en-US", {
          minimumFractionDigits: decimals,
          maximumFractionDigits: decimals,
        });
        return (
          <div
            className="benchmark-scroll"
            key={title}
            tabIndex={0}
            role="region"
            aria-label={title}
          >
            <table>
              <caption>{title}</caption>
              <colgroup>
                <col className="case-column" />
              </colgroup>
              <colgroup span={3} />
              <colgroup span={3} />
              <colgroup />
              <thead>
                <tr>
                  <th scope="col" rowSpan={2}>
                    Case
                  </th>
                  <th scope="colgroup" colSpan={3}>
                    go-intl
                  </th>
                  <th scope="colgroup" colSpan={3}>
                    go-i18n
                  </th>
                  <th scope="col" rowSpan={2}>
                    Speedup
                  </th>
                </tr>
                <tr>
                  {["go-intl", "go-i18n"].map((library) => (
                    <Fragment key={library}>
                      <th scope="col">ns/op</th>
                      <th scope="col">B/op</th>
                      <th scope="col">allocs/op</th>
                    </Fragment>
                  ))}
                </tr>
              </thead>
              <tbody>
                {cases.map(([scenario, intl, i18n]) => (
                  <tr key={scenario}>
                    <th scope="row">{scenario}</th>
                    {[intl, i18n].map(([ns, bytes, allocs], index) => (
                      <Fragment key={index}>
                        <td>{time.format(ns)}</td>
                        <td>{bytes.toLocaleString("en-US")}</td>
                        <td>{allocs}</td>
                      </Fragment>
                    ))}
                    <td>
                      <strong>{(i18n[0] / intl[0]).toFixed(2)}×</strong>
                      <span aria-hidden="true" className="speedup-track">
                        <span
                          style={{
                            width: `${(i18n[0] / intl[0] / maxSpeedup) * 100}%`,
                          }}
                        />
                      </span>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        );
      })}
    </div>
  );
}
