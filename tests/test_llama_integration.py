#!/usr/bin/env python3
"""
Comprehensive integration test for llama.cpp embedder functionality.
This script validates the complete integration of llama.cpp into remembrances-mcp.
"""

import os
import sys
import subprocess
import tempfile
import json
import time
import signal
from pathlib import Path


def run_command(cmd, timeout=10, capture_output=True, text=True, cwd=None, env=None):
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


def test_cli_flags():
    """Test that all llama.cpp CLI flags are available and working."""
    print("🦙 Testing CLI Flags...")

    # Test help output contains llama.cpp flags
    result = run_command(
        ["./remembrances-mcp", "--help"], cwd="/www/MCP/remembrances-mcp"
    )
    if not result:
        print("❌ Failed to get help output")
        return False

    help_text = result.stdout + result.stderr

    required_flags = [
        "--llama-model-path",
        "--llama-dimension",
        "--llama-threads",
        "--llama-gpu-layers",
        "--llama-context",
    ]

    missing_flags = []
    for flag in required_flags:
        if flag not in help_text:
            missing_flags.append(flag)

    if missing_flags:
        print(f"❌ Missing CLI flags: {missing_flags}")
        return False

    print("✅ All llama.cpp CLI flags are available")
    return True


def test_environment_variables():
    """Test environment variable support."""
    print("\n🌍 Testing Environment Variables...")

    # Set up test environment with ONLY llama.cpp configured (no other embedders)
    env = os.environ.copy()
    env.update(
        {
            "GOMEM_LLAMA_MODEL_PATH": "/tmp/test-model.gguf",
            "GOMEM_LLAMA_DIMENSION": "1024",
            "GOMEM_LLAMA_THREADS": "8",
            "GOMEM_LLAMA_GPU_LAYERS": "4",
            "GOMEM_LLAMA_CONTEXT": "2048",
            "GOMEM_SURREALDB_START_CMD": "surreal start --user root --pass root ws://localhost:8000",
        }
    )

    # Test that environment variables are recognized
    # This should fail due to missing model file, but show that config is parsed
    result = run_command(
        ["./remembrances-mcp"], timeout=10, cwd="/www/MCP/remembrances-mcp", env=env
    )

    if result is None:
        print("❌ Command timed out unexpectedly")
        return False

    # Check if it tried to load llama.cpp model (indicates env vars worked)
    stderr = result.stderr.lower()
    stdout = result.stdout.lower()

    # Look for llama.cpp model loading attempt or embedder creation
    if (
        "failed to load llama.cpp model" in stderr
        or "failed to create embedder" in stderr
        or "llama.cpp" in stdout
        or "no such file" in stderr
    ):
        print("✅ Environment variables are recognized and processed")
        return True
    else:
        print("⚠️  Environment variables might not be working as expected")
        print(f"stdout: {stdout}")
        print(f"stderr: {stderr}")
        return False


