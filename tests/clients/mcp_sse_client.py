#!/usr/bin/env python3
"""
Simple MCP SSE/HTTP client used for smoke-testing the remembrances-mcp binary when
started with --sse.

Behavior:
- Starts the binary with --sse (if --start provided)
- Connects to the SSE endpoint (/sse) and reads the initial `endpoint` event
  which contains the message endpoint (e.g. /message?sessionID=...)
- Posts JSON-RPC requests (initialize, tools/list, tools/call) to the message endpoint
- Reads responses arriving as SSE `message` events

Usage:
  python3 tests/clients/mcp_sse_client.py --binary ./dist/remembrances-mcp --addr 127.0.0.1:4001

Notes:
- Requires `requests` package. Install with `pip install requests`
"""

import argparse
import json
import subprocess
import time
import sys
import os
import requests
from threading import Thread


def read_sse_events(resp):
    """Generator that yields (event, data) tuples from an SSE response."""
    event = None
    data_lines = []
    for raw in resp.iter_lines(decode_unicode=True):
        if raw is None:
            break
        line = raw.strip()
        if line == "":
            if event and data_lines:
                yield event, "\n".join(data_lines)
            event = None
            data_lines = []
            continue
        if line.startswith("event:"):
            event = line.split("event:", 1)[1].strip()
        elif line.startswith("data:"):
            data_lines.append(line.split("data:", 1)[1].strip())


def wait_for_endpoint(sse_url, timeout=15):
    """Open an SSE connection and wait for the initial 'endpoint' event.

    Returns a tuple (endpoint, resp) where `resp` is the active requests
    response object that must remain open while the session is used.
    """
    deadline = time.time() + timeout
    resp = requests.get(sse_url, stream=True, timeout=timeout)
    for ev, data in read_sse_events(resp):
        if ev == "endpoint":
            return data, resp
        if time.time() > deadline:
            break
    # If we reach here, close the response and raise
    try:
        resp.close()
    except Exception:
        pass
    raise TimeoutError("Timed out waiting for endpoint event on SSE stream")


def post_message(url, payload):
    headers = {"Content-Type": "application/json"}
    # Try several times in case the session is not fully established yet
    max_attempts = 5
    r = None
    for attempt in range(max_attempts):
        r = requests.post(url, data=json.dumps(payload), headers=headers)
        if r.status_code in (200, 202):
            return r
        # If server reports session closed, wait with linear backoff and retry
        if r.status_code == 400 and "session closed" in (r.text or ""):
            time.sleep(0.25 * (attempt + 1))
            continue
        raise RuntimeError(f"POST failed: {r.status_code} {r.text}")
    # Final attempt failed
    raise RuntimeError(f"POST failed after retries: {r.status_code} {r.text}")


def listen_responses(resp, stop_event, out_list):
    """Read SSE events from an already-open response object and append message events to out_list."""
    for ev, data in read_sse_events(resp):
        if ev == "message":
            try:
                out_list.append(json.loads(data))
            except Exception:
                out_list.append({"raw": data})
        if stop_event():
            break


def main():
    p = argparse.ArgumentParser()
    p.add_argument("--binary", required=True)
    p.add_argument("--addr", default="127.0.0.1:4001")
    p.add_argument("--start", action="store_true",
                   help="start the server subprocess")
    args = p.parse_args()

    bin_path = os.path.abspath(args.binary)
    if not os.path.exists(bin_path):
        print("Binary not found:", bin_path, file=sys.stderr)
        sys.exit(2)

    proc = None
    if args.start:
        env = os.environ.copy()
        env["GOMEM_OPENAI_KEY"] = env.get("GOMEM_OPENAI_KEY", "testkey")
        env["GOMEM_DB_PATH"] = env.get(
            "GOMEM_DB_PATH", os.path.abspath("./remembrances.db"))
        env["GOMEM_SSE_ADDR"] = ":4001"
        proc = subprocess.Popen(
            [bin_path, "--sse"], env=env, stdout=subprocess.PIPE, stderr=subprocess.STDOUT)
        # Give server a moment to start
        time.sleep(1.0)

    sse_url = f"http://{args.addr}/sse"
    print("Connecting to SSE url:", sse_url)

    # Perform the typical initialize/list/call workflow. Retry the whole
    # sequence a couple times if we hit a session closed error to reduce
    # flakiness caused by tight timing between SSE connect and message binding.
    attempts = 3
    last_exc = None
    for attempt in range(attempts):
        try:
            # Wait for endpoint event which provides message endpoint including sessionID
            endpoint, sse_resp = wait_for_endpoint(sse_url, timeout=20)
            print("Received endpoint:", endpoint)

            # The endpoint may be relative (e.g. /message?sessionID=...), make it absolute
            if endpoint.startswith("/"):
                message_url = f"http://{args.addr}{endpoint}"
            else:
                message_url = endpoint

            print("Message endpoint ->", message_url)

            # Start a background SSE listener reusing the open response to capture message events
            responses = []
            stop_flag = {"stop": False}

            def stop_event():
                return stop_flag["stop"]

            listener = Thread(target=listen_responses, args=(
                sse_resp, stop_event, responses), daemon=True)
            listener.start()

            # Send initialize
            init_req = {"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": {
                "clientInfo": {"name": "mcp-sse-py", "version": "0.1"}, "protocolVersion": "2025-03-26"}}
            post_message(message_url, init_req)
            time.sleep(0.5)

            # List tools
            list_req = {"jsonrpc": "2.0", "id": 2,
                        "method": "tools/list", "params": {}}
            post_message(message_url, list_req)
            time.sleep(0.5)

            # Call save/get/delete workflow
            save_req = {"jsonrpc": "2.0", "id": 10, "method": "tools/call", "params": {
                "name": "remembrance_save_fact", "arguments": {"user_id": "sse_user", "key": "pet", "value": "gato"}}}
            post_message(message_url, save_req)
            time.sleep(0.5)

            get_req = {"jsonrpc": "2.0", "id": 11, "method": "tools/call", "params": {
                "name": "remembrance_get_fact", "arguments": {"user_id": "sse_user", "key": "pet"}}}
            post_message(message_url, get_req)
            time.sleep(0.5)

            del_req = {"jsonrpc": "2.0", "id": 12, "method": "tools/call", "params": {
                "name": "remembrance_delete_fact", "arguments": {"user_id": "sse_user", "key": "pet"}}}
            post_message(message_url, del_req)

            # Give some time for responses to arrive
            time.sleep(2.0)
            stop_flag["stop"] = True
            try:
                sse_resp.close()
            except Exception:
                pass

            # If we reached here without exception, break out
            last_exc = None
            break
        except RuntimeError as e:
            last_exc = e
            print("SSE client attempt failed, retrying:", e)
            time.sleep(0.5)
            continue

    if last_exc:
        raise last_exc

    print("Captured responses:")
    for r in responses:
        print(json.dumps(r, indent=2))

    if proc:
        proc.send_signal(subprocess.signal.SIGINT)
        proc.wait(timeout=5)


if __name__ == "__main__":
    main()
