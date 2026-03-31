"""
Tests for client/log_watcher.py

テスト方針:
- WatcherState: 実ファイルを使ったロード/セーブ
- LogWatcher._watch_file: 実ファイルへの書き込みでライン検出を確認
- LogWatcher._scan_loop / run: 実ディレクトリ上での統合テスト
- タイムアウト系は poll_interval / scan_interval / idle_timeout を短縮して実行
"""

import asyncio
import json
import sys
import time
from pathlib import Path

import pytest

# client/ を sys.path に追加
sys.path.insert(0, str(Path(__file__).parent.parent))

from log_watcher import LogWatcher, WatcherState


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------


async def collect_lines(
    watcher: LogWatcher,
    *,
    run_seconds: float,
) -> list[tuple[Path, str]]:
    """watcher.run() を run_seconds 秒後に stop して収集したラインを返す。"""
    collected: list[tuple[Path, str]] = []

    async def on_line(path: Path, line: str) -> None:
        collected.append((path, line))

    watcher.on_line = on_line

    async def _stop_after() -> None:
        await asyncio.sleep(run_seconds)
        watcher.stop()

    await asyncio.gather(watcher.run(), _stop_after(), return_exceptions=True)
    return collected


# ---------------------------------------------------------------------------
# WatcherState
# ---------------------------------------------------------------------------


class TestWatcherState:
    def test_load_missing_file(self, tmp_path: Path) -> None:
        state = WatcherState(tmp_path / "state.json")
        assert state.load() == {}

    def test_load_broken_json(self, tmp_path: Path) -> None:
        f = tmp_path / "state.json"
        f.write_text("NOT JSON", encoding="utf-8")
        state = WatcherState(f)
        assert state.load() == {}

    def test_save_and_load_roundtrip(self, tmp_path: Path) -> None:
        state_file = tmp_path / "state.json"
        state = WatcherState(state_file)

        offsets = {tmp_path / "a.txt": 100, tmp_path / "b.txt": 200}
        state.save(offsets)

        assert state_file.exists()
        loaded = state.load()
        assert loaded == offsets

    def test_save_is_atomic(self, tmp_path: Path) -> None:
        """保存中に .tmp が残らないことを確認。"""
        state_file = tmp_path / "state.json"
        state = WatcherState(state_file)
        state.save({tmp_path / "x.txt": 42})

        tmp = state_file.with_suffix(".tmp")
        assert not tmp.exists()
        assert state_file.exists()

    def test_load_ignores_nonexistent_paths(self, tmp_path: Path) -> None:
        """ロード自体はパスの存在を問わず返す（存在チェックは LogWatcher 側）。"""
        state_file = tmp_path / "state.json"
        ghost = tmp_path / "ghost.txt"
        state = WatcherState(state_file)
        state.save({ghost: 999})

        loaded = state.load()
        assert loaded == {ghost: 999}


# ---------------------------------------------------------------------------
# LogWatcher — _watch_file（実ファイル）
# ---------------------------------------------------------------------------


