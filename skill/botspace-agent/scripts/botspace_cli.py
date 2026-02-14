#!/usr/bin/env python3
"""Botspace command-line client for registration, chat monitoring, and updates."""

from __future__ import annotations

import argparse
import json
import os
import subprocess
import sys
import time
from pathlib import Path
from typing import Any
from urllib.error import HTTPError, URLError
from urllib.parse import urlencode
from urllib.request import Request, urlopen


DEFAULT_API_URL = "http://localhost:8080/api/v1"
DEFAULT_STATE_FILE = ".botspace/state.json"
DEFAULT_LIMIT = 30
DEFAULT_POLL_INTERVAL = 5.0
REQUEST_TIMEOUT_SECONDS = 30.0


class CliError(Exception):
    """Known command failure with optional HTTP status."""

    def __init__(self, message: str, status: int | None = None) -> None:
        super().__init__(message)
        self.status = status


def fail(message: str, status: int | None = None) -> None:
    raise CliError(message, status=status)


def read_json_file(path: Path) -> Any:
    try:
        raw = path.read_text(encoding="utf-8")
    except OSError as exc:
        fail(f"failed to read file '{path}': {exc}")

    try:
        return json.loads(raw)
    except json.JSONDecodeError as exc:
        fail(f"invalid JSON in '{path}': {exc}")


def load_openclaw_config(account: str | None = None) -> dict[str, Any]:
    """Read credentials from ~/.openclaw/openclaw.json under channels.claw-swarm.accounts."""
    config_path = Path.home() / ".openclaw" / "openclaw.json"
    if not config_path.exists():
        return {}
    try:
        raw = config_path.read_text(encoding="utf-8")
        data = json.loads(raw)
    except (OSError, json.JSONDecodeError):
        return {}
    accounts = (
        data.get("channels", {})
        .get("claw-swarm", {})
        .get("accounts", {})
    )
    if not isinstance(accounts, dict) or not accounts:
        return {}
    if account:
        entry = accounts.get(account)
        if not isinstance(entry, dict):
            return {}
    else:
        entry = next(
            (v for v in accounts.values() if isinstance(v, dict) and v.get("enabled")),
            None,
        )
        if entry is None:
            return {}
    result: dict[str, Any] = {}
    for key in ("token", "apiUrl", "botSpaceId", "botId", "botName"):
        val = entry.get(key)
        if isinstance(val, str) and val:
            result[key] = val
    return result


def load_state(path: str) -> dict[str, Any]:
    state_path = Path(path)
    if not state_path.exists():
        return {}

    data = read_json_file(state_path)
    if not isinstance(data, dict):
        fail(f"state file '{state_path}' must contain a JSON object")
    return data


def save_state(path: str, state: dict[str, Any]) -> None:
    state_path = Path(path)
    state_path.parent.mkdir(parents=True, exist_ok=True)
    tmp_path = state_path.with_name(state_path.name + ".tmp")
    payload = json.dumps(state, indent=2, sort_keys=True) + "\n"
    try:
        tmp_path.write_text(payload, encoding="utf-8")
        os.replace(tmp_path, state_path)
        os.chmod(state_path, 0o600)
    except OSError as exc:
        fail(f"failed to write state file '{state_path}': {exc}")


def resolve_api_url(args: argparse.Namespace, state: dict[str, Any]) -> str:
    if args.api_url:
        return args.api_url.rstrip("/")
    env_url = os.getenv("BOTSPACE_API_URL")
    if env_url:
        return env_url.rstrip("/")
    state_url = state.get("apiUrl")
    if isinstance(state_url, str) and state_url:
        return state_url.rstrip("/")
    return DEFAULT_API_URL


def resolve_token(args: argparse.Namespace, state: dict[str, Any], required: bool = True) -> str | None:
    token = args.token or os.getenv("BOTSPACE_TOKEN") or state.get("token")
    if token:
        return str(token)
    if required:
        fail("missing token: pass --token, set BOTSPACE_TOKEN, or run register")
    return None


def resolve_space_id(args: argparse.Namespace, state: dict[str, Any], required: bool = True) -> str | None:
    value = getattr(args, "space_id", None) or state.get("botSpaceId")
    if value:
        return str(value)
    if required:
        fail("missing bot space id: pass --space-id or run register first")
    return None


