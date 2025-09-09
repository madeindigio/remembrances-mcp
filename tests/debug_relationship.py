#!/usr/bin/env python3
"""
Debug script to test entity creation and relationship workflow
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
                "clientInfo": {"name": "debug-entities", "version": "0.1"},
                "protocolVersion": "2025-03-26"
            }
        }
        send_json(proc.stdin, init_req)
        print("Sent initialize")
        resp = wait_for_response(reader, timeout=10)
        print("initialize response: OK")

        # Create entity 1
        call_id = 2
        create_entity1_params = {
            "name": "remembrance_create_entity",
            "arguments": {
                "entity_type": "person",
                "name": "Alice Debug",
                "properties": {"role": "developer"}
            }
        }
        call_req = {"jsonrpc": "2.0", "id": call_id,
                    "method": "tools/call", "params": create_entity1_params}
        send_json(proc.stdin, call_req)
        print("Sent create_entity request for Alice")
        resp = wait_for_response(reader, timeout=10)
        print("create_entity Alice response:", json.dumps(resp, indent=2))

        # Create entity 2
        call_id += 1
        create_entity2_params = {
            "name": "remembrance_create_entity",
            "arguments": {
                "entity_type": "person",
                "name": "Bob Debug",
                "properties": {"role": "manager"}
            }
        }
        call_req = {"jsonrpc": "2.0", "id": call_id,
                    "method": "tools/call", "params": create_entity2_params}
        send_json(proc.stdin, call_req)
        print("Sent create_entity request for Bob")
        resp = wait_for_response(reader, timeout=10)
        print("create_entity Bob response:", json.dumps(resp, indent=2))

        # Check stats
        call_id += 1
        stats_params = {
            "name": "remembrance_get_stats",
            "arguments": {"user_id": "debug_user"}
        }
        call_req = {"jsonrpc": "2.0", "id": call_id,
                    "method": "tools/call", "params": stats_params}
        send_json(proc.stdin, call_req)
        print("Sent get_stats request")
        resp = wait_for_response(reader, timeout=10)
        print("get_stats response:", json.dumps(resp, indent=2))

        # Try to create relationship
        call_id += 1
        create_rel_params = {
            "name": "remembrance_create_relationship",
            "arguments": {
                "from_entity": "Alice Debug",
                "to_entity": "Bob Debug",
                "relationship_type": "works_for",
                "properties": {"since": "2024-01-01"}
            }
        }
        call_req = {"jsonrpc": "2.0", "id": call_id,
                    "method": "tools/call", "params": create_rel_params}
        send_json(proc.stdin, call_req)
        print("Sent create_relationship request")
        resp = wait_for_response(reader, timeout=10)
        print("create_relationship response:", json.dumps(resp, indent=2))

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
