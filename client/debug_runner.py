import os
import sys
import asyncio
import logging
from pathlib import Path

from dotenv import load_dotenv

from api_client import ApiClient
from log_parser import VRChatLogParser

load_dotenv()

logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(message)s")


class DebugFileRunner:
    """指定したログファイルを先頭から末尾まで一括処理する"""

    def __init__(self, path: str | Path) -> None:
        self._path = Path(path)
        base_url = os.environ["API_BASE"]
        self._parser = VRChatLogParser(ApiClient(base_url))

    async def run(self) -> None:
        with self._path.open("r", encoding="utf-8", errors="replace") as f:
            for raw_line in f:
                await self._parser.on_line(self._path, raw_line.rstrip("\n"))


if __name__ == "__main__":
    if len(sys.argv) < 2:
        print(f"Usage: python {sys.argv[0]} <log_file>")
        sys.exit(1)

    asyncio.run(DebugFileRunner(sys.argv[1]).run())
