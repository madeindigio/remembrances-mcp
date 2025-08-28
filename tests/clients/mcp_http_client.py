#!/usr/bin/env python3
"""
Simple HTTP client for testing the MCP HTTP JSON API transport.
Tests basic HTTP endpoints exposed by the remembrances-mcp server.
"""

import argparse
import json
import sys
import requests
import time


def test_health(base_url):
    """Test the health endpoint."""
    print("Testing health endpoint...")
    try:
        response = requests.get(f"{base_url}/health", timeout=5)
        response.raise_for_status()
        data = response.json()
        if data.get("status") == "ok":
            print("✓ Health check passed")
            return True
        else:
            print(f"✗ Health check failed: unexpected response {data}")
            return False
    except Exception as e:
        print(f"✗ Health check failed: {e}")
        return False


def test_tools_list(base_url):
    """Test the tools listing endpoint."""
    print("Testing tools list endpoint...")
    try:
        response = requests.get(f"{base_url}/mcp/tools", timeout=5)
        response.raise_for_status()
        data = response.json()
        if "tools" in data:
            print(f"✓ Tools list returned {len(data['tools'])} tools")
            return True
        else:
            print(f"✗ Tools list failed: no 'tools' key in response {data}")
            return False
    except Exception as e:
        print(f"✗ Tools list failed: {e}")
        return False


def test_tool_call(base_url):
    """Test the tool call endpoint."""
    print("Testing tool call endpoint...")
    try:
        payload = {
            "name": "remembrance_save_fact",
            "arguments": {
                "key": "test_key_http",
                "value": "test_value_from_http_client"
            }
        }
        response = requests.post(
            f"{base_url}/mcp/tools/call",
            json=payload,
            headers={"Content-Type": "application/json"},
            timeout=5
        )
        response.raise_for_status()
        data = response.json()
        if "content" in data:
            print("✓ Tool call succeeded")
            return True
        else:
            print(f"✗ Tool call failed: no 'content' key in response {data}")
            return False
    except Exception as e:
        print(f"✗ Tool call failed: {e}")
        return False


def test_cors_preflight(base_url):
    """Test CORS preflight request."""
    print("Testing CORS preflight...")
    try:
        response = requests.options(
            f"{base_url}/mcp/tools/call",
            headers={
                "Origin": "http://localhost:3000",
                "Access-Control-Request-Method": "POST",
                "Access-Control-Request-Headers": "Content-Type"
            },
            timeout=5
        )
        response.raise_for_status()
        if response.status_code == 200:
            print("✓ CORS preflight succeeded")
            return True
        else:
            print(f"✗ CORS preflight failed: status {response.status_code}")
            return False
    except Exception as e:
        print(f"✗ CORS preflight failed: {e}")
        return False


def main():
    parser = argparse.ArgumentParser(
        description="Test MCP HTTP JSON API transport")
    parser.add_argument("--base-url", default="http://localhost:8081",
                        help="Base URL of the MCP HTTP server")
    parser.add_argument("--timeout", type=int, default=10,
                        help="Timeout for server to be ready")
    args = parser.parse_args()

    base_url = args.base_url.rstrip('/')

    print(f"Testing MCP HTTP API at {base_url}")

    # Wait for server to be ready
    print(f"Waiting up to {args.timeout}s for server to be ready...")
    for _ in range(args.timeout):
        try:
            response = requests.get(f"{base_url}/health", timeout=2)
            if response.status_code == 200:
                print("Server is ready!")
                break
        except requests.RequestException:
            pass
        time.sleep(1)
    else:
        print("Server did not become ready in time")
        sys.exit(1)

    # Run tests
    tests = [
        test_health,
        test_tools_list,
        test_cors_preflight,
        test_tool_call,
    ]

    results = []
    for test_func in tests:
        result = test_func(base_url)
        results.append(result)
        print()  # Add spacing between tests

    # Summary
    passed = sum(results)
    total = len(results)
    print(f"Test Results: {passed}/{total} passed")

    if passed == total:
        print("✓ All HTTP tests passed!")
        sys.exit(0)
    else:
        print("✗ Some HTTP tests failed!")
        sys.exit(1)


if __name__ == "__main__":
    main()
