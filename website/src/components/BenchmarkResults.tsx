import "./benchmark-results.css";

type Result = [
  scenario: string,
  intlTime: number,
  intlAllocs: number,
  i18nTime: number,
  i18nAllocs: number,
];

const groups: { title: string; decimals: number; cases: Result[] }[] = [
  {
    title: "Interpolation · median of 6 runs",
    decimals: 2,
    cases: [
      ["Bare argument {first}", 27.54, 1, 284.6, 7],
      ["Hello, {first}!", 52.34, 2, 300.7, 7],
      ["Item {id}. (42)", 88.86, 3, 307.6, 7],
      ["{first} {last}", 70.28, 2, 420.15, 9],
      ["{first}{last}", 68.54, 2, 410.2, 9],
      ["4 distinct arguments", 97.93, 2, 670.65, 13],
      ["8 distinct arguments", 156.7, 2, 1152.5, 21],
      ["16 distinct arguments", 257.1, 3, 2015.5, 37],
      ["32 distinct arguments", 470.55, 3, 3877.0, 70],
      ["One argument repeated 16 times", 265.55, 3, 2004.0, 37],
    ],
  },
  {
    title: "Static lookup and plural · median of 3 runs",
    decimals: 1,
    cases: [
      ["Repeated static key", 12.9, 0, 116.8, 0],
      ["Rotating static keys (1,024 keys)", 15.8, 0, 125.8, 0],
      ["English plural (count 7)", 81.2, 2, 336.5, 8],
    ],
  },
];

const maxSpeedup = Math.max(
  ...groups.flatMap(({ cases }) =>
    cases.map(([, intl, , i18n]) => i18n / intl),
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
          <table key={title}>
            <caption>{title}</caption>
            <thead>
              <tr>
                <th scope="col">Case</th>
                <th scope="col">
                  go-intl<small>ns/op</small>
                </th>
                <th scope="col">
                  go-i18n<small>ns/op</small>
                </th>
                <th scope="col">Speedup</th>
              </tr>
            </thead>
            <tbody>
              {cases.map(
                ([scenario, intlTime, intlAllocs, i18nTime, i18nAllocs]) => (
                  <tr key={scenario}>
                    <th scope="row">{scenario}</th>
                    <td>
                      {time.format(intlTime)}
                      <small>
                        {intlAllocs} {intlAllocs === 1 ? "alloc" : "allocs"}
                      </small>
                    </td>
                    <td>
                      {time.format(i18nTime)}
                      <small>{i18nAllocs} allocs</small>
                    </td>
                    <td>
                      <strong>{(i18nTime / intlTime).toFixed(2)}×</strong>
                      <span aria-hidden="true" className="speedup-track">
                        <span
                          style={{
                            width: `${(i18nTime / intlTime / maxSpeedup) * 100}%`,
                          }}
                        />
                      </span>
                    </td>
                  </tr>
                ),
              )}
            </tbody>
          </table>
        );
      })}
    </div>
  );
}
