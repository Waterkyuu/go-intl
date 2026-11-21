"""Generate immutable list and relative-time tables from published CLDR 48.2 archives."""

import hashlib
import io
import json
import pathlib
import re
import subprocess
import tempfile
import tarfile
import urllib.request

HERE = pathlib.Path(__file__).resolve().parent
VERSION = "48.2.0"
DIGESTS = {
    "misc": "c6ba8384d7ea8701cf86935db0461379231ddd970cc41c249f4a33b9857ded3c",
    "core": "5310e0c7a06c1feb83dc8e54c8584bbe0b9c2ea8320172a18d541986b124585d",
    "numbers": "2d17a1453c559a62112caeed52e0bcfe3cb8539c99d239ae7b7ed4d0827679d9",
    "dates": "0256f1cefeca14f7d515be4872dda48fcdd7e75381430f25aafd0143eae5b430",
}
UNITS = ("second", "minute", "hour", "day", "week", "month", "year")
FORMS = ("other", "zero", "one", "two", "few", "many")  # x/text/plural.Form order


def archive_for(package):
    """Open a release archive after checking its pinned SHA-256 digest."""
    name = "cldr-core" if package == "core" else f"cldr-{package}-full"
    url = f"https://registry.npmjs.org/{name}/-/{name}-{VERSION}.tgz"
    with urllib.request.urlopen(url, timeout=60) as response:
        data = response.read()
    if hashlib.sha256(data).hexdigest() != DIGESTS[package]:
        raise ValueError(f"unexpected archive digest for {name}")
    return tarfile.open(fileobj=io.BytesIO(data))


def documents(package, filename):
    """Read only the requested locale documents from a checksum-verified tarball."""
    with archive_for(package) as archive:
        license_text = archive.extractfile("package/LICENSE").read()
        if (HERE / "LICENSE").read_bytes() != license_text:
            raise ValueError("review the updated CLDR data license before regenerating")
        for member in archive:
            if member.name.endswith("/" + filename):
                yield next(iter(json.load(archive.extractfile(member))["main"].items()))


def list_fields(doc):
    """Split the conjunction patterns into text between and after the two items."""
    pattern = doc["listPatterns"]["listPattern-type-standard"]
    for part in ("2", "start", "middle", "end"):
        text = pattern[part]
        if not text.startswith("{0}") or text.count("{1}") != 1:
            raise ValueError(f"unsupported CLDR list pattern {text!r}")
        yield from text[3:].split("{1}")


def relative_fields(doc):
    """Resolve plural fallback while generating the locale's compact indexed row."""
    fields = doc["dates"]["fields"]
    yield fields["second"].get("relative-type-0", "")
    for unit in UNITS:
        for direction in ("future", "past"):
            forms = fields[unit]["relativeTime-type-" + direction]
            for category in FORMS:
                yield forms.get("relativeTimePattern-count-" + category, forms["relativeTimePattern-count-other"])


def currency_tables():
    """Compile standard currency affixes, grouping widths and localized punctuation."""
    with archive_for("core") as archive:
        systems = json.load(archive.extractfile("package/supplemental/numberingSystems.json"))
    systems = systems["supplemental"]["numberingSystems"]
    locales, rows = {}, {}
    for locale, doc in sorted(documents("numbers", "numbers.json")):
        numbers = doc["numbers"]
        for key, formats in numbers.items():
            if not key.startswith("currencyFormats-numberSystem-"):
                continue
            system = key.rsplit("-", 1)[1]
            symbols = numbers[f"symbols-numberSystem-{system}"]
            standard = formats["standard"]
            alpha = formats.get("standard-alphaNextToNumber", standard)
            patterns = []
            for fmt in (standard, alpha):
                positive, sep, negative = fmt.partition(";")
                if not sep:
                    negative = "-" + positive
                for pat in (positive, negative):
                    normalized, count = re.subn(r"[#,0]+\.00", "#", pat)
                    if count != 1 or normalized.count("¤") != 1:
                        raise ValueError(f"unsupported currency pattern {pat!r}")
                    patterns.append(normalized.replace("-", symbols["minusSign"]))
            grouping = re.search(r"([#,0]+)\.00", standard)[1].split(",")
            primary = len(grouping[-1])
            secondary = primary if len(grouping) < 3 else len(grouping[-2])
            digits = systems[system]["_digits"]
            strings = [symbols.get("currencyGroup", symbols["group"]),
                       symbols.get("currencyDecimal", symbols["decimal"]), symbols["infinity"], symbols["nan"]]
            record = ("{[%d]rune{%s},[4]string{%s},%s,%d,%d,%d}" %
                      (len(digits), ",".join(str(ord(d)) for d in digits),
                       ",".join(json.dumps(x, ensure_ascii=False) for x in patterns),
                       ",".join(json.dumps(x, ensure_ascii=False) for x in strings),
                       primary, secondary, int(numbers["minimumGroupingDigits"])))
            idx = rows.setdefault(record, len(rows))
            locales[locale + "-u-nu-" + system] = idx
            if system == numbers["defaultNumberingSystem"]:
                locales[locale] = idx
    keys = sorted(locales)
    return [
        ("currencyLocales", "string", [json.dumps(k) for k in keys]),
        ("currencyRows", "uint16", [str(locales[k]) for k in keys]),
        ("currencies", "CurrencyFormat", list(rows)),
    ]


def main():
    """Intern shared words and rows, then write standard Go arrays in lexical locale order."""
    lists = dict(documents("misc", "listPatterns.json"))
    dates = dict(documents("dates", "dateFields.json"))
    if lists.keys() != dates.keys():
        raise ValueError("CLDR packages have different locale coverage")
    words, rows, indexes = {}, {}, []
    for locale in sorted(lists):
        slots = (*list_fields(lists[locale]), *relative_fields(dates[locale]))
        row = tuple(words.setdefault(word, len(words)) for word in slots)
        indexes.append(rows.setdefault(row, len(rows)))
    if max(len(words), len(rows)) >= 65536:
        raise ValueError("CLDR exceeds the uint16 index width")
    with tempfile.TemporaryDirectory(prefix=".cldr-", dir=HERE) as tmp:
        output = pathlib.Path(tmp) / "data.go"
        with output.open("w", encoding="utf-8") as out:
            out.write(f"// Code generated from Unicode CLDR {VERSION} by generate.py. DO NOT EDIT.\n")
            out.write("// Licensed under the Unicode License v3. See LICENSE.\npackage cldrtext\n")
            for name, typ, values in [
                ("locales", "string", [json.dumps(tag) for tag in sorted(lists)]),
                ("localeRows", "uint16", map(str, indexes)),
                ("words", "string", [json.dumps(word, ensure_ascii=False) for word in words]),
                ("rows", f"[{len(row)}]uint16", ["{" + ",".join(map(str, row)) + "}" for row in rows]),
            ] + currency_tables():
                out.write(f"\nvar {name} = [...]%s{{\n" % typ)
                for value in values:
                    out.write(value + ",\n")
                out.write("}\n")
        subprocess.run(["gofmt", "-w", output], check=True)
        output.replace(HERE / "data.go")


if __name__ == "__main__":
    main()
