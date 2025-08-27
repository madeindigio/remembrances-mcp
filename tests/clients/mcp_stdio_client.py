#!/usr/bin/env python3
"""
Simple MCP stdio client used for smoke-testing the remembrances-mcp binary.

Behavior:
- Starts the binary as a subprocess (stdin/stdout pipes)
- Sends an `initialize` JSON-RPC request
- Calls `tools/list` to enumerate available tools
- Calls `tools/call` for a small save/get/delete fact workflow
- Prints responses to stdout

Usage:
  python3 tests/clients/mcp_stdio_client.py --binary ./dist/remembrances-mcp

Note: the binary must be built with `tests/build.sh` before running this client.
"""

import argparse
import json
import subprocess
import time
import threading
import sys
import os

DELIM = "\n"


def send_json(pipe, obj):
    data = json.dumps(obj, separators=(",", ":")) + DELIM
    pipe.write(data.encode())
    pipe.flush()


class ReaderThread(threading.Thread):
    def __init__(self, stream):
        super().__init__(daemon=True)
        self.stream = stream
        self.lines = []
        self.lock = threading.Lock()
        self.running = True

    def run(self):
        while self.running:
            line = self.stream.readline()
            if not line:
                break
            try:
                text = line.decode().strip()
            except Exception:
                text = line.strip()
            if text == "":
                continue
            with self.lock:
                self.lines.append(text)

    def pop_lines(self):
        with self.lock:
            l = self.lines[:]
            self.lines.clear()
        return l


def wait_for_response(reader, timeout=15):
    deadline = time.time() + timeout
    while time.time() < deadline:
        lines = reader.pop_lines()
        if lines:
            # Return the first JSON parseable line
            for ln in lines:
                try:
                    return json.loads(ln)
                except Exception:
                    # keep searching
                    print("Non-JSON line:", ln, file=sys.stderr)
        time.sleep(0.1)
    raise TimeoutError("Timed out waiting for response")


def main():
    p = argparse.ArgumentParser()
    p.add_argument("--binary", required=True)
    args = p.parse_args()

    bin_path = os.path.abspath(args.binary)
    if not os.path.exists(bin_path):
        print("Binary not found:", bin_path, file=sys.stderr)
        sys.exit(2)

    proc = subprocess.Popen([bin_path], stdin=subprocess.PIPE,
                            stdout=subprocess.PIPE, stderr=subprocess.STDOUT)
    reader = ReaderThread(proc.stdout)
    reader.start()

    try:
        # Initialize (JSON-RPC initialize)
        init_req = {
            "jsonrpc": "2.0",
            "id": 1,
            "method": "initialize",
            "params": {
                "clientInfo": {"name": "mcp-py-tests", "version": "0.1"},
                "protocolVersion": "2025-03-26"
            }
        }
        send_json(proc.stdin, init_req)
        print("Sent initialize")
        resp = wait_for_response(reader, timeout=10)
        print("initialize response:", json.dumps(resp, indent=2))

        # List tools
        list_req = {"jsonrpc": "2.0", "id": 2,
                    "method": "tools/list", "params": {}}
        send_json(proc.stdin, list_req)
        resp = wait_for_response(reader)
        print("tools/list response:", json.dumps(resp, indent=2))

        # Call save_fact
        call_id = 3
        save_params = {"name": "remembrance_save_fact", "arguments": {
            "user_id": "pytest_user", "key": "fav_color", "value": "verde"}}
        call_req = {"jsonrpc": "2.0", "id": call_id,
                    "method": "tools/call", "params": save_params}
        send_json(proc.stdin, call_req)
        resp = wait_for_response(reader)
        print("save_fact response:", json.dumps(resp, indent=2))

        # Call get_fact
        call_id += 1
        get_params = {"name": "remembrance_get_fact", "arguments": {
            "user_id": "pytest_user", "key": "fav_color"}}
        call_req = {"jsonrpc": "2.0", "id": call_id,
                    "method": "tools/call", "params": get_params}
        send_json(proc.stdin, call_req)
        resp = wait_for_response(reader)
        print("get_fact response:", json.dumps(resp, indent=2))

        # Call delete_fact
        call_id += 1
        del_params = {"name": "remembrance_delete_fact", "arguments": {
            "user_id": "pytest_user", "key": "fav_color"}}
        call_req = {"jsonrpc": "2.0", "id": call_id,
                    "method": "tools/call", "params": del_params}
        send_json(proc.stdin, call_req)
        resp = wait_for_response(reader)
        print("delete_fact response:", json.dumps(resp, indent=2))

    finally:
        try:
            proc.send_signal(subprocess.signal.SIGINT)
        except Exception:
            pass
        proc.wait(timeout=5)


if __name__ == "__main__":
    main()
