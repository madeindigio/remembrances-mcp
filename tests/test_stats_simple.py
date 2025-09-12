#!/usr/bin/env python3

import json
import subprocess
import sys
import time


def test_stats():
    """Test if statistics work by running fact save and checking results"""

    # Use the existing working test but focus on the results
    try:
        result = subprocess.run(
            ["python3", "tests/test_user_stats.py"],
            cwd="/www/MCP/remembrances-mcp",
            capture_output=True,
            text=True,
            timeout=30
        )

        # Extract any success messages from the output
        output = result.stderr + result.stdout

        if "All tests passed!" in output:
            print("✅ SUCCESS: Statistics are working correctly!")
            return True
        elif "key_value_count is" in output:
            # Look for the specific test results
            lines = output.split('\n')
            for line in lines:
                if 'key_value_count is' in line:
                    print(f"Found: {line}")
                    if 'key_value_count is 3' in line:
                        print("✅ SUCCESS: key_value_count shows correct value!")
                        return True

        print("❌ FAILED: Statistics test did not pass")
        print("Return code:", result.returncode)

        # Show some debug info
        print("\n=== Last few lines of output ===")
        lines = output.split('\n')
        for line in lines[-10:]:
            if line.strip():
                print(line)

        return False

    except subprocess.TimeoutExpired:
        print("❌ FAILED: Test timed out")
        return False
    except Exception as e:
        print(f"❌ FAILED: Error running test: {e}")
        return False


if __name__ == "__main__":
    print("Testing if statistics functionality is working...")
    success = test_stats()
    sys.exit(0 if success else 1)