class TestWatchFile:
    @pytest.fixture
    def log_file(self, tmp_path: Path) -> Path:
        p = tmp_path / "output_log_test.txt"
        p.write_text("", encoding="utf-8")
        return p

    def _make_watcher(
        self,
        tmp_path: Path,
        log_file: Path,
        *,
        read_from_end: bool = False,
    ) -> tuple[LogWatcher, list[tuple[Path, str]]]:
        collected: list[tuple[Path, str]] = []

        async def on_line(path: Path, line: str) -> None:
            collected.append((path, line))

        watcher = LogWatcher(
            log_dir=tmp_path,
            on_line=on_line,
            state_file=tmp_path / "state.json",
            pattern="output_log_*.txt",
            poll_interval=0.05,
            scan_interval=0.1,
            idle_timeout=10.0,
            read_from_end=read_from_end,
        )
        return watcher, collected

    @pytest.mark.asyncio
    async def test_reads_existing_lines(self, tmp_path: Path, log_file: Path) -> None:
        """起動前に書き込まれた行を先頭から読む。"""
        log_file.write_text("line1\nline2\nline3\n", encoding="utf-8")
        watcher, collected = self._make_watcher(tmp_path, log_file)

        async def _stop() -> None:
            await asyncio.sleep(0.5)
            watcher.stop()

        await asyncio.gather(watcher.run(), _stop(), return_exceptions=True)

        assert [line for _, line in collected] == ["line1", "line2", "line3"]

    @pytest.mark.asyncio
    async def test_detects_appended_lines(self, tmp_path: Path, log_file: Path) -> None:
        """起動後に追記された行を検出する。"""
        watcher, collected = self._make_watcher(tmp_path, log_file)

        async def _write_and_stop() -> None:
            await asyncio.sleep(0.2)
            with log_file.open("a", encoding="utf-8") as f:
                f.write("hello\nworld\n")
            await asyncio.sleep(0.4)
            watcher.stop()

        await asyncio.gather(watcher.run(), _write_and_stop(), return_exceptions=True)

        assert [line for _, line in collected] == ["hello", "world"]

    @pytest.mark.asyncio
    async def test_incomplete_line_held_until_newline(
        self, tmp_path: Path, log_file: Path
    ) -> None:
        """改行が来るまで不完全な行を配信しない。"""
        watcher, collected = self._make_watcher(tmp_path, log_file)

        async def _write_and_stop() -> None:
            await asyncio.sleep(0.2)
            # 改行なしで書く
            with log_file.open("a", encoding="utf-8") as f:
                f.write("partial")
            await asyncio.sleep(0.2)
            assert collected == [], "改行前に配信されてはいけない"
            # 改行を追記
            with log_file.open("a", encoding="utf-8") as f:
                f.write("\n")
            await asyncio.sleep(0.2)
            watcher.stop()

        await asyncio.gather(watcher.run(), _write_and_stop(), return_exceptions=True)

        assert [line for _, line in collected] == ["partial"]

    @pytest.mark.asyncio
    async def test_read_from_end(self, tmp_path: Path, log_file: Path) -> None:
        """read_from_end=True のとき起動前の行は読まない。"""
        log_file.write_text("old_line\n", encoding="utf-8")
        watcher, collected = self._make_watcher(tmp_path, log_file, read_from_end=True)

        async def _write_and_stop() -> None:
            await asyncio.sleep(0.2)
            with log_file.open("a", encoding="utf-8") as f:
                f.write("new_line\n")
            await asyncio.sleep(0.3)
            watcher.stop()

        await asyncio.gather(watcher.run(), _write_and_stop(), return_exceptions=True)

        lines = [line for _, line in collected]
        assert "old_line" not in lines
        assert "new_line" in lines

    @pytest.mark.asyncio
    async def test_multiline_burst(self, tmp_path: Path, log_file: Path) -> None:
        """大量ラインの一括書き込みをすべて受け取る。"""
        watcher, collected = self._make_watcher(tmp_path, log_file)

        expected = [f"line{i}" for i in range(100)]

        async def _write_and_stop() -> None:
            await asyncio.sleep(0.1)
            log_file.write_text("\n".join(expected) + "\n", encoding="utf-8")
            await asyncio.sleep(0.5)
            watcher.stop()

        await asyncio.gather(watcher.run(), _write_and_stop(), return_exceptions=True)

        assert [line for _, line in collected] == expected

    @pytest.mark.asyncio
    async def test_on_line_exception_does_not_stop_watcher(
        self, tmp_path: Path
    ) -> None:
        """on_line が例外を出してもウォッチャーが止まらない。"""
        log_file = tmp_path / "output_log_test.txt"
        log_file.write_text("", encoding="utf-8")

        call_count = 0

        async def flaky_on_line(path: Path, line: str) -> None:
            nonlocal call_count
            call_count += 1
            if call_count == 1:
                raise RuntimeError("intentional error")

        watcher = LogWatcher(
            log_dir=tmp_path,
            on_line=flaky_on_line,
            state_file=tmp_path / "state.json",
            poll_interval=0.05,
            scan_interval=0.1,
            idle_timeout=10.0,
        )

        async def _write_and_stop() -> None:
            await asyncio.sleep(0.1)
            log_file.write_text("line1\nline2\n", encoding="utf-8")
            await asyncio.sleep(0.4)
            watcher.stop()

        await asyncio.gather(watcher.run(), _write_and_stop(), return_exceptions=True)

        assert call_count == 2


# ---------------------------------------------------------------------------
# LogWatcher — idle_timeout
# ---------------------------------------------------------------------------


class TestIdleTimeout:
    @pytest.mark.asyncio
    async def test_idle_task_exits(self, tmp_path: Path) -> None:
        """idle_timeout を過ぎたらタスクが終了する。"""
        log_file = tmp_path / "output_log_idle.txt"
        log_file.write_text("init\n", encoding="utf-8")

        collected: list[str] = []

        async def on_line(path: Path, line: str) -> None:
            collected.append(line)

        watcher = LogWatcher(
            log_dir=tmp_path,
            on_line=on_line,
            state_file=tmp_path / "state.json",
            poll_interval=0.05,
            scan_interval=0.1,
            idle_timeout=0.3,  # 短縮
        )

        async def _stop() -> None:
            await asyncio.sleep(1.0)
            watcher.stop()

        await asyncio.gather(watcher.run(), _stop(), return_exceptions=True)

        # タスクが idle_timeout で終了していること
        assert log_file not in watcher._watch_tasks
        assert "init" in collected


