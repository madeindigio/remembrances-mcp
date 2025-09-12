#!/usr/bin/env python3
"""
Test script to verify knowledge base document storage behavior.
This script tests if documents are saved to both database and filesystem.
"""

import os
import sys
import json
import tempfile
import subprocess
import time
from pathlib import Path


def test_knowledge_base_storage():
    """Test if kb_add_document saves to both database and filesystem."""

    # Create temporary directories
    with tempfile.TemporaryDirectory() as temp_dir:
        kb_path = Path(temp_dir) / "knowledge-base"
        kb_path.mkdir()

        db_path = Path(temp_dir) / "test.db"

        print(f"Testing with knowledge-base path: {kb_path}")
        print(f"Testing with database path: {db_path}")

        # Start the MCP server with knowledge-base parameter
        cmd = [
            "go", "run", "./cmd/remembrances-mcp/main.go",
            "--knowledge-base", str(kb_path),
            "--db-path", str(db_path)
        ]

        print(f"Starting server with command: {' '.join(cmd)}")

        try:
            # Start the server process
            proc = subprocess.Popen(
                cmd,
                stdin=subprocess.PIPE,
                stdout=subprocess.PIPE,
                stderr=subprocess.PIPE,
                text=True,
                cwd="/www/MCP/remembrances-mcp"
            )

            # Wait a bit for server to start
            time.sleep(2)

            # Send initialization request
            init_request = {
                "jsonrpc": "2.0",
                "id": 1,
                "method": "initialize",
                "params": {
                    "protocolVersion": "2024-11-05",
                    "capabilities": {},
                    "clientInfo": {
                        "name": "test-client",
                        "version": "1.0.0"
                    }
                }
            }

            proc.stdin.write(json.dumps(init_request) + "\n")
            proc.stdin.flush()

            # Wait for initialization response
            time.sleep(1)

            # Test adding a document
            add_doc_request = {
                "jsonrpc": "2.0",
                "id": 2,
                "method": "tools/call",
                "params": {
                    "name": "kb_add_document",
                    "arguments": {
                        "file_path": "test_document.md",
                        "content": "# Test Document\n\nThis is a test document for knowledge base storage.\n\n## Content\n\nSome test content here.",
                        "metadata": {
                            "source": "test",
                            "category": "documentation"
                        }
                    }
                }
            }

            print("Sending kb_add_document request...")
            proc.stdin.write(json.dumps(add_doc_request) + "\n")
            proc.stdin.flush()

            # Wait for response
            time.sleep(2)

            # Check if file was created in knowledge-base directory
            expected_file = kb_path / "test_document.md"
            file_exists = expected_file.exists()

            print("\nResults:")
            print(f"Expected file path: {expected_file}")
            print(f"File exists in knowledge-base directory: {file_exists}")

            if file_exists:
                print("File content:")
                with open(expected_file, 'r') as f:
                    print(f.read())
            else:
                print("No file found in knowledge-base directory")

            # List all files in knowledge-base directory
            kb_files = list(kb_path.iterdir())
            print(
                f"Files in knowledge-base directory: {[f.name for f in kb_files]}")

            # Check database directory was created
            print(f"Database path exists: {db_path.exists()}")

            return file_exists

        except Exception as e:
            print(f"Error during test: {e}")
            return False
        finally:
            # Clean up
            if proc:
                proc.terminate()
                proc.wait()


if __name__ == "__main__":
    print("Testing knowledge base document storage...")
    result = test_knowledge_base_storage()

    if result:
        print("\n✅ SUCCESS: Documents are saved to filesystem")
    else:
        print("\n❌ ISSUE: Documents are NOT saved to filesystem")
        print("Only database storage is currently implemented")

    sys.exit(0 if result else 1)