def request_json(
    api_url: str,
    method: str,
    path: str,
    *,
    token: str | None = None,
    query: dict[str, Any] | None = None,
    body: dict[str, Any] | None = None,
) -> Any:
    clean_path = path if path.startswith("/") else f"/{path}"
    url = f"{api_url.rstrip('/')}{clean_path}"
    if query:
        filtered = {k: str(v) for k, v in query.items() if v is not None}
        if filtered:
            url = f"{url}?{urlencode(filtered)}"

    headers = {"Accept": "application/json"}
    data: bytes | None = None
    if body is not None:
        data = json.dumps(body).encode("utf-8")
        headers["Content-Type"] = "application/json"
    if token:
        headers["Authorization"] = f"Bearer {token}"

    req = Request(url=url, method=method, data=data, headers=headers)

    try:
        with urlopen(req, timeout=REQUEST_TIMEOUT_SECONDS) as resp:
            status = getattr(resp, "status", resp.getcode())
            raw = resp.read().decode("utf-8")
    except HTTPError as exc:
        raw_error = exc.read().decode("utf-8", errors="replace")
        message = f"HTTP {exc.code}"
        if raw_error:
            try:
                parsed = json.loads(raw_error)
                if isinstance(parsed, dict) and "error" in parsed:
                    message = f"HTTP {exc.code}: {parsed['error']}"
                else:
                    message = f"HTTP {exc.code}: {raw_error}"
            except json.JSONDecodeError:
                message = f"HTTP {exc.code}: {raw_error}"
        fail(message, status=exc.code)
    except URLError as exc:
        fail(f"request failed: {exc.reason}")
    except TimeoutError:
        fail("request timed out")

    if status == 204 or not raw:
        return None
    try:
        return json.loads(raw)
    except json.JSONDecodeError:
        fail("server returned non-JSON response")
    return None


def sorted_messages(messages: list[dict[str, Any]]) -> list[dict[str, Any]]:
    return sorted(messages, key=lambda msg: str(msg.get("createdAt", "")))


def format_message(msg: dict[str, Any]) -> str:
    created_at = msg.get("createdAt", "?")
    sender_name = msg.get("senderName", "unknown")
    sender_type = msg.get("senderType", "unknown")
    content = msg.get("content", "")
    return f"[{created_at}] {sender_name} ({sender_type}): {content}"


def print_json(data: Any) -> None:
    print(json.dumps(data, ensure_ascii=True), flush=True)


def print_message_batch(data: dict[str, Any]) -> None:
    messages = data.get("messages", [])
    if isinstance(messages, list):
        for msg in sorted_messages([m for m in messages if isinstance(m, dict)]):
            print(format_message(msg))
    count = data.get("count", len(messages) if isinstance(messages, list) else 0)
    has_more = bool(data.get("hasMore", False))
    print(f"count={count} hasMore={str(has_more).lower()}")


def run_reaction_command(command: str, message: dict[str, Any]) -> None:
    payload = json.dumps(message, ensure_ascii=True) + "\n"
    result = subprocess.run(
        command,
        input=payload,
        text=True,
        shell=True,
        capture_output=True,
    )
    if result.returncode == 0:
        return
    err = (result.stderr or result.stdout).strip()
    if not err:
        err = f"exit code {result.returncode}"
    print(f"reaction command failed: {err}", file=sys.stderr)


def emit_follow_messages(
    messages: list[dict[str, Any]],
    output: str,
    on_message_cmd: str | None,
) -> str | None:
    last_id: str | None = None
    for message in sorted_messages(messages):
        msg_id = message.get("id")
        if isinstance(msg_id, str):
            last_id = msg_id
        if output == "json":
            print_json({"type": "message", "message": message})
        else:
            print(format_message(message), flush=True)
        if on_message_cmd:
            run_reaction_command(on_message_cmd, message)
    return last_id


def cmd_register(args: argparse.Namespace, state: dict[str, Any], api_url: str) -> int:
    resp = request_json(
        api_url,
        "POST",
        "/auth/bots/register",
        body={
            "joinCode": args.join_code,
            "name": args.name,
            "capabilities": args.capabilities,
        },
    )
    if not isinstance(resp, dict):
        fail("unexpected register response")

    token = resp.get("token")
    bot = resp.get("bot")
    bot_space = resp.get("botSpace")
    if not isinstance(token, str) or not isinstance(bot, dict) or not isinstance(bot_space, dict):
        fail("registration response missing required fields")

    next_state = dict(state)
    next_state.update(
        {
            "token": token,
            "botId": bot.get("id"),
            "botSpaceId": bot_space.get("id"),
            "isManager": bool(bot.get("isManager", False)),
            "apiUrl": api_url,
        }
    )
    save_state(args.state_file, next_state)

    if args.output == "json":
        print_json(resp)
        return 0

    bot_name = bot.get("name", args.name)
    space_name = bot_space.get("name", "unknown-space")
    print(f"registered bot '{bot_name}' in space '{space_name}'")
    print(f"state saved to {args.state_file}")
    return 0


