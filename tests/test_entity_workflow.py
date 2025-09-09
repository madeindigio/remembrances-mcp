#!/usr/bin/env python3
"""
Test script for full entity creation and retrieval workflow
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
                continue
            if text:
                with self.lock:
                    self.lines.append(text)

    def stop(self):
        self.running = False

    def get_next_json(self):
        while True:
            with self.lock:
                for i, line in enumerate(self.lines):
                    if line.startswith("{"):
                        self.lines.pop(i)
                        return json.loads(line)
                    else:
                        print(f"Non-JSON line: {line}")
                        self.lines.pop(i)
            time.sleep(0.01)


def wait_for_response(reader, timeout=5):
    start = time.time()
    while time.time() - start < timeout:
        try:
            return reader.get_next_json()
        except json.JSONDecodeError:
            continue
        except Exception as e:
            print(f"Error: {e}")
            continue
    raise TimeoutError("No valid JSON response within timeout")


def main():
    bin_path = "./dist/remembrances-mcp"

    # Start the process
    proc = subprocess.Popen([bin_path, "--ollama-model", "llama3.2"],
                            stdin=subprocess.PIPE,
                            stdout=subprocess.PIPE,
                            stderr=subprocess.STDOUT)
    reader = ReaderThread(proc.stdout)
    reader.start()

    try:
        # Initialize
        init_req = {
            "jsonrpc": "2.0",
            "id": 1,
            "method": "initialize",
            "params": {
                "clientInfo": {"name": "test-entity-workflow", "version": "0.1"},
                "protocolVersion": "2025-03-26"
            }
        }
        send_json(proc.stdin, init_req)
        print("Sent initialize")
        resp = wait_for_response(reader, timeout=10)
        print("initialize response: OK")

        # Test create_entity
        call_id = 2
        create_entity_params = {
            "name": "remembrance_create_entity",
            "arguments": {
                "entity_type": "person",
                "name": "Bob",
                "properties": {"email": "bob@example.com", "role": "manager"}
            }
        }
        call_req = {"jsonrpc": "2.0", "id": call_id,
                    "method": "tools/call", "params": create_entity_params}
        send_json(proc.stdin, call_req)
        print("Sent create_entity request")
        resp = wait_for_response(reader, timeout=10)
        print("create_entity response:", json.dumps(resp, indent=2))

        # Extract entity ID from response if successful
        entity_id = None
        if "result" in resp and "content" in resp["result"]:
            for content in resp["result"]["content"]:
                if content["type"] == "text" and "entity" in content["text"]:
                    # Try to extract entity ID - it's often included in the response
                    print("Entity created successfully!")

        # Test get_stats to see our created entities
        call_id += 1
        stats_params = {
            "name": "remembrance_get_stats",
            "arguments": {"user_id": "test_user"}
        }
        call_req = {"jsonrpc": "2.0", "id": call_id,
                    "method": "tools/call", "params": stats_params}
        send_json(proc.stdin, call_req)
        print("Sent get_stats request")
        resp = wait_for_response(reader, timeout=10)
        print("get_stats response:", json.dumps(resp, indent=2))

    except Exception as e:
        print(f"Error during test: {e}")
        import traceback
        traceback.print_exc()
    finally:
        try:
            proc.send_signal(subprocess.signal.SIGINT)
        except Exception:
            pass
        proc.wait(timeout=5)


if __name__ == "__main__":
    main()
