#!/usr/bin/env python3
"""
Test script for relationship creation fix
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
    binary_path = "./dist/remembrances-mcp"

    if not os.path.exists(binary_path):
        print("Binary not found:", binary_path, file=sys.stderr)
        sys.exit(2)

    proc = subprocess.Popen([binary_path], stdin=subprocess.PIPE,
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
                "clientInfo": {"name": "relationship-test", "version": "0.1"},
                "protocolVersion": "2025-03-26"
            }
        }
        send_json(proc.stdin, init_req)
        print("Sent initialize")
        resp = wait_for_response(reader, timeout=10)
        print("initialize response:", json.dumps(resp, indent=2))

        call_id = 2

        # Create John Developer
        print("\n1. Creating John Developer...")
        create_john_params = {
            "name": "remembrance_create_entity",
            "arguments": {
                "entity_type": "person",
                "name": "John Developer"
            }
        }
        create_john_req = {
            "jsonrpc": "2.0",
            "id": call_id,
            "method": "tools/call",
            "params": create_john_params
        }
        send_json(proc.stdin, create_john_req)
        resp = wait_for_response(reader)
        print("Create John Developer response:", json.dumps(resp, indent=2))
        call_id += 1

        # Create Sarah Manager
        print("\n2. Creating Sarah Manager...")
        create_sarah_params = {
            "name": "remembrance_create_entity",
            "arguments": {
                "entity_type": "person",
                "name": "Sarah Manager"
            }
        }
        create_sarah_req = {
            "jsonrpc": "2.0",
            "id": call_id,
            "method": "tools/call",
            "params": create_sarah_params
        }
        send_json(proc.stdin, create_sarah_req)
        resp = wait_for_response(reader)
        print("Create Sarah Manager response:", json.dumps(resp, indent=2))
        call_id += 1

        # Create relationship
        print("\n3. Creating relationship...")
        create_rel_params = {
            "name": "remembrance_create_relationship",
            "arguments": {
                "from_entity": "John Developer",
                "to_entity": "Sarah Manager",
                "relationship_type": "reports_to"
            }
        }
        create_rel_req = {
            "jsonrpc": "2.0",
            "id": call_id,
            "method": "tools/call",
            "params": create_rel_params
        }
        send_json(proc.stdin, create_rel_req)
        resp = wait_for_response(reader)
        print("Create relationship response:", json.dumps(resp, indent=2))
        call_id += 1

        # Get stats to verify
        print("\n4. Getting stats...")
        stats_params = {
            "name": "remembrance_get_stats",
            "arguments": {
                "user_id": "test_user"
            }
        }
        stats_req = {
            "jsonrpc": "2.0",
            "id": call_id,
            "method": "tools/call",
            "params": stats_params
        }
        send_json(proc.stdin, stats_req)
        resp = wait_for_response(reader)
        print("Stats response:", json.dumps(resp, indent=2))

    finally:
        try:
            proc.send_signal(subprocess.signal.SIGINT)
        except Exception:
            pass
        proc.wait(timeout=5)


if __name__ == "__main__":
    main()