def cmd_me(args: argparse.Namespace, state: dict[str, Any], api_url: str) -> int:
    token = resolve_token(args, state)
    resp = request_json(api_url, "GET", "/auth/me", token=token)
    if args.output == "json":
        print_json(resp)
        return 0

    if isinstance(resp, dict) and "botSpaceId" in resp:
        print(
            f"bot id={resp.get('id')} name={resp.get('name')} "
            f"space={resp.get('botSpaceId')} manager={resp.get('isManager')}"
        )
    elif isinstance(resp, dict):
        print(f"user id={resp.get('id')} email={resp.get('email')}")
    else:
        print("authenticated identity returned")
    return 0


def cmd_overall(args: argparse.Namespace, state: dict[str, Any], api_url: str) -> int:
    token = resolve_token(args, state)
    space_id = resolve_space_id(args, state)
    resp = request_json(
        api_url,
        "GET",
        f"/bot-spaces/{space_id}/overall",
        token=token,
        query={"limit": args.limit},
    )
    if args.output == "json":
        print_json(resp)
        return 0

    if not isinstance(resp, dict):
        print("empty overall response")
        return 0
    summary = resp.get("summary")
    if isinstance(summary, dict) and isinstance(summary.get("content"), str):
        print("summary:")
        print(summary["content"])
        print("")
    else:
        print("summary: (none)\n")

    messages = resp.get("messages")
    if isinstance(messages, dict):
        print("messages:")
        print_message_batch(messages)
    else:
        print("messages: (none)")
    return 0


def fetch_messages(
    api_url: str,
    token: str,
    space_id: str,
    *,
    limit: int,
    before: str | None = None,
    since_id: str | None = None,
) -> dict[str, Any]:
    if since_id:
        resp = request_json(
            api_url,
            "GET",
            f"/bot-spaces/{space_id}/messages/since/{since_id}",
            token=token,
            query={"limit": limit},
        )
    else:
        resp = request_json(
            api_url,
            "GET",
            f"/bot-spaces/{space_id}/messages",
            token=token,
            query={"limit": limit, "before": before},
        )

    if not isinstance(resp, dict):
        fail("unexpected messages response")
    return resp


def cmd_messages_once(args: argparse.Namespace, token: str, space_id: str, api_url: str) -> int:
    if args.before and args.since_id:
        fail("--before and --since-id cannot be used together")
    resp = fetch_messages(
        api_url,
        token,
        space_id,
        limit=args.limit,
        before=args.before,
        since_id=args.since_id,
    )
    if args.output == "json":
        print_json(resp)
        return 0
    print_message_batch(resp)
    return 0


def resolve_follow_start_cursor(args: argparse.Namespace, token: str, space_id: str, api_url: str) -> str | None:
    if args.since_id:
        return args.since_id

    latest = fetch_messages(api_url, token, space_id, limit=1)
    raw_messages = latest.get("messages", [])
    if not isinstance(raw_messages, list) or not raw_messages:
        return None
    first = raw_messages[0]
    if not isinstance(first, dict):
        return None
    msg_id = first.get("id")
    return str(msg_id) if isinstance(msg_id, str) else None


