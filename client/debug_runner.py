import asyncio
import logging
import sys
from pathlib import Path

from log_parser import VRChatLogParser

logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(message)s")


class DebugFileRunner:
    """指定したログファイルを先頭から末尾まで一括処理する"""

    def __init__(self, path: str | Path) -> None:
        self._path = Path(path)
        self._parser = VRChatLogParser()

    async def run(self) -> None:
        with self._path.open("r", encoding="utf-8", errors="replace") as f:
            for raw_line in f:
                await self._parser.on_line(self._path, raw_line.rstrip("\n"))


if __name__ == "__main__":
    if len(sys.argv) < 2:
        print(f"Usage: python {sys.argv[0]} <log_file>")
        sys.exit(1)

    asyncio.run(DebugFileRunner(sys.argv[1]).run())
