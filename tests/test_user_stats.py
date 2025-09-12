#!/usr/bin/env python3
"""
Test user statistics functionality for the remembrances-mcp server.
This test verifies that the user_stats table is properly updated when
facts, vectors, entities, relationships, and documents are created/deleted.
"""

import json
import sys
import os
import subprocess
import time
import threading

# Add the current directory to path for importing helper functions
sys.path.append(os.path.dirname(__file__))


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


class UserStatsTest:
    def __init__(self):
        self.proc = None
        self.reader = None
        self.request_id = 1
        self.test_user = "test_user_stats"

    def setup(self):
        """Set up the test environment"""
        print("Setting up test environment...")

        # Find the binary
        bin_path = os.path.abspath("./dist/remembrances-mcp")
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
                    "clientInfo": {"name": "mcp-stats-test", "version": "0.1"},
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
        """Clean up test data"""
        print("Cleaning up test data...")
        try:
            # Delete test facts
            for i in range(3):
                self.call_tool("remembrance_delete_fact", {
                    "user_id": self.test_user,
                    "key": f"test_key_{i}"
                })

        except Exception as e:
            print(f"Warning: cleanup error: {e}")

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
            raise Exception(f"Tool call failed: {resp['error']}")

        return resp.get("result", {})

    def get_stats(self) -> dict:
        """Get user statistics"""
        try:
            result = self.call_tool("remembrance_get_stats", {
                "user_id": self.test_user
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

    def test_initial_stats(self):
        """Test that initial stats are zero or empty"""
        print("\n1. Testing initial stats...")

        stats = self.get_stats()
        print(f"Initial stats: {stats}")

        # Initial stats should be zero for new users
        expected_keys = ["key_value_count", "vector_count",
                         "entity_count", "relationship_count", "document_count"]
        for key in expected_keys:
            if key not in stats:
                print(f"✗ Missing stat key: {key}")
                return False
            # Allow for existing global data (entities, relationships, documents)
            if key in ["key_value_count", "vector_count"] and stats[key] != 0:
                print(f"✗ Expected {key} to be 0, got {stats[key]}")
                return False

        print("✓ Initial stats look correct")
        return True

    def test_fact_operations(self):
        """Test that fact operations update statistics correctly"""
        print("\n2. Testing fact operations...")

        initial_stats = self.get_stats()
        initial_kv_count = initial_stats.get("key_value_count", 0)

        # Save some facts
        for i in range(3):
            result = self.call_tool("remembrance_save_fact", {
                "user_id": self.test_user,
                "key": f"test_key_{i}",
                "value": f"test_value_{i}"
            })
            print(f"Saved fact {i}: {result}")

        # Check stats after saving
        stats = self.get_stats()
        print(f"Stats after saving facts: {stats}")

        expected_kv_count = initial_kv_count + 3
        if stats.get("key_value_count", 0) != expected_kv_count:
            print(
                f"✗ Expected key_value_count to be {expected_kv_count}, got {stats.get('key_value_count', 0)}")
            return False

        # Delete one fact
        result = self.call_tool("remembrance_delete_fact", {
            "user_id": self.test_user,
            "key": "test_key_1"
        })
        print(f"Deleted fact: {result}")

        # Check stats after deleting
        stats = self.get_stats()
        print(f"Stats after deleting one fact: {stats}")

        expected_kv_count = initial_kv_count + 2
        if stats.get("key_value_count", 0) != expected_kv_count:
            print(
                f"✗ Expected key_value_count to be {expected_kv_count}, got {stats.get('key_value_count', 0)}")
            return False

        print("✓ Fact operations update stats correctly")
        return True

    def test_vector_operations(self):
        """Test that vector operations update statistics correctly"""
        print("\n3. Testing vector operations...")

        initial_stats = self.get_stats()
        initial_vector_count = initial_stats.get("vector_count", 0)

        # Add some vectors
        for i in range(2):
            result = self.call_tool("remembrance_add_vector", {
                "user_id": self.test_user,
                "content": f"This is test content {i} for vector testing",
                "metadata": {"test": "true", "index": str(i)}
            })
            print(f"Added vector {i}: {result}")

        # Check stats after adding vectors
        stats = self.get_stats()
        print(f"Stats after adding vectors: {stats}")

        expected_vector_count = initial_vector_count + 2
        if stats.get("vector_count", 0) != expected_vector_count:
            print(
                f"✗ Expected vector_count to be {expected_vector_count}, got {stats.get('vector_count', 0)}")
            return False

        print("✓ Vector operations update stats correctly")
        return True

    def test_entity_operations(self):
        """Test that entity operations update global statistics"""
        print("\n4. Testing entity operations...")

        initial_stats = self.get_stats()
        initial_entity_count = initial_stats.get("entity_count", 0)

        # Create an entity
        result = self.call_tool("remembrance_create_entity", {
            "entity_type": "person",
            "name": "Test Person Stats",
            "properties": {"role": "test_subject"}
        })
        print(f"Created entity: {result}")

        # Check stats after creating entity
        stats = self.get_stats()
        print(f"Stats after creating entity: {stats}")

        expected_entity_count = initial_entity_count + 1
        if stats.get("entity_count", 0) != expected_entity_count:
            print(
                f"✗ Expected entity_count to be {expected_entity_count}, got {stats.get('entity_count', 0)}")
            return False

        print("✓ Entity operations update global stats correctly")
        return True

    def run_tests(self):
        """Run all tests"""
        print("Starting user statistics tests...")

        if not self.setup():
            return False

        try:
            tests = [
                self.test_initial_stats,
                self.test_fact_operations,
                self.test_vector_operations,
                self.test_entity_operations
            ]

            passed = 0
            total = len(tests)

            for test in tests:
                if test():
                    passed += 1
                else:
                    print(f"✗ Test {test.__name__} failed")

            print(f"\n{'='*50}")
            print(f"Tests completed: {passed}/{total} passed")

            if passed == total:
                print("✓ All user statistics tests passed!")
                return True
            else:
                print(f"✗ {total - passed} tests failed")
                return False

        finally:
            self.cleanup()


if __name__ == "__main__":
    test = UserStatsTest()
    success = test.run_tests()
    sys.exit(0 if success else 1)