def cmd_messages_follow(args: argparse.Namespace, token: str, space_id: str, api_url: str) -> int:
    if args.before:
        fail("--before cannot be used with --follow")
    if args.interval <= 0:
        fail("--interval must be greater than 0")

    cursor = resolve_follow_start_cursor(args, token, space_id, api_url)
    if args.output == "text":
        if cursor:
            print(f"monitoring {space_id}, starting after message {cursor}")
        else:
            print(f"monitoring {space_id}, waiting for first message")

    while True:
        try:
            if cursor is None:
                latest = fetch_messages(api_url, token, space_id, limit=args.limit)
                raw_messages = latest.get("messages", [])
                messages = [m for m in raw_messages if isinstance(m, dict)] if isinstance(raw_messages, list) else []
                if messages:
                    new_cursor = emit_follow_messages(messages, args.output, args.on_message_cmd)
                    if new_cursor:
                        cursor = new_cursor
                time.sleep(args.interval)
                continue

            drained_any = False
            while True:
                resp = fetch_messages(
                    api_url,
                    token,
                    space_id,
                    limit=args.limit,
                    since_id=cursor,
                )
                raw_messages = resp.get("messages", [])
                messages = [m for m in raw_messages if isinstance(m, dict)] if isinstance(raw_messages, list) else []
                if not messages:
                    break
                new_cursor = emit_follow_messages(messages, args.output, args.on_message_cmd)
                if new_cursor:
                    cursor = new_cursor
                drained_any = True
                if not bool(resp.get("hasMore", False)):
                    break

            if not drained_any:
                time.sleep(args.interval)
        except CliError as exc:
            if exc.status in (401, 403):
                raise
            print(f"warning: {exc}", file=sys.stderr)
            time.sleep(args.interval)
        except KeyboardInterrupt:
            if args.output == "text":
                print("\nstopped monitoring")
            return 0


def cmd_messages(args: argparse.Namespace, state: dict[str, Any], api_url: str) -> int:
    token = resolve_token(args, state)
    space_id = resolve_space_id(args, state)
    if args.limit <= 0:
        fail("--limit must be greater than 0")
    if args.follow:
        return cmd_messages_follow(args, token, space_id, api_url)
    return cmd_messages_once(args, token, space_id, api_url)


def cmd_send(args: argparse.Namespace, state: dict[str, Any], api_url: str) -> int:
    token = resolve_token(args, state)
    space_id = resolve_space_id(args, state)
    resp = request_json(
        api_url,
        "POST",
        f"/bot-spaces/{space_id}/messages",
        token=token,
        body={"content": args.content},
    )
    if args.output == "json":
        print_json(resp)
        return 0

    if isinstance(resp, dict):
        print(f"sent message id={resp.get('id')}")
    else:
        print("sent message")
    return 0


def cmd_bots(args: argparse.Namespace, state: dict[str, Any], api_url: str) -> int:
    token = resolve_token(args, state)
    space_id = resolve_space_id(args, state)
    resp = request_json(api_url, "GET", f"/bot-spaces/{space_id}/bots", token=token)
    if args.output == "json":
        print_json(resp)
        return 0

    if isinstance(resp, list):
        for item in resp:
            if isinstance(item, dict):
                print(
                    f"{item.get('id')} name={item.get('name')} "
                    f"manager={item.get('isManager')} lastSeen={item.get('lastSeenAt')}"
                )
    else:
        print("no bots returned")
    return 0


def cmd_statuses(args: argparse.Namespace, state: dict[str, Any], api_url: str) -> int:
    token = resolve_token(args, state)
    space_id = resolve_space_id(args, state)
    resp = request_json(api_url, "GET", f"/bot-spaces/{space_id}/statuses", token=token)
    if args.output == "json":
        print_json(resp)
        return 0

    if isinstance(resp, list):
        for item in resp:
            if isinstance(item, dict):
                print(f"{item.get('botId')} ({item.get('botName')}): {item.get('status')}")
    else:
        print("no statuses returned")
    return 0


def cmd_status_get(args: argparse.Namespace, state: dict[str, Any], api_url: str) -> int:
    token = resolve_token(args, state)
    space_id = resolve_space_id(args, state)
    resp = request_json(
        api_url,
        "GET",
        f"/bot-spaces/{space_id}/statuses/{args.bot_id}",
        token=token,
    )
    if args.output == "json":
        print_json(resp)
        return 0

    if isinstance(resp, dict):
        print(f"{resp.get('botId')} ({resp.get('botName')}): {resp.get('status')}")
    else:
        print("status not returned")
    return 0


def cmd_status_set(args: argparse.Namespace, state: dict[str, Any], api_url: str) -> int:
    token = resolve_token(args, state)
    space_id = resolve_space_id(args, state)
    resp = request_json(
        api_url,
        "PUT",
        f"/bot-spaces/{space_id}/statuses/{args.bot_id}",
        token=token,
        body={"status": args.status},
    )
    if args.output == "json":
        print_json(resp)
        return 0

    if isinstance(resp, dict):
        print(f"updated {resp.get('botId')} ({resp.get('botName')}): {resp.get('status')}")
    else:
        print("status updated")
    return 0


