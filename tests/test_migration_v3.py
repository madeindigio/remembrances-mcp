#!/usr/bin/env python3
"""
Test Migration v3 - Verify that user statistics persist correctly after the schema fix.

This test verifies that the migration v3 successfully fixes the user_stats field definitions
by removing the problematic VALUE constraints that made fields read-only.
"""

import json
import subprocess
import sys
import os
import time
import threading

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
                text = line.strip()
            if text == "":
                continue
            with self.lock:
                self.lines.append(text)

    def pop_lines(self):
        with self.lock:
            l = self.lines[:]
            self.lines.clear()
        return l


def wait_for_response(reader, timeout=15):
    deadline = time.time() + timeout
    while time.time() < deadline:
        lines = reader.pop_lines()
        if lines:
            # Return the first JSON parseable line
            for ln in lines:
                try:
                    return json.loads(ln)
                except Exception:
                    # keep searching
                    print("Non-JSON line:", ln, file=sys.stderr)
        time.sleep(0.1)
    raise TimeoutError("Timed out waiting for response")


class MigrationV3Test:
    def __init__(self):
        self.proc = None
        self.reader = None
        self.request_id = 1

    def setup(self):
        """Set up the test environment"""
        print("Setting up test environment...")

        # Find the binary
        bin_path = os.path.abspath("../dist/remembrances-mcp")
        if not os.path.exists(bin_path):
            print(f"✗ Binary not found: {bin_path}")
            return False

        # Start the MCP server with stdio transport
        try:
            self.proc = subprocess.Popen([bin_path], stdin=subprocess.PIPE,
                                         stdout=subprocess.PIPE, stderr=subprocess.STDOUT)
            self.reader = ReaderThread(self.proc.stdout)
            self.reader.start()

            # Initialize
            init_req = {
                "jsonrpc": "2.0",
                "id": self.request_id,
                "method": "initialize",
                "params": {
                    "clientInfo": {"name": "migration-v3-test", "version": "0.1"},
                    "protocolVersion": "2025-03-26"
                }
            }
            self.request_id += 1
            send_json(self.proc.stdin, init_req)
            resp = wait_for_response(self.reader, timeout=10)

            if resp.get("error"):
                print(f"✗ Initialize failed: {resp['error']}")
                return False

            print("✓ Connected to MCP server")
        except Exception as e:
            print(f"✗ Failed to connect to MCP server: {e}")
            return False

        return True

    def cleanup(self):
        """Clean up"""
        if self.proc:
            try:
                self.proc.send_signal(subprocess.signal.SIGINT)
            except Exception:
                pass
            self.proc.wait(timeout=5)

    def call_tool(self, tool_name: str, arguments: dict):
        """Call an MCP tool"""
        call_req = {
            "jsonrpc": "2.0",
            "id": self.request_id,
            "method": "tools/call",
            "params": {
                "name": tool_name,
                "arguments": arguments
            }
        }
        self.request_id += 1

        send_json(self.proc.stdin, call_req)
        resp = wait_for_response(self.reader)

        if resp.get("error"):
            raise RuntimeError(f"Tool call failed: {resp['error']}")

        return resp.get("result", {})

    def get_stats(self, user_id: str) -> dict:
        """Get user statistics"""
        try:
            result = self.call_tool("remembrance_get_stats", {
                "user_id": user_id
            })

            # Parse the JSON content from the response
            content = result.get("content", [])
            if content and len(content) > 0:
                text = content[0].get("text", "{}")

                # The text includes a header, we need to extract just the JSON part
                # Format: "Memory statistics for user 'test_user_stats':\n{...}"
                lines = text.split('\n', 1)
                if len(lines) > 1:
                    json_text = lines[1]
                else:
                    json_text = text

                return json.loads(json_text)
            return {}
        except Exception as e:
            print(f"Error getting stats: {e}")
            print(f"Raw result: {result}")
            return {}

    def test_migration_v3(self):
        """Test that migration v3 fixes the statistics persistence issue."""

        user_id = "test_migration_v3_user"

        try:
            # Get initial stats
            print("Getting initial stats...")
            initial_stats = self.get_stats(user_id)
            print(f"Initial stats: {initial_stats}")

            # Save a fact
            print("Saving a key-value fact...")
            result = self.call_tool("remembrance_save_fact", {
                "user_id": user_id,
                "key": "test_migration_key",
                "value": "test_migration_value"
            })
            print(f"Save fact result: {result}")

            # Add a vector
            print("Adding a vector memory...")
            result = self.call_tool("remembrance_add_vector", {
                "user_id": user_id,
                "content": "This is test content for migration v3 verification",
                "metadata": {"test": "migration_v3"}
            })
            print(f"Add vector result: {result}")

            # Create an entity
            print("Creating an entity...")
            result = self.call_tool("remembrance_create_entity", {
                "entity_type": "test_migration_entity",
                "name": "Migration Test Entity",
                "properties": {"test": "migration_v3"}
            })
            print(f"Create entity result: {result}")
            entity_id = result.get("entity_id") if isinstance(
                result, dict) else None

            # Create a relationship if we have an entity
            if entity_id:
                print("Creating a second entity and relationship...")
                result2 = self.call_tool("remembrance_create_entity", {
                    "entity_type": "test_migration_entity2",
                    "name": "Migration Test Entity 2",
                    "properties": {"test": "migration_v3"}
                })
                entity_id2 = result2.get("entity_id") if isinstance(
                    result2, dict) else None

                if entity_id2:
                    result = self.call_tool("remembrance_create_relationship", {
                        "from_entity": entity_id,
                        "to_entity": entity_id2,
                        "relationship_type": "test_migration_relationship",
                        "properties": {"test": "migration_v3"}
                    })
                    print(f"Create relationship result: {result}")

            # Add a document
            print("Adding a knowledge base document...")
            result = self.call_tool("kb_add_document", {
                "file_path": "/test_migration_v3/test_doc.txt",
                "content": "This is a test document for migration v3 verification",
                "metadata": {"test": "migration_v3"}
            })
            print(f"Add document result: {result}")

            # Now check the updated stats
            print("Getting updated stats after operations...")
            final_stats = self.get_stats(user_id)
            print(f"Updated stats: {final_stats}")

            # Verify that statistics are no longer all 0
            if isinstance(final_stats, dict):
                key_value_count = final_stats.get("key_value_count", 0)
                vector_count = final_stats.get("vector_count", 0)
                entity_count = final_stats.get("entity_count", 0)
                relationship_count = final_stats.get("relationship_count", 0)
                document_count = final_stats.get("document_count", 0)

                print("\nFinal verification:")
                print(f"  key_value_count: {key_value_count}")
                print(f"  vector_count: {vector_count}")
                print(f"  entity_count: {entity_count}")
                print(f"  relationship_count: {relationship_count}")
                print(f"  document_count: {document_count}")

                # Check if any statistics are non-zero (success!)
                total_items = key_value_count + vector_count + \
                    entity_count + relationship_count + document_count

                if total_items > 0:
                    print(
                        f"\n✅ SUCCESS: Migration v3 FIXED the statistics! Total items: {total_items}")
                    print(
                        "The VALUE constraints have been successfully removed from user_stats fields.")
                    return True
                else:
                    print("\n❌ FAILURE: Statistics are still all 0 after migration v3")
                    print(
                        "The migration may not have been applied correctly or there may be another issue.")
                    return False
            else:
                print(
                    f"\n❌ FAILURE: Unexpected stats result format: {final_stats}")
                return False

        except Exception as e:
            print(f"❌ Error during migration v3 test: {e}")
            return False

    def run_test(self):
        """Run the migration v3 test"""
        print("=== Migration v3 Test - User Statistics Persistence Fix ===")
        print("This test verifies that migration v3 fixes the VALUE constraint issue.")
        print()

        if not self.setup():
            return False

        try:
            return self.test_migration_v3()
        finally:
            self.cleanup()


if __name__ == "__main__":
    test = MigrationV3Test()
    success = test.run_test()

    if success:
        print("\n🎉 Migration v3 test PASSED!")
        print("User statistics are now persisting correctly.")
        sys.exit(0)
    else:
        print("\n💥 Migration v3 test FAILED!")
        print("There may be an issue with the migration or field definitions.")
        sys.exit(1)
