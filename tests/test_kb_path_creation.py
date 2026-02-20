#!/usr/bin/env python3
"""Regression test for knowledge-base path creation and KB tool gating.

Expected behavior:
- If --knowledge-base points to a missing directory, the server should create it.
- If the directory cannot be created, the server should still start, but KB tools
  (kb_*) must be disabled.
"""

import os
import sys
import tempfile

# Add the e2e module to path
sys.path.insert(0, os.path.dirname(__file__))

from e2e.client import MCPClient


KB_TOOL_NAMES = {
    "kb_add_document",
    "kb_search_documents",
    "kb_get_document",
    "kb_delete_document",
}


def _tool_names(resp: dict) -> set[str]:
    tools = resp.get("result", {}).get("tools", [])
    names: set[str] = set()
    for t in tools:
        if isinstance(t, dict) and isinstance(t.get("name"), str):
            names.add(t["name"])
    return names


def main() -> int:
    server_bin = os.path.join(os.path.dirname(__file__), "..", "build", "remembrances-mcp")
    server_bin = os.path.abspath(server_bin)

    if not os.path.exists(server_bin):
        print(f"ERROR: Server binary not found at {server_bin}")
        return 1

    # Case 1: missing path -> should be created and KB tools available.
    tmp_root = tempfile.mkdtemp(prefix="kb_test_")
    kb_path = os.path.join(tmp_root, "nested", "kb")
    client = MCPClient([server_bin, "--knowledge-base", kb_path])
    try:
        if not client.start_server():
            print("ERROR: server failed to start (mkdir success case)")
            return 1

        if not os.path.isdir(kb_path):
            print(f"ERROR: knowledge base dir was not created: {kb_path}")
            return 1

        resp = client.call_method("tools/list", {})
        if "error" in resp:
            print(f"ERROR: tools/list failed: {resp['error']}")
            return 1

        names = _tool_names(resp)
        if not (KB_TOOL_NAMES & names):
            print("ERROR: expected KB tools to be present when KB dir is usable")
            print(f"Tools: {sorted(names)}")
            return 1
    finally:
        client.close()

    # Case 2: uncreatable path -> server starts, KB tools absent.
    bad_kb_path = "/proc/remembrances-kb-test/nested"
    client2 = MCPClient([server_bin, "--knowledge-base", bad_kb_path])
    try:
        if not client2.start_server():
            print("ERROR: server failed to start (mkdir failure case)")
            return 1

        resp = client2.call_method("tools/list", {})
        if "error" in resp:
            print(f"ERROR: tools/list failed: {resp['error']}")
            return 1

        names = _tool_names(resp)
        if KB_TOOL_NAMES & names:
            print("ERROR: expected KB tools to be disabled when KB dir is not usable")
            print(f"KB tools present: {sorted(KB_TOOL_NAMES & names)}")
            return 1

        print("✅ knowledge base path creation/disable behavior is correct")
        return 0
    finally:
        client2.close()


if __name__ == "__main__":
    raise SystemExit(main())