def load_bulk_statuses(path: str) -> list[dict[str, Any]]:
    data = read_json_file(Path(path))
    items: Any
    if isinstance(data, dict) and "statuses" in data:
        items = data["statuses"]
    else:
        items = data

    if not isinstance(items, list):
        fail("bulk status file must be a JSON array or {'statuses': [...]}")

    statuses: list[dict[str, Any]] = []
    for idx, item in enumerate(items):
        if not isinstance(item, dict):
            fail(f"bulk status item #{idx} must be an object")
        bot_id = item.get("botId")
        status = item.get("status")
        if not isinstance(bot_id, str) or not bot_id:
            fail(f"bulk status item #{idx} missing botId")
        if not isinstance(status, str) or not status:
            fail(f"bulk status item #{idx} missing status")
        statuses.append({"botId": bot_id, "status": status})
    return statuses


def cmd_status_bulk(args: argparse.Namespace, state: dict[str, Any], api_url: str) -> int:
    token = resolve_token(args, state)
    space_id = resolve_space_id(args, state)
    statuses = load_bulk_statuses(args.file)
    resp = request_json(
        api_url,
        "PUT",
        f"/bot-spaces/{space_id}/statuses",
        token=token,
        body={"statuses": statuses},
    )
    if args.output == "json":
        print_json(resp)
        return 0

    if isinstance(resp, list):
        print(f"updated {len(resp)} statuses")
    else:
        print("bulk status update completed")
    return 0


def cmd_summary_get(args: argparse.Namespace, state: dict[str, Any], api_url: str) -> int:
    token = resolve_token(args, state)
    space_id = resolve_space_id(args, state)
    resp = request_json(api_url, "GET", f"/bot-spaces/{space_id}/summary", token=token)
    if args.output == "json":
        print_json(resp)
        return 0

    if isinstance(resp, dict):
        print(resp.get("content", ""))
    else:
        print("summary not returned")
    return 0


def cmd_summary_set(args: argparse.Namespace, state: dict[str, Any], api_url: str) -> int:
    token = resolve_token(args, state)
    space_id = resolve_space_id(args, state)
    resp = request_json(
        api_url,
        "PUT",
        f"/bot-spaces/{space_id}/summary",
        token=token,
        body={"content": args.content},
    )
    if args.output == "json":
        print_json(resp)
        return 0

    if isinstance(resp, dict):
        print(f"summary updated at {resp.get('updatedAt')}")
    else:
        print("summary updated")
    return 0


def cmd_skills(args: argparse.Namespace, state: dict[str, Any], api_url: str) -> int:
    token = resolve_token(args, state)
    space_id = resolve_space_id(args, state)
    resp = request_json(api_url, "GET", f"/bot-spaces/{space_id}/skills", token=token)
    if args.output == "json":
        print_json(resp)
        return 0

    if isinstance(resp, list):
        for item in resp:
            if isinstance(item, dict):
                tags = item.get("tags") or []
                tags_str = ",".join(tags) if isinstance(tags, list) else str(tags)
                print(
                    f"{item.get('id')} name={item.get('name')} "
                    f"bot={item.get('botName')} "
                    f"description={item.get('description')} "
                    f"tags=[{tags_str}]"
                )
    else:
        print("no skills returned")
    return 0


def cmd_skill_create(args: argparse.Namespace, state: dict[str, Any], api_url: str) -> int:
    token = resolve_token(args, state)
    space_id = resolve_space_id(args, state)
    body: dict[str, Any] = {
        "name": args.name,
        "description": args.description,
    }
    if args.tags:
        body["tags"] = [t.strip() for t in args.tags.split(",") if t.strip()]
    resp = request_json(
        api_url,
        "POST",
        f"/bot-spaces/{space_id}/skills",
        token=token,
        body=body,
    )
    if args.output == "json":
        print_json(resp)
        return 0

    if isinstance(resp, dict):
        print(f"created skill id={resp.get('id')} name={resp.get('name')}")
    else:
        print("skill created")
    return 0


def cmd_skill_update(args: argparse.Namespace, state: dict[str, Any], api_url: str) -> int:
    token = resolve_token(args, state)
    space_id = resolve_space_id(args, state)
    body: dict[str, Any] = {}
    if args.name is not None:
        body["name"] = args.name
    if args.description is not None:
        body["description"] = args.description
    if args.tags is not None:
        body["tags"] = [t.strip() for t in args.tags.split(",") if t.strip()]
    if not body:
        fail("at least one of --name, --description, or --tags is required")
    resp = request_json(
        api_url,
        "PUT",
        f"/bot-spaces/{space_id}/skills/{args.skill_id}",
        token=token,
        body=body,
    )
    if args.output == "json":
        print_json(resp)
        return 0

    if isinstance(resp, dict):
        print(f"updated skill id={resp.get('id')} name={resp.get('name')}")
    else:
        print("skill updated")
    return 0


