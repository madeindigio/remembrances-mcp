#!/usr/bin/env python3
"""
MCP Client for E2E Testing.

Improved stdio client that properly handles server initialization and MCP communication.
"""

import json
import subprocess
import tempfile
import os
import sys
import shutil
import time
import threading
import select


class MCPClient:
    """Improved stdio MCP client that handles server initialization properly."""
    
    def __init__(self, server_cmd: list[str]):
        self.server_cmd = server_cmd
        self.process = None
        self.temp_dir = None
        self.request_id = 0
        self.log_file = None
    
    def cleanup_test_artifacts(self):
        """Clean up test database and log files before starting server."""
        # Clean up test database
        test_db_path = "/tmp/remembrances_test.db"
        if os.path.exists(test_db_path):
            try:
                shutil.rmtree(test_db_path)
                print(f"✅ Cleaned up test database: {test_db_path}")
            except Exception as e:
                print(f"⚠️  Failed to clean test database: {e}")
        
        # Clean up test log file
        script_dir = os.path.dirname(os.path.dirname(os.path.dirname(__file__)))
        log_file = os.path.join(script_dir, "remembrances-mcp-test.log")
        if os.path.exists(log_file):
            try:
                os.remove(log_file)
                print(f"✅ Cleaned up test log: {log_file}")
            except Exception as e:
                print(f"⚠️  Failed to clean test log: {e}")
        
        self.log_file = log_file
    
    def start_server(self):
        """Start the MCP server process."""
        if self.process is not None:
            return True
        
        # Clean up test artifacts from previous runs
        self.cleanup_test_artifacts()
            
        # Create temp directory first
        self.temp_dir = tempfile.mkdtemp(prefix="mcp_e2e_")
            
        # Build the full command with config
        cmd = self.server_cmd.copy()
        # Get the project root directory (parent of tests directory)
        script_dir = os.path.dirname(os.path.dirname(os.path.dirname(__file__)))
        print(f"Project dir: {script_dir}")
        
        # Create a temporary config file to avoid conflicts with user config
        temp_config = os.path.join(self.temp_dir, "config.yaml")
        config_file = os.path.join(script_dir, "config.test.yaml")
        print(f"Looking for config at: {config_file}")
        if os.path.exists(config_file):
            shutil.copy2(config_file, temp_config)
            print(f"Copied config from {config_file} to {temp_config}")
        else:
            config_file = os.path.join(script_dir, "config.sample.gguf.yaml")
            print(f"Trying fallback config at: {config_file}")
            if os.path.exists(config_file):
                shutil.copy2(config_file, temp_config)
                print(f"Copied config from {config_file} to {temp_config}")
            else:
                print(f"❌ No config file found")
                return False
        
        # Use the temporary config file
        cmd.extend(["--config", temp_config])

        # Add temp db path (this will override any db-path in config)
        #cmd.extend(["--db-path", os.path.join(self.temp_dir, "test.db")])

        print(f"Starting server: {' '.join(cmd)}")
        self.process = subprocess.Popen(
            cmd,
            stdin=subprocess.PIPE,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            text=True,
            bufsize=1  # Line buffered
        )
        
        # Start threads to consume stdout and stderr
        self.stdout_thread = threading.Thread(target=self._consume_stdout, daemon=True)
        self.stderr_thread = threading.Thread(target=self._consume_stderr, daemon=True)
        self.stdout_thread.start()
        self.stderr_thread.start()
        
        # Wait for server to be fully initialized
        print("Waiting for server initialization...")
        start_time = time.time()
        timeout = 15  # Reduced from 60 to 15 seconds
        
        while time.time() - start_time < timeout:
            if self.process.poll() is not None:
                print("❌ Server process terminated unexpectedly")
                return False
            
            # Check stderr for initialization message
            if hasattr(self, 'server_ready') and self.server_ready:
                print("✅ Server is ready (detected via stderr)")
                return True
            
            # Also check log file for initialization message
            if self.log_file and os.path.exists(self.log_file):
                try:
                    with open(self.log_file, 'r') as f:
                        log_content = f.read()
                        if "Remembrances-MCP server initialized successfully" in log_content:
                            self.server_ready = True
                            print("✅ Server is ready (detected via log file)")
                            return True
                except Exception as e:
                    # Log file may not be ready yet, continue waiting
                    pass
            
            time.sleep(0.5)  # Check more frequently (every 500ms instead of 1s)
        
        print(f"❌ Server failed to initialize within {timeout} seconds")
        print(f"   Check log file: {self.log_file}")
        return False
    
    def _consume_stdout(self):
        """Consume stdout and buffer JSON responses."""
        self.response_buffer = []
        try:
            while True:
                line = self.process.stdout.readline()
                if not line:
                    break
                line = line.strip()
                if line and line.startswith('{'):
                    # This looks like a JSON response
                    try:
                        response = json.loads(line)
                        self.response_buffer.append(response)
                    except json.JSONDecodeError:
                        # Not valid JSON, skip
                        pass
        except:
            pass
    
    def _consume_stderr(self):
        """Consume stderr and look for initialization message."""
        self.server_ready = False
        try:
            while True:
                line = self.process.stderr.readline()
                if not line:
                    break
                line = line.strip()
                # Look for the initialization message
                if "Remembrances-MCP server initialized successfully" in line:
                    self.server_ready = True
                    print("Detected server initialization message")
                # Also print stderr for debugging
                if line:
                    print(f"[SERVER] {line}", file=sys.stderr)
        except:
            pass
    
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
        
        # Send request
        request_json = json.dumps(request) + "\n"
        try:
            self.process.stdin.write(request_json)
            self.process.stdin.flush()
        except BrokenPipeError:
            return {"error": "Broken pipe - server may have terminated"}
        
        # Wait for response in buffer
        start_time = time.time()
        while time.time() - start_time < 30:  # Wait up to 30 seconds
            if self.process.poll() is not None:
                return {"error": "Server process terminated"}
            
            # Check if we have a response for our request
            for response in self.response_buffer:
                if isinstance(response, dict) and response.get("id") == self.request_id:
                    self.response_buffer.remove(response)
                    return response
            
            time.sleep(0.1)
        
        return {"error": "Timeout waiting for response"}
    
    def close(self):
        """Close the MCP server process."""
        if self.process:
            try:
                self.process.terminate()
                self.process.wait(timeout=5)
                print("✅ Server stopped")
            except subprocess.TimeoutExpired:
                self.process.kill()
                self.process.wait()
                print("⚠️  Server force-killed")

        if self.temp_dir and os.path.exists(self.temp_dir):
            shutil.rmtree(self.temp_dir)
            print(f"✅ Cleaned up temp directory: {self.temp_dir}")