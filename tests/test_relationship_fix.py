#!/usr/bin/env python3
import subprocess
import json
import sys


def send_mcp_request(method, params):
    """Send an MCP request and return the response"""
    request = {
        "jsonrpc": "2.0",
        "id": 1,
        "method": method,
        "params": params
    }

    # Convert to JSON and add newline
    json_input = json.dumps(request) + "\n"

    result = subprocess.run(['./dist/remembrances-mcp'],
                            input=json_input,
                            text=True,
                            capture_output=True)

    return result


def test_relationship_creation():
    """Test creating a relationship between entities"""
    print("Testing relationship creation...")

    # First create entities
    print("\n1. Creating entities...")

    # Create John Developer
    result = send_mcp_request("tools/call", {
        "name": "remembrance_create_entity",
        "arguments": {
            "entity_type": "person",
            "name": "John Developer"
        }
    })

    print("Create John Developer result:")
    print("STDOUT:", result.stdout)
    print("STDERR:", result.stderr)

    # Create Sarah Manager
    result = send_mcp_request("tools/call", {
        "name": "remembrance_create_entity",
        "arguments": {
            "entity_type": "person",
            "name": "Sarah Manager"
        }
    })

    print("\nCreate Sarah Manager result:")
    print("STDOUT:", result.stdout)
    print("STDERR:", result.stderr)

    # Now create relationship
    print("\n2. Creating relationship...")
    result = send_mcp_request("tools/call", {
        "name": "remembrance_create_relationship",
        "arguments": {
            "from_entity": "John Developer",
            "to_entity": "Sarah Manager",
            "relationship_type": "reports_to"
        }
    })

    print("Create relationship result:")
    print("STDOUT:", result.stdout)
    print("STDERR:", result.stderr)


if __name__ == "__main__":
    test_relationship_creation()
