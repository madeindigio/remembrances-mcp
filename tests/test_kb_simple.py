#!/usr/bin/env python3
"""
Simplified test to debug the knowledge base storage issue.
"""

import os
import tempfile
import subprocess
import time
import json
from pathlib import Path


def test_simple():
    """Simple test with better debugging."""

    # Create temporary directories
    with tempfile.TemporaryDirectory() as temp_dir:
        kb_path = Path(temp_dir) / "knowledge-base"
        kb_path.mkdir()

        db_path = Path(temp_dir) / "test.db"

        print(f"KB path: {kb_path}")
        print(f"DB path: {db_path}")

        # Create a simple markdown file to test
        test_content = "# Test Document\n\nThis is a test.\n"

        # Test direct file creation first
        test_file = kb_path / "direct_test.md"
        test_file.write_text(test_content)

        print(f"Direct file creation works: {test_file.exists()}")

        # Build the project first
        print("Building project...")
        build_result = subprocess.run(
            ["go", "build", "./cmd/remembrances-mcp"],
            cwd="/www/MCP/remembrances-mcp",
            capture_output=True,
            text=True
        )

        if build_result.returncode != 0:
            print(f"Build failed: {build_result.stderr}")
            return False

        print("Build successful")

        # Run server with test configuration
        cmd = [
            "./remembrances-mcp",
            "--knowledge-base", str(kb_path),
            "--db-path", str(db_path)
        ]

        print(f"Running: {' '.join(cmd)}")

        try:
            proc = subprocess.Popen(
                cmd,
                stdin=subprocess.PIPE,
                stdout=subprocess.PIPE,
                stderr=subprocess.PIPE,
                text=True,
                cwd="/www/MCP/remembrances-mcp"
            )

            # Send messages
            messages = [
                {
                    "jsonrpc": "2.0",
                    "id": 1,
                    "method": "initialize",
                    "params": {
                        "protocolVersion": "2024-11-05",
                        "capabilities": {},
                        "clientInfo": {"name": "test-client", "version": "1.0.0"}
                    }
                },
                {
                    "jsonrpc": "2.0",
                    "id": 2,
                    "method": "tools/call",
                    "params": {
                        "name": "kb_add_document",
                        "arguments": {
                            "file_path": "test_via_mcp.md",
                            "content": test_content,
                            "metadata": {"source": "test"}
                        }
                    }
                }
            ]

            print("Sending messages...")
            for msg in messages:
                proc.stdin.write(json.dumps(msg) + "\n")
                proc.stdin.flush()
                time.sleep(0.5)

            # Give it time to process
            time.sleep(2)

            # Check results
            mcp_file = kb_path / "test_via_mcp.md"

            print("\nResults:")
            print(f"Direct test file exists: {test_file.exists()}")
            print(f"MCP test file exists: {mcp_file.exists()}")
            print(
                f"Files in KB directory: {[f.name for f in kb_path.iterdir()]}")

            # Get any stdout/stderr
            try:
                stdout, stderr = proc.communicate(timeout=1)
                if stdout:
                    print(f"Server stdout: {stdout}")
                if stderr:
                    print(f"Server stderr: {stderr}")
            except subprocess.TimeoutExpired:
                proc.kill()
                stdout, stderr = proc.communicate()
                print("Server timed out")
                if stdout:
                    print(f"Server stdout: {stdout}")
                if stderr:
                    print(f"Server stderr: {stderr}")

            return mcp_file.exists()

        except Exception as e:
            print(f"Error: {e}")
            return False
        finally:
            if proc:
                proc.terminate()


if __name__ == "__main__":
    result = test_simple()
    print(f"\nTest result: {'PASS' if result else 'FAIL'}")