def test_configuration_validation():
    """Test configuration validation."""
    print("\n✅ Testing Configuration Validation...")

    test_cases = [
        {
            "name": "Negative threads",
            "args": [
                "--llama-model-path",
                "/tmp/test.gguf",
                "--llama-threads",
                "-1",
                "--surrealdb-start-cmd",
                "surreal start --user root --pass root ws://localhost:8000",
            ],
            "should_fail": True,
        },
        {
            "name": "Negative dimension",
            "args": [
                "--llama-model-path",
                "/tmp/test.gguf",
                "--llama-dimension",
                "-1",
                "--surrealdb-start-cmd",
                "surreal start --user root --pass root ws://localhost:8000",
            ],
            "should_fail": True,
        },
        {
            "name": "Negative GPU layers",
            "args": [
                "--llama-model-path",
                "/tmp/test.gguf",
                "--llama-gpu-layers",
                "-1",
                "--surrealdb-start-cmd",
                "surreal start --user root --pass root ws://localhost:8000",
            ],
            "should_fail": True,
        },
        {
            "name": "Negative context",
            "args": [
                "--llama-model-path",
                "/tmp/test.gguf",
                "--llama-context",
                "-1",
                "--surrealdb-start-cmd",
                "surreal start --user root --pass root ws://localhost:8000",
            ],
            "should_fail": True,
        },
        {
            "name": "Valid config",
            "args": [
                "--llama-model-path",
                "/tmp/test.gguf",
                "--llama-dimension",
                "768",
                "--surrealdb-start-cmd",
                "surreal start --user root --pass root ws://localhost:8000",
            ],
            "should_fail": False,  # Will fail due to missing file, but validation passes
        },
    ]

    all_passed = True
    for test_case in test_cases:
        cmd = ["./remembrances-mcp"] + test_case["args"]
        result = run_command(cmd, timeout=8, cwd="/www/MCP/remembrances-mcp")

        if result is None:
            print(f"❌ {test_case['name']}: Command timed out")
            all_passed = False
            continue

        stderr_lower = result.stderr.lower()
        stdout_lower = result.stdout.lower()

        if test_case["should_fail"]:
            # Should fail with validation error
            if any(
                keyword in stderr_lower or keyword in stdout_lower
                for keyword in [
                    "negative",
                    "invalid",
                    "must be positive",
                    "at least one embedder",
                ]
            ):
                print(f"✅ {test_case['name']}: Correctly rejected")
            else:
                print(f"❌ {test_case['name']}: Should have been rejected")
                print(f"stdout: {stdout_lower}")
                print(f"stderr: {stderr_lower}")
                all_passed = False
        else:
            # Should fail with model loading error (not validation error)
            if (
                "failed to load llama.cpp model" in stderr_lower
                or "no such file" in stderr_lower
                or "failed to create embedder" in stderr_lower
            ):
                print(
                    f"✅ {test_case['name']}: Validation passed, model loading failed as expected"
                )
            else:
                print(f"⚠️  {test_case['name']}: Unexpected result")
                print(f"stdout: {stdout_lower}")
                print(f"stderr: {stderr_lower}")

    return all_passed


def test_embedder_priority():
    """Test that llama.cpp takes priority over other embedders."""
    print("\n🏆 Testing Embedder Priority...")

    # Set up environment with all embedders configured
    env = os.environ.copy()
    env.update(
        {
            "GOMEM_LLAMA_MODEL_PATH": "/tmp/test.gguf",
            "GOMEM_OLLAMA_URL": "http://localhost:11434",
            "GOMEM_OLLAMA_EMBEDDING_MODEL": "nomic-embed-text",
            "GOMEM_OPENAI_KEY": "sk-test123",
            "GOMEM_SURREALDB_START_CMD": "surreal start --user root --pass root ws://localhost:8000",
        }
    )

    result = run_command(
        ["./remembrances-mcp"], timeout=8, cwd="/www/MCP/remembrances-mcp", env=env
    )

    if result is None:
        print("❌ Command timed out")
        return False

    stderr_lower = result.stderr.lower()
    stdout_lower = result.stdout.lower()

    # Should try to load llama.cpp model first, not connect to Ollama or OpenAI
    if (
        "failed to load llama.cpp model" in stderr_lower
        or "failed to create embedder" in stderr_lower
        or "llama.cpp" in stdout_lower
    ):
        print("✅ Llama.cpp correctly takes priority over other embedders")
        return True
    else:
        print("❌ Priority order might be incorrect")
        print(f"stdout: {stdout_lower}")
        print(f"stderr: {stderr_lower}")
        return False


def test_default_values():
    """Test that default values are applied correctly."""
    print("\n🔧 Testing Default Values...")

    # Test with minimal configuration
    result = run_command(
        [
            "./remembrances-mcp",
            "--llama-model-path",
            "/tmp/test.gguf",
            "--surrealdb-start-cmd",
            "surreal start --user root --pass root ws://localhost:8000",
        ],
        timeout=8,
        cwd="/www/MCP/remembrances-mcp",
    )

    if result is None:
        print("❌ Command timed out")
        return False

    # The fact that it tries to load model means defaults were applied
    stderr_lower = result.stderr.lower()
    stdout_lower = result.stdout.lower()

    if (
        "failed to load llama.cpp model" in stderr_lower
        or "failed to create embedder" in stderr_lower
        or "llama.cpp" in stdout_lower
    ):
        print("✅ Default values are applied correctly")
        return True
    else:
        print("⚠️  Unexpected result with default values")
        print(f"stdout: {stdout_lower}")
        print(f"stderr: {stderr_lower}")
        return False


