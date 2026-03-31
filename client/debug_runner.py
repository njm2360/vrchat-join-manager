import os
import sys
import csv
import asyncio
import logging
from pathlib import Path
from datetime import datetime, timezone, timedelta

from dotenv import load_dotenv

from api_client import ApiClient
from log_parser import VRChatLogParser

load_dotenv(override=True)

logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(message)s")

_JST = timezone(timedelta(hours=9))


def _csv_to_vrc_line(timestamp_utc: str, message: str) -> str:
    """GrayLog CSVのタイムスタンプ(UTC ISO 8601)をVRCログ形式(JST)に変換して結合する"""
    dt = datetime.fromisoformat(timestamp_utc.replace("Z", "+00:00"))
    dt_jst = dt.astimezone(_JST)
    return f"{dt_jst.strftime('%Y.%m.%d %H:%M:%S')} {message}"


class DebugFileRunner:
    """指定したログファイルを先頭から末尾まで一括処理する"""

    def __init__(self, path: str | Path) -> None:
        self._path = Path(path)
        base_url = os.environ["API_BASE"]
        self._parser = VRChatLogParser(ApiClient(base_url))

    async def run(self) -> None:
        try:
            if self._path.suffix.lower() == ".csv":
                await self._proc_csv()
            else:
                await self._proc_raw_log()
        finally:
            await self._parser._api.aclose()

    async def _proc_raw_log(self) -> None:
        with self._path.open("r", encoding="utf-8", errors="replace") as f:
            for raw_line in f:
                await self._parser.on_line(self._path, raw_line.rstrip("\n"))

    async def _proc_csv(self) -> None:
        with self._path.open("r", encoding="utf-8", newline="") as f:
            for row in csv.DictReader(f):
                line = _csv_to_vrc_line(row["timestamp"], row["message"])
                await self._parser.on_line(self._path, line)


if __name__ == "__main__":
    if len(sys.argv) < 2:
        print(f"Usage: python {sys.argv[0]} <log_file_or_csv>")
        sys.exit(1)

    asyncio.run(DebugFileRunner(sys.argv[1]).run())
