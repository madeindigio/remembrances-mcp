#!/usr/bin/env python3
"""
Simplified MCP SSE client for debugging the session issue.
This version tries to keep the SSE connection truly open.
"""

import argparse
import json
import subprocess
import time
import sys
import os
import requests


def main():
    p = argparse.ArgumentParser()
    p.add_argument("--binary", required=True)
    p.add_argument("--addr", default="127.0.0.1:4001")
    args = p.parse_args()

    bin_path = os.path.abspath(args.binary)
    if not os.path.exists(bin_path):
        print("Binary not found:", bin_path, file=sys.stderr)
        sys.exit(2)

    # Start the server
    env = os.environ.copy()
    env["GOMEM_OPENAI_KEY"] = "testkey"
    env["GOMEM_DB_PATH"] = os.path.abspath("./remembrances.db")
    env["GOMEM_SSE_ADDR"] = ":4001"
    proc = subprocess.Popen(
        [bin_path, "--sse"], env=env, stdout=subprocess.PIPE, stderr=subprocess.STDOUT)
    time.sleep(3.0)  # Give server time to start

    try:
        sse_url = f"http://{args.addr}/sse"
        print("Connecting to SSE url:", sse_url)

        # Open SSE connection with a longer timeout and keep-alive
        resp = requests.get(sse_url, stream=True, timeout=30)
        print(f"SSE connection status: {resp.status_code}")

        # Read the endpoint event
        endpoint = None
        for line in resp.iter_lines(decode_unicode=True):
            if not line:
                continue
            print(f"SSE line: {line}")
            if line.startswith("event: endpoint"):
                continue
            elif line.startswith("data: "):
                endpoint = line[6:]  # Remove "data: " prefix
                break

        if not endpoint:
            print("Failed to get endpoint from SSE stream")
            return

        print(f"Got endpoint: {endpoint}")

        # Build message URL
        if endpoint.startswith("/"):
            message_url = f"http://{args.addr}{endpoint}"
        else:
            message_url = endpoint

        print(f"Message URL: {message_url}")

        # Try a single initialize request
        init_req = {
            "jsonrpc": "2.0",
            "id": 1,
            "method": "initialize",
            "params": {
                "clientInfo": {"name": "simple-mcp-sse", "version": "0.1"},
                "protocolVersion": "2025-03-26"
            }
        }

        print("Sending initialize request...")
        headers = {"Content-Type": "application/json"}
        post_resp = requests.post(
            message_url, json=init_req, headers=headers, timeout=10)
        print(f"POST response: {post_resp.status_code} {post_resp.reason}")
        print(f"POST body: {post_resp.text}")

        # Check if we get any more SSE events
        print("Checking for SSE response events...")
        start_time = time.time()
        while time.time() - start_time < 5:
            try:
                line = next(resp.iter_lines(decode_unicode=True))
                if line:
                    print(f"SSE response: {line}")
            except StopIteration:
                break
            except Exception as e:
                print(f"SSE read error: {e}")
                break

    finally:
        try:
            resp.close()
        except:
            pass
        proc.send_signal(subprocess.signal.SIGINT)
        proc.wait(timeout=5)


if __name__ == "__main__":
    main()