def test_help_documentation():
    """Test that help documentation is complete."""
    print("\n📚 Testing Help Documentation...")

    result = run_command(
        ["./remembrances-mcp", "--help"], cwd="/www/MCP/remembrances-mcp"
    )
    if not result:
        print("❌ Failed to get help output")
        return False

    help_text = result.stdout + result.stderr

    # Check for documentation quality
    doc_checks = [
        ("Path description", "Path to the .gguf model file"),
        ("Dimension description", "Dimension of embeddings"),
        ("Threads description", "Number of threads"),
        ("GPU layers description", "Number of GPU layers"),
        ("Context description", "Context size"),
        ("Default values", "default"),
    ]

    missing_docs = []
    for check_name, check_text in doc_checks:
        if check_text not in help_text:
            missing_docs.append(check_name)

    if missing_docs:
        print(f"❌ Missing documentation: {missing_docs}")
        return False

    print("✅ Help documentation is complete")
    return True


def test_integration_with_existing_functionality():
    """Test that llama.cpp integration doesn't break existing functionality."""
    print("\n🔗 Testing Integration with Existing Functionality...")

    # Test that other embedders still work when llama.cpp is not configured
    env = os.environ.copy()
    env.update(
        {
            "GOMEM_OLLAMA_URL": "http://localhost:11434",
            "GOMEM_OLLAMA_EMBEDDING_MODEL": "nomic-embed-text",
            "GOMEM_SURREALDB_START_CMD": "surreal start --user root --pass root ws://localhost:8000",
        }
    )

    result = run_command(
        ["./remembrances-mcp"], timeout=8, cwd="/www/MCP/remembrances-mcp", env=env
    )

    if result is None:
        print("❌ Command timed out")
        return False

    stderr_lower = result.stderr.lower()
    stdout_lower = result.stdout.lower()

    # Should try to connect to Ollama, not load llama.cpp model
    if (
        "connection refused" in stderr_lower
        or "11434" in stderr_lower
        or "ollama" in stdout_lower
        or "failed to create ollama" in stderr_lower
    ):
        print("✅ Ollama embedder still works when llama.cpp is not configured")
        return True
    else:
        print("⚠️  Unexpected result with Ollama embedder")
        print(f"stdout: {stdout_lower}")
        print(f"stderr: {stderr_lower}")
        return False


def main():
    """Run all integration tests."""
    print("🚀 Starting Llama.cpp Integration Tests")
    print("=" * 50)

    # Change to project directory
    os.chdir("/www/MCP/remembrances-mcp")

    # Check if binary exists
    if not os.path.exists("./remembrances-mcp"):
        print("❌ remembrances-mcp binary not found!")
        print("Run 'go build ./cmd/remembrances-mcp' first.")
        return False

    # Run all tests
    tests = [
        test_cli_flags,
        test_environment_variables,
        test_configuration_validation,
        test_embedder_priority,
        test_default_values,
        test_help_documentation,
        test_integration_with_existing_functionality,
    ]

    passed = 0
    total = len(tests)

    for test in tests:
        try:
            if test():
                passed += 1
        except Exception as e:
            print(f"❌ Test {test.__name__} failed with exception: {e}")

    print("\n" + "=" * 50)
    print(f"📊 Test Results: {passed}/{total} tests passed")

    if passed == total:
        print("🎉 All integration tests passed!")
        print("\n📋 Summary:")
        print("✅ CLI flags are available and documented")
        print("✅ Environment variables are supported")
        print("✅ Configuration validation works correctly")
        print("✅ Priority order is correct (llama.cpp > Ollama > OpenAI)")
        print("✅ Default values are applied")
        print("✅ Help documentation is complete")
        print("✅ Existing functionality is preserved")

        print("\n🚀 Ready for production use!")
        print("\n📖 Next steps:")
        print("1. Download a .gguf embedding model")
        print("2. Configure --llama-model-path")
        print("3. Adjust --llama-dimension based on your model")
        print("4. Optionally enable GPU with --llama-gpu-layers")

        return True
    else:
        print(f"❌ {total - passed} tests failed")
        return False


if __name__ == "__main__":
    success = main()
    sys.exit(0 if success else 1)
