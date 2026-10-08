"""Deterministic Windows client ZIP (build tooling, not a runtime dependency)."""
import datetime,os,sys,zipfile
from pathlib import Path
if len(sys.argv)!=4:raise SystemExit('usage: zip.py SOURCE OUTPUT EPOCH')
source=Path(sys.argv[1]);epoch=int(sys.argv[3]);stamp=datetime.datetime.fromtimestamp(max(epoch,315532800),datetime.timezone.utc)
with zipfile.ZipFile(sys.argv[2],'w',compression=zipfile.ZIP_DEFLATED,compresslevel=9) as archive:
 for file in sorted(source.rglob('*')):
  if file.is_symlink():raise SystemExit('symlink in client ZIP')
  if not file.is_file():continue
  info=zipfile.ZipInfo(file.relative_to(source).as_posix(),stamp.timetuple()[:6]);info.create_system=3;info.external_attr=(0o100755 if file.name.endswith('.exe') else 0o100644)<<16;info.compress_type=zipfile.ZIP_DEFLATED
  archive.writestr(info,file.read_bytes(),compress_type=zipfile.ZIP_DEFLATED,compresslevel=9)
