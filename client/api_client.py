import httpx
import logging
from typing import Optional
from datetime import datetime


logger = logging.getLogger(__name__)
logging.getLogger("httpx").setLevel(logging.WARNING)


class ApiClient:
    def __init__(self, base_url: str) -> None:
        self._events_url = f"{base_url}/api/events"
        self._client = httpx.AsyncClient(timeout=10.0)

    async def aclose(self) -> None:
        await self._client.aclose()

    async def send_event(
        self,
        event: str,
        location_id: str,
        name: str,
        user_id: str,
        internal_id: Optional[int],
        timestamp: datetime,
    ) -> None:
        payload = {
            "event": event,
            "location_id": location_id,
            "name": name,
            "user_id": user_id,
            "internal_id": internal_id,
            "timestamp": timestamp.strftime("%Y-%m-%dT%H:%M:%SZ"),
        }
        try:
            resp = await self._client.post(self._events_url, json=payload)
            logger.debug("POST -> %d", resp.status_code)
        except Exception as exc:
            logger.warning("Failed to send: %s", exc)
