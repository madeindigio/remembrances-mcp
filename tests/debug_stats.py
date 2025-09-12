#!/usr/bin/env python3
"""
Debug test to check user stats persistence directly
"""

import json
import sys
import os
import subprocess
import time
import threading

# Add the current directory to path for importing helper functions
sys.path.append(os.path.dirname(__file__))

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
    # Find the binary
    bin_path = os.path.abspath("./dist/remembrances-mcp")
    if not os.path.exists(bin_path):
        print(f"Binary not found: {bin_path}")
        return False

    # Start the MCP server with stdio transport
    proc = subprocess.Popen([bin_path], stdin=subprocess.PIPE,
                            stdout=subprocess.PIPE, stderr=subprocess.STDOUT)
    reader = ReaderThread(proc.stdout)
    reader.start()

    request_id = 1

    try:
        # Initialize
        init_req = {
            "jsonrpc": "2.0",
            "id": request_id,
            "method": "initialize",
            "params": {
                "clientInfo": {"name": "mcp-debug-test", "version": "0.1"},
                "protocolVersion": "2025-03-26"
            }
        }
        request_id += 1
        send_json(proc.stdin, init_req)
        resp = wait_for_response(reader, timeout=10)

        if resp.get("error"):
            print(f"Initialize failed: {resp['error']}")
            return False

        print("✓ Connected to MCP server")

        # Get initial stats
        print("\n=== Getting initial stats ===")
        call_req = {
            "jsonrpc": "2.0",
            "id": request_id,
            "method": "tools/call",
            "params": {
                "name": "remembrance_get_stats",
                "arguments": {"user_id": "debug_user"}
            }
        }
        request_id += 1
        send_json(proc.stdin, call_req)
        resp = wait_for_response(reader)

        if resp.get("error"):
            print(f"Get stats failed: {resp['error']}")
        else:
            content = resp.get("result", {}).get("content", [])
            if content:
                print(f"Initial stats response: {content[0].get('text', '')}")

        # Save a fact
        print("\n=== Saving a fact ===")
        call_req = {
            "jsonrpc": "2.0",
            "id": request_id,
            "method": "tools/call",
            "params": {
                "name": "remembrance_save_fact",
                "arguments": {
                    "user_id": "debug_user",
                    "key": "debug_key",
                    "value": "debug_value"
                }
            }
        }
        request_id += 1
        send_json(proc.stdin, call_req)
        resp = wait_for_response(reader)

        if resp.get("error"):
            print(f"Save fact failed: {resp['error']}")
        else:
            content = resp.get("result", {}).get("content", [])
            if content:
                print(f"Save fact response: {content[0].get('text', '')}")

        # Get stats after saving
        print("\n=== Getting stats after saving fact ===")
        call_req = {
            "jsonrpc": "2.0",
            "id": request_id,
            "method": "tools/call",
            "params": {
                "name": "remembrance_get_stats",
                "arguments": {"user_id": "debug_user"}
            }
        }
        request_id += 1
        send_json(proc.stdin, call_req)
        resp = wait_for_response(reader)

        if resp.get("error"):
            print(f"Get stats failed: {resp['error']}")
        else:
            content = resp.get("result", {}).get("content", [])
            if content:
                print(
                    f"Stats after save response: {content[0].get('text', '')}")

    finally:
        try:
            proc.send_signal(subprocess.signal.SIGINT)
        except Exception:
            pass
        proc.wait(timeout=5)


if __name__ == "__main__":
    main()
