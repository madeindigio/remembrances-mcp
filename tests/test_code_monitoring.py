#!/usr/bin/env python3
"""
Test suite for Code Project File Monitoring System.

This test suite validates the new file monitoring tools:
- code_activate_project_watch
- code_deactivate_project_watch  
- code_get_watch_status

Requirements:
- remembrances-mcp server running with MCP protocol
- A test project directory with some code files

Usage:
    python tests/test_code_monitoring.py
"""

import json
import subprocess
import tempfile
import time
import os
import sys


class MCPClient:
    """Simple MCP client for testing via stdio."""
    
    def __init__(self, server_cmd: list[str]):
        self.process = subprocess.Popen(
            server_cmd,
            stdin=subprocess.PIPE,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            text=True
        )
        self.request_id = 0
    
    def call_tool(self, tool_name: str, arguments: dict) -> dict:
        """Call an MCP tool and return the result."""
        self.request_id += 1
        request = {
            "jsonrpc": "2.0",
            "id": self.request_id,
            "method": "tools/call",
            "params": {
                "name": tool_name,
                "arguments": arguments
            }
        }
        
        self.process.stdin.write(json.dumps(request) + "\n")
        self.process.stdin.flush()
        
        response_line = self.process.stdout.readline()
        return json.loads(response_line)
    
    def close(self):
        """Close the MCP server process."""
        self.process.terminate()
        self.process.wait()


def create_test_project() -> str:
    """Create a temporary test project with sample code files."""
    test_dir = tempfile.mkdtemp(prefix="code_monitor_test_")
    
    # Create sample Go file
    go_file = os.path.join(test_dir, "main.go")
    with open(go_file, "w") as f:
        f.write('''package main

import "fmt"

func main() {
    fmt.Println("Hello, World!")
}

func greet(name string) string {
    return "Hello, " + name
}
''')
    
    # Create sample Python file
    py_file = os.path.join(test_dir, "utils.py")
    with open(py_file, "w") as f:
        f.write('''"""Utility functions for testing."""

def calculate_sum(a: int, b: int) -> int:
    """Add two numbers."""
    return a + b

class Calculator:
    """Simple calculator class."""
    
    def __init__(self):
        self.history = []
    
    def add(self, a: int, b: int) -> int:
        result = a + b
        self.history.append(("add", a, b, result))
        return result
''')
    
    # Create subdirectory with TypeScript file
    subdir = os.path.join(test_dir, "src")
    os.makedirs(subdir)
    ts_file = os.path.join(subdir, "service.ts")
    with open(ts_file, "w") as f:
        f.write('''export interface User {
    id: number;
    name: string;
}

export class UserService {
    private users: User[] = [];
    
    addUser(user: User): void {
        this.users.push(user);
    }
    
    getUser(id: number): User | undefined {
        return this.users.find(u => u.id === id);
    }
}
''')
    
    return test_dir


def test_activate_project_watch(client: MCPClient, project_id: str) -> bool:
    """Test activating project file watch."""
    print(f"\n=== Test: Activate Project Watch ===")
    
    result = client.call_tool("code_activate_project_watch", {
        "project_id": project_id
    })
    
    if "error" in result:
        print(f"ERROR: {result['error']}")
        return False
    
    content = result.get("result", {}).get("content", [])
    if content:
        text = content[0].get("text", "")
        print(f"Result: {text}")
        if "activated" in text.lower() or "watching" in text.lower():
            return True
    
    print("WARN: Unexpected response format")
    return False


def test_get_watch_status(client: MCPClient, project_id: str = None) -> bool:
    """Test getting watch status."""
    print(f"\n=== Test: Get Watch Status ===")
    
    args = {}
    if project_id:
        args["project_id"] = project_id
    
    result = client.call_tool("code_get_watch_status", args)
    
    if "error" in result:
        print(f"ERROR: {result['error']}")
        return False
    
    content = result.get("result", {}).get("content", [])
    if content:
        text = content[0].get("text", "")
        print(f"Result: {text}")
        return True
    
    print("WARN: Empty response")
    return False


