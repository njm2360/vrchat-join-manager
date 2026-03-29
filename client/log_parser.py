import re
import logging
from collections import deque
from pathlib import Path
from datetime import datetime, timezone, timedelta

_JST = timezone(timedelta(hours=9))
from api_client import ApiClient

logger = logging.getLogger(__name__)

_LOG_TIME = re.compile(r"^(\d{4}\.\d{2}\.\d{2} \d{2}:\d{2}:\d{2})")
_JOINING = re.compile(r"\[Behaviour\] Joining (wrld_\S+)")
_PLAYER_JOIN = re.compile(r"\[Behaviour\] OnPlayerJoined (.+) \((usr_[0-9a-f\-]+)\)")
_PLAYER_LEFT = re.compile(r"\[Behaviour\] OnPlayerLeft (.+) \((usr_[0-9a-f\-]+)\)")
_PLAYER_API = re.compile(
    r"\[Behaviour\] Initialized PlayerAPI \"(.+)\" is (remote|local)"
)
_RESTORED = re.compile(r"\[Behaviour\] Restored player (\d+)")
_DESTROYING = re.compile(r"\[Behaviour\] Destroying (.+)")


class VRChatLogParser:
    def __init__(self, api_client: ApiClient) -> None:
        self._api = api_client
        self._location: str | None = None
        self._local_name: str | None = None
        self._pending_names: deque[str] = deque()
        self._internal_ids: dict[str, int] = {}
        self._pending_joins: dict[str, str] = {}

    def _timestamp(self, line: str) -> datetime | None:
        m = _LOG_TIME.match(line)
        if m:
            naive = datetime.strptime(m.group(1), "%Y.%m.%d %H:%M:%S")
            return naive.replace(tzinfo=_JST).astimezone(timezone.utc)
        return None

    async def _close_current_location(self, ts: datetime) -> None:
        if self._location:
            logger.info("Closing location %s at %s", self._location, ts)
            await self._api.close_location(self._location, ts)

    async def on_line(self, _: Path, line: str) -> None:
        ts = self._timestamp(line)
        if ts is None:
            return

        # ロケーション移動検知
        m = _JOINING.search(line)
        if m:
            # Destroying が発火しないことがあるためここでもcloseを呼ぶ
            # （既にclose済みなら leave_ts IS NULL 条件で空振り）
            await self._close_current_location(ts)
            self._pending_joins.clear()
            self._pending_names.clear()
            self._internal_ids.clear()
            self._local_name = None
            self._location = m.group(1)
            logger.info("Location: %s", self._location)
            return

        # ロケーションが取得できていない場合は処理しない
        if self._location is None:
            return

        # OnPlayerJoined
        m = _PLAYER_JOIN.search(line)
        if m:
            name, user_id = m.group(1), m.group(2)
            self._pending_joins[name] = user_id
            return

        # Initialized PlayerAPI "XXXX" is (remote | local)
        if "Initialized PlayerAPI" in line:
            m = _PLAYER_API.search(line)
            if m:
                name, kind = m.group(1), m.group(2)
                self._pending_names.append(name)
                # 自分自身のユーザー名を記録
                if kind == "local":
                    self._local_name = name
            return

        # Restored player N
        m = _RESTORED.search(line)
        if m:
            if self._pending_names:
                name = self._pending_names.popleft()
                self._internal_ids[name] = int(m.group(1))
                if name in self._pending_joins:
                    user_id = self._pending_joins.pop(name)
                    logger.info(
                        "JOIN  [%s] %s (%s) internal_id=%s",
                        ts,
                        name,
                        user_id,
                        m.group(1),
                    )
                    await self._api.send_event(
                        "join",
                        self._location,
                        name,
                        user_id,
                        self._internal_ids.get(name),
                        ts,
                    )
            return

        # OnPlayerLeft
        m = _PLAYER_LEFT.search(line)
        if m:
            name, user_id = m.group(1), m.group(2)
            self._pending_joins.pop(name, None)

            # 正常にJoinされないユーザーは 「Restored player N」 が
            # 出ないためJoin未確定のユーザーはLEAVE判定をスキップする
            try:
                self._pending_names.remove(name)
            except ValueError:
                pass
            if name not in self._internal_ids:
                return

            logger.info(
                "LEAVE [%s] %s (%s) internal_id=%s",
                ts,
                name,
                user_id,
                self._internal_ids.get(name),
            )
            await self._api.send_event(
                "leave",
                self._location,
                name,
                user_id,
                self._internal_ids.get(name),
                ts,
            )
            self._internal_ids.pop(name, None)
            return

        # インスタンス移動 or アプリ終了時
        # Destroying <local_name> で残留者を一括退室扱いにする
        #
        # 厳密には不正確であるが、マスター固定用は常にインスタンスにいるのでこの扱いとする
        if self._local_name and "Destroying" in line:
            m = _DESTROYING.search(line)
            if m and m.group(1) == self._local_name:
                await self._close_current_location(ts)
