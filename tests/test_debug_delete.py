#!/usr/bin/env python3
"""
Debug test for delete operation.
"""

import tempfile
import subprocess
import time
import json
from pathlib import Path


def debug_delete_test():
    """Debug the delete operation."""

    with tempfile.TemporaryDirectory() as temp_dir:
        kb_path = Path(temp_dir) / "knowledge-base"
        kb_path.mkdir()
        db_path = Path(temp_dir) / "test.db"

        print(f"KB path: {kb_path}")

        # Start server
        cmd = [
            "./remembrances-mcp",
            "--knowledge-base", str(kb_path),
            "--db-path", str(db_path)
        ]

        try:
            proc = subprocess.Popen(
                cmd,
                stdin=subprocess.PIPE,
                stdout=subprocess.PIPE,
                stderr=subprocess.PIPE,
                text=True,
                cwd="/www/MCP/remembrances-mcp"
            )

            # Initialize
            init_msg = {
                "jsonrpc": "2.0",
                "id": 1,
                "method": "initialize",
                "params": {
                    "protocolVersion": "2024-11-05",
                    "capabilities": {},
                    "clientInfo": {"name": "debug-test", "version": "1.0.0"}
                }
            }
            proc.stdin.write(json.dumps(init_msg) + "\n")
            proc.stdin.flush()
            time.sleep(1)

            # Add document
            add_msg = {
                "jsonrpc": "2.0",
                "id": 2,
                "method": "tools/call",
                "params": {
                    "name": "kb_add_document",
                    "arguments": {
                        "file_path": "debug_test.md",
                        "content": "# Debug Test\n\nThis is for debugging.",
                        "metadata": {"source": "debug"}
                    }
                }
            }
            proc.stdin.write(json.dumps(add_msg) + "\n")
            proc.stdin.flush()
            time.sleep(1)

            # Check file exists
            debug_file = kb_path / "debug_test.md"
            print(f"File exists after add: {debug_file.exists()}")

            # Delete document
            delete_msg = {
                "jsonrpc": "2.0",
                "id": 3,
                "method": "tools/call",
                "params": {
                    "name": "kb_delete_document",
                    "arguments": {
                        "file_path": "debug_test.md"
                    }
                }
            }
            proc.stdin.write(json.dumps(delete_msg) + "\n")
            proc.stdin.flush()
            time.sleep(2)

            # Check if file was deleted
            print(f"File exists after delete: {debug_file.exists()}")

            # Get server output
            proc.terminate()
            _, stderr = proc.communicate(timeout=3)

            print("\n=== Server Output ===")
            if stderr:
                print("STDERR:")
                print(stderr)

            return not debug_file.exists()

        except Exception as e:
            print(f"Error: {e}")
            return False
        finally:
            if proc and proc.poll() is None:
                proc.kill()


if __name__ == "__main__":
    success = debug_delete_test()
    print(f"\nDelete test {'PASSED' if success else 'FAILED'}")