def format_task(task: dict[str, Any]) -> str:
    status = task.get("status", "unknown")
    name = task.get("name", "unnamed")
    bot_id = task.get("botId") or "unassigned"
    return f"{task.get('id')} name={name} status={status} bot={bot_id}"


def cmd_tasks(args: argparse.Namespace, state: dict[str, Any], api_url: str) -> int:
    token = resolve_token(args, state)
    space_id = resolve_space_id(args, state)
    query: dict[str, Any] = {}
    if args.status:
        query["status"] = args.status
    resp = request_json(
        api_url,
        "GET",
        f"/bot-spaces/{space_id}/tasks",
        token=token,
        query=query if query else None,
    )
    if args.output == "json":
        print_json(resp)
        return 0

    if isinstance(resp, list):
        for item in resp:
            if isinstance(item, dict):
                print(format_task(item))
    else:
        print("no tasks returned")
    return 0


def cmd_task_current(args: argparse.Namespace, state: dict[str, Any], api_url: str) -> int:
    token = resolve_token(args, state)
    space_id = resolve_space_id(args, state)
    resp = request_json(
        api_url,
        "GET",
        f"/bot-spaces/{space_id}/tasks/current",
        token=token,
    )
    if args.output == "json":
        print_json(resp)
        return 0

    if isinstance(resp, dict):
        print(format_task(resp))
        print(f"description: {resp.get('description', '')}")
    else:
        print("no active task")
    return 0


def cmd_task_accept(args: argparse.Namespace, state: dict[str, Any], api_url: str) -> int:
    token = resolve_token(args, state)
    space_id = resolve_space_id(args, state)
    resp = request_json(
        api_url,
        "POST",
        f"/bot-spaces/{space_id}/tasks/{args.task_id}/accept",
        token=token,
    )
    if args.output == "json":
        print_json(resp)
        return 0

    if isinstance(resp, dict):
        print(f"accepted task: {resp.get('name')}")
    else:
        print("task accepted")
    return 0


def cmd_task_complete(args: argparse.Namespace, state: dict[str, Any], api_url: str) -> int:
    token = resolve_token(args, state)
    space_id = resolve_space_id(args, state)
    resp = request_json(
        api_url,
        "POST",
        f"/bot-spaces/{space_id}/tasks/{args.task_id}/complete",
        token=token,
    )
    if args.output == "json":
        print_json(resp)
        return 0

    if isinstance(resp, dict):
        print(f"completed task: {resp.get('name')}")
    else:
        print("task completed")
    return 0


def cmd_task_block(args: argparse.Namespace, state: dict[str, Any], api_url: str) -> int:
    token = resolve_token(args, state)
    space_id = resolve_space_id(args, state)
    resp = request_json(
        api_url,
        "POST",
        f"/bot-spaces/{space_id}/tasks/{args.task_id}/block",
        token=token,
    )
    if args.output == "json":
        print_json(resp)
        return 0

    if isinstance(resp, dict):
        print(f"blocked task: {resp.get('name')}")
    else:
        print("task blocked")
    return 0


def cmd_task_create(args: argparse.Namespace, state: dict[str, Any], api_url: str) -> int:
    token = resolve_token(args, state)
    space_id = resolve_space_id(args, state)
    body: dict[str, Any] = {
        "name": args.name,
        "description": args.description,
    }
    if args.bot_id:
        body["botId"] = args.bot_id
    resp = request_json(
        api_url,
        "POST",
        f"/bot-spaces/{space_id}/tasks",
        token=token,
        body=body,
    )
    if args.output == "json":
        print_json(resp)
        return 0

    if isinstance(resp, dict):
        print(f"created task id={resp.get('id')} name={resp.get('name')} status={resp.get('status')}")
    else:
        print("task created")
    return 0


def cmd_task_assign(args: argparse.Namespace, state: dict[str, Any], api_url: str) -> int:
    token = resolve_token(args, state)
    space_id = resolve_space_id(args, state)
    resp = request_json(
        api_url,
        "POST",
        f"/bot-spaces/{space_id}/tasks/{args.task_id}/assign",
        token=token,
        body={"botId": args.bot_id},
    )
    if args.output == "json":
        print_json(resp)
        return 0

    if isinstance(resp, dict):
        print(f"assigned task '{resp.get('name')}' to bot {resp.get('botId')}")
    else:
        print("task assigned")
    return 0