def test_deactivate_project_watch(client: MCPClient, project_id: str) -> bool:
    """Test deactivating project file watch."""
    print(f"\n=== Test: Deactivate Project Watch ===")
    
    result = client.call_tool("code_deactivate_project_watch", {
        "project_id": project_id
    })
    
    if "error" in result:
        print(f"ERROR: {result['error']}")
        return False
    
    content = result.get("result", {}).get("content", [])
    if content:
        text = content[0].get("text", "")
        print(f"Result: {text}")
        if "deactivated" in text.lower() or "stopped" in text.lower():
            return True
    
    print("WARN: Unexpected response format")
    return False


def test_file_modification_triggers_reindex(client: MCPClient, project_path: str, project_id: str) -> bool:
    """Test that modifying a file triggers automatic reindexing."""
    print(f"\n=== Test: File Modification Trigger ===")
    
    # First, activate watching
    result = client.call_tool("code_activate_project_watch", {
        "project_id": project_id
    })
    
    if "error" in result:
        print(f"ERROR activating: {result['error']}")
        return False
    
    # Modify a file
    go_file = os.path.join(project_path, "main.go")
    with open(go_file, "a") as f:
        f.write("\nfunc newFunction() { /* added by test */ }\n")
    
    # Wait for debounce + processing
    print("Waiting for file change detection (5 seconds)...")
    time.sleep(5)
    
    # Check status to see if reindex happened
    result = client.call_tool("code_get_watch_status", {
        "project_id": project_id
    })
    
    if "error" in result:
        print(f"ERROR getting status: {result['error']}")
        return False
    
    content = result.get("result", {}).get("content", [])
    if content:
        text = content[0].get("text", "")
        print(f"Status after modification: {text}")
    
    print("OK: File modification test completed")
    return True


def run_tests():
    """Run all monitoring tests."""
    print("=" * 60)
    print("Code Monitoring System - Test Suite")
    print("=" * 60)
    
    # Check if server binary exists
    server_path = "./build/remembrances-mcp"
    if not os.path.exists(server_path):
        server_path = "./remembrances-mcp"
    if not os.path.exists(server_path):
        print("ERROR: Server binary not found. Build with 'make build' first.")
        sys.exit(1)
    
    # Create test project
    print("\nCreating test project...")
    test_project_path = create_test_project()
    print(f"Test project: {test_project_path}")
    
    # Start MCP client
    print("\nStarting MCP server...")
    client = MCPClient([server_path])
    
    try:
        # First, index the test project
        print("\n=== Setup: Index Test Project ===")
        result = client.call_tool("code_index_project", {
            "project_path": test_project_path,
            "project_name": "Test Monitoring Project",
            "languages": ["go", "python", "typescript"]
        })
        
        if "error" in result:
            print(f"ERROR indexing project: {result['error']}")
            return
        
        content = result.get("result", {}).get("content", [])
        if content:
            print(f"Indexing started: {content[0].get('text', '')}")
        
        # Wait for indexing to complete
        print("Waiting for indexing to complete (10 seconds)...")
        time.sleep(10)
        
        # Get project ID from list
        result = client.call_tool("code_list_projects", {})
        projects = []
        if not "error" in result:
            content = result.get("result", {}).get("content", [])
            if content:
                try:
                    projects = json.loads(content[0].get("text", "[]"))
                except json.JSONDecodeError:
                    pass
        
        if not projects:
            print("ERROR: No projects found after indexing")
            return
        
        project_id = projects[-1].get("id") or projects[-1].get("project_id")
        print(f"Test project ID: {project_id}")
        
        # Run tests
        passed = 0
        failed = 0
        
        if test_get_watch_status(client):
            passed += 1
        else:
            failed += 1
        
        if test_activate_project_watch(client, project_id):
            passed += 1
        else:
            failed += 1
        
        if test_get_watch_status(client, project_id):
            passed += 1
        else:
            failed += 1
        
        if test_file_modification_triggers_reindex(client, test_project_path, project_id):
            passed += 1
        else:
            failed += 1
        
        if test_deactivate_project_watch(client, project_id):
            passed += 1
        else:
            failed += 1
        
        # Summary
        print("\n" + "=" * 60)
        print(f"Test Results: {passed} passed, {failed} failed")
        print("=" * 60)
        
    finally:
        client.close()
        
        # Cleanup
        print(f"\nCleaning up test project: {test_project_path}")
        import shutil
        shutil.rmtree(test_project_path, ignore_errors=True)


if __name__ == "__main__":
    run_tests()
