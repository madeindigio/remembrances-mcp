#!/usr/bin/env python3
"""
End-to-End MCP Client for Remembrances-MCP Server Testing.

This client implements comprehensive testing of all MCP tools including:
- TOON format responses validation
- Levenshtein suggestions functionality
- Full CRUD operations across all data layers

Based on the development plan phases (user_id='plan'):
- e2e-phase-1-setup: Server launch and client initialization
- e2e-phase-2-seed: Create baseline test data
- e2e-phase-3-run: Exercise all tools and validate responses
- e2e-phase-4-teardown: Cleanup and reporting

Usage:
    python tests/test_mcp_e2e.py [--config CONFIG_FILE] [--server SERVER_BINARY]
"""

import argparse
import os
import sys

# Add the e2e module to path
sys.path.insert(0, os.path.dirname(__file__))

from e2e.client import MCPClient
from e2e.test_data import TestContext
from e2e.phases import phase_1_setup, phase_2_seed, phase_3_run, phase_4_teardown


def main():
    parser = argparse.ArgumentParser(description="MCP E2E Test Client")
    parser.add_argument("--server", default="../build/remembrances-mcp",
                       help="Path to remembrances-mcp binary")
    parser.add_argument("--config",
                       help="Path to config file (default: auto-detect)")
    parser.add_argument("--timeout", type=int, default=30,
                       help="Timeout for server operations")

    args = parser.parse_args()

    # Check server binary
    if not os.path.exists(args.server):
        # Try relative to script directory
        script_dir = os.path.dirname(os.path.abspath(__file__))
        server_path = os.path.join(script_dir, "..", "build", "remembrances-mcp")
        if os.path.exists(server_path):
            args.server = server_path
        else:
            print(f"ERROR: Server binary not found at {args.server} or {server_path}")
            sys.exit(1)

    print(f"Using server binary: {args.server}")

    # Initialize client
    client = MCPClient([args.server])
    context = TestContext()

    try:
        # Run test phases
        phases = [
            ("Setup", lambda c, ctx: phase_1_setup(c)),
            ("Seed", phase_2_seed),
            ("Run", phase_3_run),
            ("Teardown", phase_4_teardown)
        ]

        results = []
        for phase_name, phase_func in phases:
            try:
                print(f"\n🚀 Starting {phase_name} phase...")
                success = phase_func(client, context)
                results.append((phase_name, success))
                if success:
                    print(f"✅ {phase_name} phase completed successfully")
                else:
                    print(f"❌ {phase_name} phase failed")
            except Exception as e:
                print(f"💥 {phase_name} phase crashed: {e}")
                results.append((phase_name, False))
                break

        # Final report
        print("\n" + "="*60)
        print("FINAL RESULTS")
        print("="*60)

        all_passed = True
        for phase_name, success in results:
            status = "✅ PASS" if success else "❌ FAIL"
            print(f"{phase_name}: {status}")
            if not success:
                all_passed = False

        if all_passed:
            print("\n🎉 ALL TESTS PASSED!")
            sys.exit(0)
        else:
            print("\n💥 SOME TESTS FAILED!")
            sys.exit(1)
    finally:
        client.close()


if __name__ == "__main__":
    main()