# ---------------------------------------------------------------------------
# LogWatcher — 状態の永続化・再開
# ---------------------------------------------------------------------------


class TestStateResume:
    @pytest.mark.asyncio
    async def test_resumes_from_saved_offset(self, tmp_path: Path) -> None:
        """再起動時に保存済みオフセットから再開し、既読行を重複配信しない。"""
        log_file = tmp_path / "output_log_resume.txt"
        log_file.write_text("line1\nline2\n", encoding="utf-8")
        state_file = tmp_path / "state.json"

        collected: list[str] = []

        async def on_line(path: Path, line: str) -> None:
            collected.append(line)

        # 1回目の起動: line1, line2 を読む
        watcher1 = LogWatcher(
            log_dir=tmp_path,
            on_line=on_line,
            state_file=state_file,
            poll_interval=0.05,
            scan_interval=0.1,
            idle_timeout=10.0,
        )

        async def _stop1() -> None:
            await asyncio.sleep(0.5)
            watcher1.stop()

        await asyncio.gather(watcher1.run(), _stop1(), return_exceptions=True)
        assert collected == ["line1", "line2"]

        # 追記
        with log_file.open("a", encoding="utf-8") as f:
            f.write("line3\n")

        collected.clear()

        # 2回目の起動: line3 だけ読む
        watcher2 = LogWatcher(
            log_dir=tmp_path,
            on_line=on_line,
            state_file=state_file,
            poll_interval=0.05,
            scan_interval=0.1,
            idle_timeout=10.0,
        )

        async def _stop2() -> None:
            await asyncio.sleep(0.5)
            watcher2.stop()

        await asyncio.gather(watcher2.run(), _stop2(), return_exceptions=True)
        assert collected == ["line3"]

    @pytest.mark.asyncio
    async def test_stale_state_entry_removed(self, tmp_path: Path) -> None:
        """存在しないファイルのオフセットは起動時に削除される。"""
        ghost = tmp_path / "output_log_ghost.txt"
        state_file = tmp_path / "state.json"

        # ghost ファイルのオフセットだけ state に仕込む
        WatcherState(state_file).save({ghost: 100})

        async def on_line(path: Path, line: str) -> None:
            pass

        watcher = LogWatcher(
            log_dir=tmp_path,
            on_line=on_line,
            state_file=state_file,
            poll_interval=0.05,
            scan_interval=0.1,
            idle_timeout=10.0,
        )

        assert ghost not in watcher._offsets


# ---------------------------------------------------------------------------
# LogWatcher — 新規ファイル検出
# ---------------------------------------------------------------------------


class TestNewFileDetection:
    @pytest.mark.asyncio
    async def test_new_file_read_from_beginning(self, tmp_path: Path) -> None:
        """起動後に作られたファイルは先頭から全行読む。

        ファイル作成時にすでに複数行書き込まれていても、
        スキャンで検出されるまでの間に書かれた行を含め全行受け取ることを確認。
        """
        collected: list[str] = []

        async def on_line(path: Path, line: str) -> None:
            collected.append(line)

        watcher = LogWatcher(
            log_dir=tmp_path,
            on_line=on_line,
            state_file=tmp_path / "state.json",
            poll_interval=0.05,
            scan_interval=0.1,
            idle_timeout=10.0,
        )

        log_file = tmp_path / "output_log_new.txt"

        async def _create_and_stop() -> None:
            await asyncio.sleep(0.2)
            # ファイル作成と同時に複数行を書く（スキャン検出より前に書き込み済み）
            log_file.write_text("first\nsecond\nthird\n", encoding="utf-8")
            await asyncio.sleep(0.5)
            watcher.stop()

        await asyncio.gather(watcher.run(), _create_and_stop(), return_exceptions=True)

        # 先頭から全行受け取れていること（途中からだと first/second が欠ける）
        assert collected == ["first", "second", "third"]

    @pytest.mark.asyncio
    async def test_monitors_multiple_files(self, tmp_path: Path) -> None:
        """複数ファイルを同時に監視できる。"""
        files = [tmp_path / f"output_log_{i}.txt" for i in range(3)]
        for f in files:
            f.write_text("", encoding="utf-8")

        collected: list[tuple[Path, str]] = []

        async def on_line(path: Path, line: str) -> None:
            collected.append((path, line))

        watcher = LogWatcher(
            log_dir=tmp_path,
            on_line=on_line,
            state_file=tmp_path / "state.json",
            poll_interval=0.05,
            scan_interval=0.1,
            idle_timeout=10.0,
        )

        async def _write_and_stop() -> None:
            await asyncio.sleep(0.2)
            for i, f in enumerate(files):
                with f.open("a", encoding="utf-8") as fp:
                    fp.write(f"from_file_{i}\n")
            await asyncio.sleep(0.5)
            watcher.stop()

        await asyncio.gather(watcher.run(), _write_and_stop(), return_exceptions=True)

        for i, f in enumerate(files):
            assert (f, f"from_file_{i}") in collected


