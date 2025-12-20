#!/usr/bin/env python3
"""Regression test for MCP tool schemas.

Some clients (e.g. GitHub Copilot CLI via OpenAI tool schema validation) reject
object schemas that omit the `properties` key, even when the object has no
fields. Tools like `code_list_projects` have no inputs and must still advertise
`inputSchema: {"type":"object","properties":{}}`.

This test starts the server (stdio transport), calls `tools/list`, and asserts
that `code_list_projects` includes an explicit `properties` object.
"""

import os
import sys

# Add the e2e module to path
sys.path.insert(0, os.path.dirname(__file__))

from e2e.client import MCPClient


def main() -> int:
    server_bin = os.path.join(os.path.dirname(__file__), "..", "build", "remembrances-mcp")
    server_bin = os.path.abspath(server_bin)

    if not os.path.exists(server_bin):
        print(f"ERROR: Server binary not found at {server_bin}")
        return 1

    client = MCPClient([server_bin])
    try:
        if not client.start_server():
            print("ERROR: server failed to start")
            return 1

        resp = client.call_method("tools/list", {})
        if "error" in resp:
            print(f"ERROR: tools/list failed: {resp['error']}")
            return 1

        if resp.get("result") is None:
            print(f"ERROR: tools/list missing result: {resp}")
            return 1

        tools = resp["result"].get("tools")
        if not isinstance(tools, list):
            print(f"ERROR: tools/list result.tools is not a list: {resp['result']}")
            return 1

        target = None
        for t in tools:
            if isinstance(t, dict) and t.get("name") == "code_list_projects":
                target = t
                break

        if target is None:
            names = [t.get("name") for t in tools if isinstance(t, dict)]
            print("ERROR: code_list_projects not found in tool list")
            print(f"Available tools: {names}")
            return 1

        input_schema = target.get("inputSchema")
        if not isinstance(input_schema, dict):
            print(f"ERROR: inputSchema is not an object: {input_schema}")
            return 1

        if input_schema.get("type") != "object":
            print(f"ERROR: expected inputSchema.type == 'object', got: {input_schema.get('type')}")
            return 1

        if "properties" not in input_schema:
            print("ERROR: inputSchema is missing 'properties' key")
            print(f"inputSchema: {input_schema}")
            return 1

        if not isinstance(input_schema["properties"], dict):
            print("ERROR: inputSchema.properties is not an object")
            print(f"inputSchema: {input_schema}")
            return 1

        print("✅ tool schema includes properties for code_list_projects")
        return 0
    finally:
        client.close()


if __name__ == "__main__":
    raise SystemExit(main())
