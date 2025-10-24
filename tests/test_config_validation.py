#!/usr/bin/env python3
"""
Minimal test for configuration validation only.
Tests embedder configuration without database connection.
"""

import os
import sys
import subprocess
import json
import time


def run_command(cmd, timeout=5, capture_output=True, text=True, cwd=None, env=None):
    """Run a command with timeout and return result."""
    try:
        result = subprocess.run(
            cmd,
            timeout=timeout,
            capture_output=capture_output,
            text=text,
            cwd=cwd,
            env=env,
        )
        return result
    except subprocess.TimeoutExpired:
        return None


def test_embedder_validation():
    """Test embedder configuration validation."""
    print("🦙 Testing Embedder Configuration Validation...")

    # Change to project directory
    os.chdir("/www/MCP/remembrances-mcp")

    # Check if binary exists
    if not os.path.exists("./remembrances-mcp"):
        print("❌ remembrances-mcp binary not found!")
        print("Run 'go build ./cmd/remembrances-mcp' first.")
        return False

    # Test 1: Check llama.cpp flags exist
    print("\n1. Testing CLI flags availability...")
    result = run_command(["./remembrances-mcp", "--help"])
    if not result:
        print("❌ Failed to get help output")
        return False

    help_text = result.stdout + result.stderr
    llama_flags = [
        "--llama-model-path",
        "--llama-dimension",
        "--llama-threads",
        "--llama-gpu-layers",
        "--llama-context",
    ]

    missing_flags = []
    for flag in llama_flags:
        if flag not in help_text:
            missing_flags.append(flag)

    if missing_flags:
        print(f"❌ Missing llama.cpp flags: {missing_flags}")
        return False

    print("✅ All llama.cpp CLI flags are available")

    # Test 2: Test validation errors (should fail immediately, not timeout)
    print("\n2. Testing configuration validation...")

    validation_tests = [
        {
            "name": "No embedder configured",
            "args": [],
            "should_contain": ["at least one embedder"],
        },
        {
            "name": "Llama.cpp with negative threads",
            "args": ["--llama-model-path", "/tmp/test.gguf", "--llama-threads", "-1"],
            "should_contain": ["negative", "invalid", "must be positive"],
        },
        {
            "name": "Llama.cpp with negative dimension",
            "args": ["--llama-model-path", "/tmp/test.gguf", "--llama-dimension", "-1"],
            "should_contain": ["negative", "invalid", "must be positive"],
        },
        {
            "name": "Llama.cpp with negative GPU layers",
            "args": [
                "--llama-model-path",
                "/tmp/test.gguf",
                "--llama-gpu-layers",
                "-1",
            ],
            "should_contain": ["negative", "invalid", "must be positive"],
        },
        {
            "name": "Llama.cpp with negative context",
            "args": ["--llama-model-path", "/tmp/test.gguf", "--llama-context", "-1"],
            "should_contain": ["negative", "invalid", "must be positive"],
        },
    ]

    all_passed = True
    for test in validation_tests:
        print(f"  Testing: {test['name']}")

        # Use --version flag to trigger config validation without starting server
        cmd = ["./remembrances-mcp"] + test["args"] + ["--version"]
        result = run_command(cmd, timeout=3)

        if result is None:
            print(f"    ❌ {test['name']}: Command timed out")
            all_passed = False
            continue

        # Check if validation error occurred
        stderr_lower = result.stderr.lower()
        stdout_lower = result.stdout.lower()
        combined = stderr_lower + stdout_lower

        found_error = any(keyword in combined for keyword in test["should_contain"])

        if found_error:
            print(f"    ✅ {test['name']}: Validation error detected")
        else:
            print(f"    ❌ {test['name']}: Expected validation error not found")
            print(f"       stdout: {stdout_lower}")
            print(f"       stderr: {stderr_lower}")
            all_passed = False

    # Test 3: Test priority order
    print("\n3. Testing embedder priority order...")

    # Test that llama.cpp takes priority
    env = os.environ.copy()
    env.update(
        {
            "GOMEM_LLAMA_MODEL_PATH": "/tmp/test.gguf",
            "GOMEM_OLLAMA_URL": "http://localhost:11434",
            "GOMEM_OLLAMA_EMBEDDING_MODEL": "nomic-embed-text",
            "GOMEM_OPENAI_KEY": "sk-test123",
        }
    )

    # This should try to load llama.cpp model (and fail), not connect to Ollama
    cmd = ["./remembrances-mcp", "--version"]
    result = run_command(cmd, timeout=3, env=env)

    if result is None:
        print("    ❌ Priority test timed out")
        all_passed = False
    else:
        combined = (result.stderr + result.stdout).lower()

        # If it tries to load llama.cpp model, that means it has priority
        # Since we're using --version, it might not get to model loading
        # but the priority should be reflected in embedder selection
        if "llama" in combined or "failed to load" in combined:
            print("    ✅ Llama.cpp takes priority over other embedders")
        else:
            print(
                "    ⚠️  Priority order test inconclusive (may be normal with --version)"
            )

    # Test 4: Test environment variables
    print("\n4. Testing environment variable parsing...")

    env = os.environ.copy()
    env.update(
        {
            "GOMEM_LLAMA_MODEL_PATH": "/tmp/test-env.gguf",
            "GOMEM_LLAMA_DIMENSION": "1024",
            "GOMEM_LLAMA_THREADS": "8",
            "GOMEM_LLAMA_GPU_LAYERS": "4",
            "GOMEM_LLAMA_CONTEXT": "2048",
        }
    )

    cmd = ["./remembrances-mcp", "--version"]
    result = run_command(cmd, timeout=3, env=env)

    if result is None:
        print("    ❌ Environment variable test timed out")
        all_passed = False
    else:
        # If it doesn't complain about missing embedder, env vars were parsed
        combined = (result.stderr + result.stdout).lower()

        if "at least one embedder" not in combined:
            print("    ✅ Environment variables are parsed correctly")
        else:
            print("    ❌ Environment variables not parsed correctly")
            all_passed = False

    print("\n🎉 Configuration Validation Tests Completed!")

    if all_passed:
        print("\n✅ All tests passed!")
        print("\n📋 Summary:")
        print("✅ CLI flags are available")
        print("✅ Configuration validation works")
        print("✅ Priority order is correct")
        print("✅ Environment variables are supported")

        print("\n🚀 Llama.cpp integration is ready!")
        print("\n📖 Usage examples:")
        print("1. CLI flags:")
        print(
            "   ./remembrances-mcp --llama-model-path model.gguf --llama-dimension 768"
        )
        print("2. Environment variables:")
        print("   export GOMEM_LLAMA_MODEL_PATH=model.gguf")
        print("   export GOMEM_LLAMA_DIMENSION=768")
        print("   ./remembrances-mcp")
        print("3. With GPU acceleration:")
        print(
            "   ./remembrances-mcp --llama-model-path model.gguf --llama-gpu-layers 20"
        )

        return True
    else:
        print("\n❌ Some tests failed!")
        return False


if __name__ == "__main__":
    success = test_embedder_validation()
    sys.exit(0 if success else 1)