def cmd_skill_delete(args: argparse.Namespace, state: dict[str, Any], api_url: str) -> int:
    token = resolve_token(args, state)
    space_id = resolve_space_id(args, state)
    request_json(
        api_url,
        "DELETE",
        f"/bot-spaces/{space_id}/skills/{args.skill_id}",
        token=token,
    )
    if args.output == "json":
        print_json({"deleted": True, "skillId": args.skill_id})
        return 0

    print(f"deleted skill {args.skill_id}")
    return 0


def add_shared_flags(parser: argparse.ArgumentParser) -> None:
    parser.add_argument("--api-url", default=None, help="Base API URL (example: http://localhost:8080/api/v1)")
    parser.add_argument(
        "--state-file",
        default=os.getenv("BOTSPACE_STATE_FILE", DEFAULT_STATE_FILE),
        help="Path to local state JSON file",
    )
    parser.add_argument("--token", help="Bearer token override for this invocation")
    parser.add_argument("--output", choices=["text", "json"], default="text", help="Output format")
    parser.add_argument(
        "--account",
        default=None,
        help="Account name in ~/.openclaw/openclaw.json (default: first enabled)",
    )


def add_space_flag(parser: argparse.ArgumentParser) -> None:
    parser.add_argument("--space-id", help="Bot space ID (defaults to saved state)")


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description="Botspace CLI")
    add_shared_flags(parser)
    subparsers = parser.add_subparsers(dest="command", required=True)

    register = subparsers.add_parser("register", help="Register bot with join code")
    register.add_argument("--join-code", required=True, help="Join code or manager join code")
    register.add_argument("--name", required=True, help="Bot display name")
    register.add_argument("--capabilities", required=True, help="Capabilities summary")

    subparsers.add_parser("me", help="Show authenticated identity")

    overall = subparsers.add_parser("overall", help="Fetch recent messages and summary")
    add_space_flag(overall)
    overall.add_argument("--limit", type=int, default=DEFAULT_LIMIT, help="Max messages to request")

    messages = subparsers.add_parser("messages", help="Fetch or follow messages")
    add_space_flag(messages)
    messages.add_argument("--limit", type=int, default=DEFAULT_LIMIT, help="Max messages per request")
    messages.add_argument("--before", help="Cursor ID for pagination")
    messages.add_argument("--since-id", help="Fetch messages after this message ID")
    messages.add_argument("--follow", action="store_true", help="Continuously monitor new messages")
    messages.add_argument("--interval", type=float, default=DEFAULT_POLL_INTERVAL, help="Polling interval seconds")
    messages.add_argument("--on-message-cmd", help="Shell command to run for each new message JSON payload")

    send = subparsers.add_parser("send", help="Post a message")
    add_space_flag(send)
    send.add_argument("--content", required=True, help="Message content")

    bots = subparsers.add_parser("bots", help="List bots in the space")
    add_space_flag(bots)

    statuses = subparsers.add_parser("statuses", help="List bot statuses")
    add_space_flag(statuses)

    status_get = subparsers.add_parser("status-get", help="Get status for one bot")
    add_space_flag(status_get)
    status_get.add_argument("--bot-id", required=True, help="Bot ID")

    status_set = subparsers.add_parser("status-set", help="Set status for one bot (manager only)")
    add_space_flag(status_set)
    status_set.add_argument("--bot-id", required=True, help="Bot ID")
    status_set.add_argument("--status", required=True, help="Status text")

    status_bulk = subparsers.add_parser("status-bulk", help="Bulk update statuses from JSON file (manager only)")
    add_space_flag(status_bulk)
    status_bulk.add_argument("--file", required=True, help="Path to JSON file")

    summary_get = subparsers.add_parser("summary-get", help="Get current summary")
    add_space_flag(summary_get)

    summary_set = subparsers.add_parser("summary-set", help="Set current summary (manager only)")
    add_space_flag(summary_set)
    summary_set.add_argument("--content", required=True, help="Summary text")

    skills = subparsers.add_parser("skills", help="List all skills in the space")
    add_space_flag(skills)

    skill_create = subparsers.add_parser("skill-create", help="Create a new skill (bot token required)")
    add_space_flag(skill_create)
    skill_create.add_argument("--name", required=True, help="Skill name")
    skill_create.add_argument("--description", required=True, help="Skill description")
    skill_create.add_argument("--tags", help="Comma-separated tags")

    skill_update = subparsers.add_parser("skill-update", help="Update an existing skill (bot token required)")
    add_space_flag(skill_update)
    skill_update.add_argument("--skill-id", required=True, help="Skill ID")
    skill_update.add_argument("--name", help="New skill name")
    skill_update.add_argument("--description", help="New skill description")
    skill_update.add_argument("--tags", help="Comma-separated tags")

    skill_delete = subparsers.add_parser("skill-delete", help="Delete a skill (bot token required)")
    add_space_flag(skill_delete)
    skill_delete.add_argument("--skill-id", required=True, help="Skill ID")

    tasks = subparsers.add_parser("tasks", help="List tasks in the space")
    add_space_flag(tasks)
    tasks.add_argument("--status", help="Filter by status (available, in_progress, completed, blocked)")

    task_current = subparsers.add_parser("task-current", help="Show current in-progress task")
    add_space_flag(task_current)

    task_accept = subparsers.add_parser("task-accept", help="Accept an available task")
    add_space_flag(task_accept)
    task_accept.add_argument("--task-id", required=True, help="Task ID")

    task_complete = subparsers.add_parser("task-complete", help="Mark a task as completed")
    add_space_flag(task_complete)
    task_complete.add_argument("--task-id", required=True, help="Task ID")

    task_block = subparsers.add_parser("task-block", help="Mark a task as blocked")
    add_space_flag(task_block)
    task_block.add_argument("--task-id", required=True, help="Task ID")

    task_create = subparsers.add_parser("task-create", help="Create a new task (manager only)")
    add_space_flag(task_create)
    task_create.add_argument("--name", required=True, help="Task name")
    task_create.add_argument("--description", required=True, help="Task description")
    task_create.add_argument("--bot-id", help="Optionally assign to a bot immediately")

    task_assign = subparsers.add_parser("task-assign", help="Assign a task to a bot (manager only)")
    add_space_flag(task_assign)
    task_assign.add_argument("--task-id", required=True, help="Task ID")
    task_assign.add_argument("--bot-id", required=True, help="Bot ID to assign to")

    return parser


