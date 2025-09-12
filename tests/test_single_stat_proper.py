#!/usr/bin/env python3

import json
import subprocess
import sys
import time


def send_mcp_request(process, method, params=None):
    """Send an MCP request via stdin and read the response"""
    if params is None:
        params = {}

    request = {
        "jsonrpc": "2.0",
        "id": 1,
        "method": method,
        "params": params
    }

    # Send the request
    request_json = json.dumps(request) + "\n"
    process.stdin.write(request_json)
    process.stdin.flush()

    # Read response
    try:
        response_line = process.stdout.readline()
        response = json.loads(response_line.strip())
        if "result" in response:
            return response["result"]
        elif "error" in response:
            print(f"MCP Error: {response['error']}")
            return None
    except json.JSONDecodeError as e:
        print(f"JSON decode error: {e}, line: {response_line}")
        return None
    except Exception as e:
        print(f"Error reading response: {e}")
        return None


def main():
    print("Testing single fact save and stats check...")

    # Start the MCP server process
    process = subprocess.Popen(
        ["./dist/remembrances-mcp"],
        stdin=subprocess.PIPE,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
        cwd="/www/MCP/remembrances-mcp"
    )

    try:
        # Initialize the MCP server
        print("Initializing MCP server...")
        init_result = send_mcp_request(process, "initialize", {
            "protocolVersion": "2024-11-05",
            "capabilities": {},
            "clientInfo": {
                "name": "test-client",
                "version": "1.0.0"
            }
        })
        print(f"Init result: {init_result}")

        if not init_result:
            print("Failed to initialize MCP server")
            return False

        # Test 1: Save a fact
        print("\n1. Saving a fact...")
        save_result = send_mcp_request(process, "tools/call", {
            "name": "remembrance_save_fact",
            "arguments": {
                "user_id": "test_single_stat",
                "key": "test_key",
                "value": "test_value"
            }
        })
        print(f"Save result: {save_result}")

        if not save_result:
            print("Failed to save fact")
            return False

        # Test 2: Check stats
        print("\n2. Checking stats...")
        stats_result = send_mcp_request(process, "tools/call", {
            "name": "remembrance_get_stats",
            "arguments": {
                "user_id": "test_single_stat"
            }
        })
        print(f"Stats result: {stats_result}")

        # Parse the stats from content
        if stats_result and "content" in stats_result:
            for content in stats_result["content"]:
                if content.get("type") == "text":
                    try:
                        stats_data = json.loads(content["text"])
                        print(f"Parsed stats: {stats_data}")

                        # Check if key_value_count is 1
                        if stats_data.get("key_value_count") == 1:
                            print("✅ SUCCESS: key_value_count is correctly 1!")
                            return True
                        else:
                            print(
                                f"❌ FAILED: key_value_count is {stats_data.get('key_value_count')}, expected 1")
                            return False
                    except json.JSONDecodeError:
                        print(f"Failed to parse stats: {content['text']}")

        print("❌ FAILED: Could not get or parse stats")
        return False

    finally:
        # Clean up
        process.terminate()
        try:
            stderr_output = process.stderr.read()
            if stderr_output:
                print("=== STDERR (Logs) ===", file=sys.stderr)
                print(stderr_output, file=sys.stderr)
                print("=== END STDERR ===", file=sys.stderr)
        except:
            pass


if __name__ == "__main__":
    success = main()
    sys.exit(0 if success else 1)
