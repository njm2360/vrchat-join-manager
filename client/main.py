import os
import asyncio
import logging
from pathlib import Path

from dotenv import load_dotenv

from api_client import ApiClient
from log_parser import VRChatLogParser
from log_watcher import LogWatcher

load_dotenv()

logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(message)s")
logger = logging.getLogger(__name__)

LOG_DIR = Path.home() / "AppData" / "LocalLow" / "VRChat" / "VRChat"


async def main() -> None:
    base_url = os.environ["API_BASE"]
    api_client = ApiClient(base_url)
    parser = VRChatLogParser(api_client)
    watcher = LogWatcher(
        log_dir=LOG_DIR,
        on_line=parser.on_line,
    )
    logger.info("Watching: %s", LOG_DIR)
    await watcher.run()


if __name__ == "__main__":
    asyncio.run(main())