# ---------------------------------------------------------------------------
# LogWatcher — 追記なしスキップ → Resume
# ---------------------------------------------------------------------------


class TestNoNewDataSkipAndResume:
    @pytest.mark.asyncio
    async def test_no_new_data_does_not_start_watch_task(self, tmp_path: Path) -> None:
        """初回スキャン時、saved_offset >= current_size のファイルは監視タスクを起動しない。"""
        log_file = tmp_path / "output_log_skip.txt"
        log_file.write_text("already_read\n", encoding="utf-8")
        state_file = tmp_path / "state.json"

        # ファイルを全部読んだ状態のオフセットを保存
        WatcherState(state_file).save({log_file: log_file.stat().st_size})

        async def on_line(path: Path, line: str) -> None:
            pass

        watcher = LogWatcher(
            log_dir=tmp_path,
            on_line=on_line,
            state_file=state_file,
            poll_interval=0.05,
            scan_interval=0.1,
            idle_timeout=10.0,
        )

        async def _stop() -> None:
            await asyncio.sleep(0.3)
            watcher.stop()

        await asyncio.gather(watcher.run(), _stop(), return_exceptions=True)

        # _known_files には登録されているが watch タスクは立っていない
        assert log_file in watcher._known_files
        assert log_file not in watcher._watch_tasks

    @pytest.mark.asyncio
    async def test_resumes_watch_after_new_append(self, tmp_path: Path) -> None:
        """追記なしでスキップされたファイルに書き込まれたら監視タスクが再開し行を受け取る。"""
        log_file = tmp_path / "output_log_resume2.txt"
        log_file.write_text("old_line\n", encoding="utf-8")
        state_file = tmp_path / "state.json"

        WatcherState(state_file).save({log_file: log_file.stat().st_size})

        collected: list[str] = []

        async def on_line(path: Path, line: str) -> None:
            collected.append(line)

        watcher = LogWatcher(
            log_dir=tmp_path,
            on_line=on_line,
            state_file=state_file,
            poll_interval=0.05,
            scan_interval=0.1,
            idle_timeout=10.0,
        )

        async def _append_and_stop() -> None:
            # スキャンが1回走ってスキップされるのを待つ
            await asyncio.sleep(0.25)
            with log_file.open("a", encoding="utf-8") as f:
                f.write("new_line\n")
            # resume タスクが起動して行を読むのを待つ
            await asyncio.sleep(0.5)
            watcher.stop()

        await asyncio.gather(watcher.run(), _append_and_stop(), return_exceptions=True)

        assert "old_line" not in collected
        assert "new_line" in collected


# ---------------------------------------------------------------------------
# LogWatcher — シャットダウン時の状態保存
# ---------------------------------------------------------------------------


class TestStateSaveOnShutdown:
    @pytest.mark.asyncio
    async def test_state_saved_on_stop(self, tmp_path: Path) -> None:
        """stop() 後、run() の finally でオフセットがファイルに書き出される。"""
        log_file = tmp_path / "output_log_save.txt"
        log_file.write_text("line1\nline2\n", encoding="utf-8")
        state_file = tmp_path / "state.json"

        async def on_line(path: Path, line: str) -> None:
            pass

        watcher = LogWatcher(
            log_dir=tmp_path,
            on_line=on_line,
            state_file=state_file,
            poll_interval=0.05,
            scan_interval=0.1,
            idle_timeout=10.0,
        )

        async def _stop() -> None:
            await asyncio.sleep(0.5)
            watcher.stop()

        await asyncio.gather(watcher.run(), _stop(), return_exceptions=True)

        # run() 終了後にファイルが存在し、オフセットが記録されている
        assert state_file.exists()
        saved = WatcherState(state_file).load()
        assert log_file in saved
        assert saved[log_file] == log_file.stat().st_size
