#!/usr/bin/env python3
"""Build/check the reviewable central-workflow plan without local runtime data."""
import argparse
import hashlib
import io
from pathlib import Path
import zipfile

ROOT = Path(__file__).resolve().parents[2]
SOURCE = ROOT / 'docs/central-agent-workflow'
OUTPUT = ROOT / 'docs/central-agent-workflow-plan.zip'


def package():
    files = sorted(path for path in SOURCE.rglob('*') if path.is_file())
    entries = []
    for path in files:
        if path.is_symlink() or path.suffix not in {'.md', '.txt', '.json'}:
            raise ValueError(f'Unexpected plan entry: {path.relative_to(ROOT)}')
        entries.append((path.relative_to(SOURCE).as_posix(), path.read_bytes()))
    manifest = ''.join(f'{hashlib.sha256(data).hexdigest()}  {name}\n' for name, data in entries)
    entries.append(('SHA256SUMS', manifest.encode()))
    buffer = io.BytesIO()
    with zipfile.ZipFile(buffer, 'w', compression=zipfile.ZIP_DEFLATED, compresslevel=9) as archive:
        for name, data in entries:
            info = zipfile.ZipInfo('central-agent-workflow/' + name, (2026, 10, 16, 0, 0, 0))
            info.create_system = 3
            info.external_attr = 0o100644 << 16
            info.compress_type = zipfile.ZIP_DEFLATED
            archive.writestr(info, data, compresslevel=9)
    return buffer.getvalue(), len(entries)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--check', action='store_true')
    args = parser.parse_args()
    data, count = package()
    if args.check:
        if not OUTPUT.exists() or OUTPUT.read_bytes() != data:
            parser.exit(1, 'Plan ZIP is stale; run python3 tools/documentation/package-central-plan.py\n')
    else:
        OUTPUT.write_bytes(data)
    print(f'{OUTPUT.relative_to(ROOT)}: {count} entries, SHA-256 {hashlib.sha256(data).hexdigest()}')


if __name__ == '__main__':
    main()