def run_command(args: argparse.Namespace, state: dict[str, Any], api_url: str) -> int:
    command = args.command
    if command == "register":
        return cmd_register(args, state, api_url)
    if command == "me":
        return cmd_me(args, state, api_url)
    if command == "overall":
        return cmd_overall(args, state, api_url)
    if command == "messages":
        return cmd_messages(args, state, api_url)
    if command == "send":
        return cmd_send(args, state, api_url)
    if command == "bots":
        return cmd_bots(args, state, api_url)
    if command == "statuses":
        return cmd_statuses(args, state, api_url)
    if command == "status-get":
        return cmd_status_get(args, state, api_url)
    if command == "status-set":
        return cmd_status_set(args, state, api_url)
    if command == "status-bulk":
        return cmd_status_bulk(args, state, api_url)
    if command == "summary-get":
        return cmd_summary_get(args, state, api_url)
    if command == "summary-set":
        return cmd_summary_set(args, state, api_url)
    if command == "skills":
        return cmd_skills(args, state, api_url)
    if command == "skill-create":
        return cmd_skill_create(args, state, api_url)
    if command == "skill-update":
        return cmd_skill_update(args, state, api_url)
    if command == "skill-delete":
        return cmd_skill_delete(args, state, api_url)
    if command == "tasks":
        return cmd_tasks(args, state, api_url)
    if command == "task-current":
        return cmd_task_current(args, state, api_url)
    if command == "task-accept":
        return cmd_task_accept(args, state, api_url)
    if command == "task-complete":
        return cmd_task_complete(args, state, api_url)
    if command == "task-block":
        return cmd_task_block(args, state, api_url)
    if command == "task-create":
        return cmd_task_create(args, state, api_url)
    if command == "task-assign":
        return cmd_task_assign(args, state, api_url)
    fail(f"unknown command: {command}")
    return 1


def main() -> int:
    parser = build_parser()
    args = parser.parse_args()
    try:
        openclaw = load_openclaw_config(args.account)
        state = load_state(args.state_file)
        merged = {**state, **{k: v for k, v in openclaw.items() if v}}
        api_url = resolve_api_url(args, merged)
        return run_command(args, merged, api_url)
    except CliError as exc:
        print(f"error: {exc}", file=sys.stderr)
        return 1


if __name__ == "__main__":
    sys.exit(main())
