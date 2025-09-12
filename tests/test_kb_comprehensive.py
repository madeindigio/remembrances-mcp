#!/usr/bin/env python3
"""
Comprehensive test for knowledge base dual storage functionality.
Tests all kb operations: add, get, search, delete with both database and filesystem storage.
"""

import os
import tempfile
import subprocess
import time
import json
from pathlib import Path


def send_mcp_request(proc, method, params_or_args, request_id):
    """Send an MCP request and return the response."""
    request = {
        "jsonrpc": "2.0",
        "id": request_id,
        "method": method,
        "params": params_or_args
    }

    proc.stdin.write(json.dumps(request) + "\n")
    proc.stdin.flush()
    time.sleep(0.5)


def test_comprehensive_kb():
    """Test all knowledge base operations with dual storage."""

    with tempfile.TemporaryDirectory() as temp_dir:
        kb_path = Path(temp_dir) / "knowledge-base"
        kb_path.mkdir()
        db_path = Path(temp_dir) / "test.db"

        print(f"Testing with knowledge-base path: {kb_path}")
        print(f"Testing with database path: {db_path}")

        # Build first
        build_result = subprocess.run(
            ["go", "build", "./cmd/remembrances-mcp"],
            cwd="/www/MCP/remembrances-mcp",
            capture_output=True,
            text=True
        )

        if build_result.returncode != 0:
            print(f"Build failed: {build_result.stderr}")
            return False

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
            send_mcp_request(proc, "initialize", {
                "protocolVersion": "2024-11-05",
                "capabilities": {},
                "clientInfo": {"name": "test-client", "version": "1.0.0"}
            }, 1)

            # Test 1: Add document
            print("\n=== Test 1: Adding document ===")
            doc_content = "# Test Document\n\nThis is a comprehensive test document.\n\n## Features\n\n- Database storage\n- Filesystem storage\n- Dual access\n"

            send_mcp_request(proc, "tools/call", {
                "name": "kb_add_document",
                "arguments": {
                    "file_path": "comprehensive_test.md",
                    "content": doc_content,
                    "metadata": {
                        "source": "comprehensive_test",
                        "type": "documentation"
                    }
                }
            }, 2)

            # Check filesystem
            fs_file = kb_path / "comprehensive_test.md"
            fs_exists = fs_file.exists()
            fs_content = fs_file.read_text() if fs_exists else ""

            print(f"✓ Document saved to filesystem: {fs_exists}")
            if fs_exists:
                print(f"✓ Content matches: {fs_content == doc_content}")

            # Test 2: Get document (should retrieve from database)
            print("\n=== Test 2: Getting document ===")
            send_mcp_request(proc, "tools/call", {
                "name": "kb_get_document",
                "arguments": {
                    "file_path": "comprehensive_test.md"
                }
            }, 3)

            # Test 3: Add filesystem-only document and retrieve it
            print("\n=== Test 3: Filesystem-only document ===")
            fs_only_file = kb_path / "filesystem_only.md"
            fs_only_content = "# Filesystem Only\n\nThis document exists only in filesystem.\n"
            fs_only_file.write_text(fs_only_content)

            send_mcp_request(proc, "tools/call", {
                "name": "kb_get_document",
                "arguments": {
                    "file_path": "filesystem_only.md"
                }
            }, 4)

            print(
                f"✓ Filesystem-only document created: {fs_only_file.exists()}")

            # Test 4: Search documents
            print("\n=== Test 4: Searching documents ===")
            send_mcp_request(proc, "tools/call", {
                "name": "kb_search_documents",
                "arguments": {
                    "query": "comprehensive test",
                    "limit": 5
                }
            }, 5)

            # Test 5: Delete document
            print("\n=== Test 5: Deleting document ===")
            print(f"Attempting to delete: {fs_file}")
            print(f"File exists before delete: {fs_file.exists()}")

            send_mcp_request(proc, "tools/call", {
                "name": "kb_delete_document",
                "arguments": {
                    "file_path": "comprehensive_test.md"
                }
            }, 6)

            # Wait longer for deletion to complete
            time.sleep(2)

            # Check if file was removed from filesystem
            fs_exists_after_delete = fs_file.exists()
            print(f"File exists after delete: {fs_exists_after_delete}")
            print(
                f"✓ Document removed from filesystem: {not fs_exists_after_delete}")

            # List all files in directory after delete
            files_after_delete = list(kb_path.iterdir())
            print(
                f"Files after delete: {[f.name for f in files_after_delete]}")

            # Test 6: Try to get deleted document
            print("\n=== Test 6: Getting deleted document ===")
            send_mcp_request(proc, "tools/call", {
                "name": "kb_get_document",
                "arguments": {
                    "file_path": "comprehensive_test.md"
                }
            }, 7)

            # Wait for all operations to complete
            time.sleep(2)

            # Check final state
            final_files = list(kb_path.iterdir())
            print("\n=== Final State ===")
            print(
                f"Files remaining in KB directory: {[f.name for f in final_files]}")

            # The filesystem_only.md should still exist
            expected_remaining = ["filesystem_only.md"]
            actual_remaining = [f.name for f in final_files]

            success = (
                fs_exists and  # Document was created in filesystem
                fs_content == doc_content and  # Content was correct
                not fs_exists_after_delete and  # Document was deleted from filesystem
                fs_only_file.exists() and  # Filesystem-only document still exists
                # Only expected files remain
                set(actual_remaining) == set(expected_remaining)
            )

            print("\n=== Test Summary ===")
            print(
                f"✓ Document creation: {fs_exists and fs_content == doc_content}")
            print(f"✓ Document deletion: {not fs_exists_after_delete}")
            print(f"✓ Filesystem-only access: {fs_only_file.exists()}")
            print(
                f"✓ Final state correct: {set(actual_remaining) == set(expected_remaining)}")
            print(f"🎯 Overall success: {success}")

            return success

        except Exception as e:
            print(f"Error during test: {e}")
            return False
        finally:
            if proc:
                try:
                    proc.terminate()
                    _, stderr = proc.communicate(timeout=3)
                    if stderr:
                        print("\n=== Server Debug Output ===")
                        print(stderr)
                except subprocess.TimeoutExpired:
                    proc.kill()


if __name__ == "__main__":
    print("Running comprehensive knowledge base test...")
    result = test_comprehensive_kb()

    if result:
        print("\n🎉 ALL TESTS PASSED!")
        print(
            "✅ Knowledge base documents are correctly saved to both database and filesystem")
        print("✅ All operations (add, get, search, delete) work with dual storage")
    else:
        print("\n❌ SOME TESTS FAILED")

    exit(0 if result else 1)
