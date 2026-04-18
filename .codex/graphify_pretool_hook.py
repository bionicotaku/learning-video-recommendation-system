#!/usr/bin/env python3

import json
import os
import re
import sys
import time
from pathlib import Path
from typing import Any


SEARCH_CMD_PATTERN = re.compile(r"(^|[|&;()\s])(rg|find|fd|grep|ls|tree)([|&;()\s]|$)")
STATE_FILE = ".codex/.graphify_pretool_state.json"
REMINDER_SECONDS = 60
REMINDER = "graphify: graph exists; read graphify-out/GRAPH_REPORT.md before raw file search."


def walk_strings(value: Any):
    if isinstance(value, str):
        yield value
        return
    if isinstance(value, dict):
        for child in value.values():
            yield from walk_strings(child)
        return
    if isinstance(value, list):
        for item in value:
            yield from walk_strings(item)


def extract_command_text(payload: Any) -> str:
    parts = []
    for text in walk_strings(payload):
        parts.append(text)
    return "\n".join(parts)


def detect_cwd(payload: Any) -> Path:
    if isinstance(payload, dict):
        for key in ("cwd", "working_directory", "workdir"):
            value = payload.get(key)
            if isinstance(value, str) and value:
                return Path(value)
    return Path.cwd()


def load_state(path: Path) -> dict[str, float]:
    try:
        return json.loads(path.read_text())
    except Exception:
        return {}


def save_state(path: Path, state: dict[str, float]) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(state))


def should_remind(cwd: Path, state_path: Path) -> bool:
    state = load_state(state_path)
    now = time.time()
    key = str(cwd.resolve())
    last = float(state.get(key, 0))
    if now-last < REMINDER_SECONDS:
        return False
    state[key] = now
    save_state(state_path, state)
    return True


def main() -> int:
    raw = sys.stdin.read()
    if not raw.strip():
        return 0

    try:
        payload = json.loads(raw)
    except json.JSONDecodeError:
        return 0

    cwd = detect_cwd(payload)
    graph_path = cwd / "graphify-out" / "graph.json"
    if not graph_path.is_file():
        return 0

    command_text = extract_command_text(payload)
    if not SEARCH_CMD_PATTERN.search(command_text):
        return 0

    state_path = cwd / STATE_FILE
    if not should_remind(cwd, state_path):
        return 0

    sys.stderr.write(f"{REMINDER}\n")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
