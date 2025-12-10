#!/usr/bin/env python3
"""
Test Data Management for E2E Tests.
"""

import tempfile
import os
from typing import Dict, List, Any


class TestContext:
    """Context for storing test data and results."""

    def __init__(self):
        self.facts: Dict[str, str] = {}
        self.vectors: Dict[str, str] = {}
        self.entities: Dict[str, str] = {}
        self.relationships: List[Dict] = []
        self.documents: Dict[str, str] = {}
        self.events: List[Dict] = []
        self.projects: Dict[str, str] = {}
        self.results: Dict[str, Any] = {}

    def add_fact(self, key: str, value: str):
        self.facts[key] = value

    def add_vector(self, content: str, vector_id: str):
        self.vectors[vector_id] = content

    def add_entity(self, entity_id: str, name: str):
        self.entities[entity_id] = name

    def add_relationship(self, from_id: str, to_id: str, rel_type: str):
        self.relationships.append({
            "from": from_id,
            "to": to_id,
            "type": rel_type
        })

    def add_document(self, path: str, content: str):
        self.documents[path] = content

    def add_event(self, user_id: str, subject: str, content: str):
        self.events.append({
            "user_id": user_id,
            "subject": subject,
            "content": content
        })

    def add_project(self, project_id: str, path: str):
        self.projects[project_id] = path


def create_test_project() -> str:
    """Create a temporary test project with sample code files."""
    project_dir = tempfile.mkdtemp(prefix="test_project_")

    # Create sample Go files
    os.makedirs(os.path.join(project_dir, "src"))
    os.makedirs(os.path.join(project_dir, "tests"))

    files_to_create = {
        "main.go": '''package main

import "fmt"

func main() {
    fmt.Println("Hello, World!")
}

func calculate(x, y int) int {
    return x + y
}
''',
        "src/utils.go": '''package utils

import "strings"

func ToUpper(s string) string {
    return strings.ToUpper(s)
}

func Reverse(s string) string {
    runes := []rune(s)
    for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
        runes[i], runes[j] = runes[j], runes[i]
    }
    return string(runes)
}
''',
        "tests/utils_test.go": '''package utils

import "testing"

func TestToUpper(t *testing.T) {
    result := ToUpper("hello")
    if result != "HELLO" {
        t.Errorf("Expected HELLO, got %s", result)
    }
}
'''
    }

    for file_path, content in files_to_create.items():
        full_path = os.path.join(project_dir, file_path)
        os.makedirs(os.path.dirname(full_path), exist_ok=True)
        with open(full_path, "w") as f:
            f.write(content)

    return project_dir


def get_seed_data():
    """Get predefined seed data for testing."""
    return {
        "facts": {
            "test_preference": "dark_mode",
            "test_setting": "enabled",
            "user_name": "test_user",
            "app_version": "1.0.0"
        },
        "vectors": [
            "This is a test memory about Python programming",
            "Another memory about machine learning concepts",
            "Memory about web development best practices"
        ],
        "entities": [
            ("person_john", "John Doe", {"type": "person", "role": "developer"}),
            ("person_jane", "Jane Smith", {"type": "person", "role": "designer"}),
            ("company_acme", "ACME Corp", {"type": "company", "industry": "tech"})
        ],
        "relationships": [
            ("person_john", "company_acme", "works_at"),
            ("person_jane", "company_acme", "works_at"),
            ("person_john", "person_jane", "collaborates_with")
        ],
        "documents": {
            "docs/readme.md": "# Test Project\n\nThis is a test project for MCP testing.",
            "docs/api.md": "# API Documentation\n\n## Endpoints\n\n- GET /health\n- POST /data",
            "guides/setup.md": "# Setup Guide\n\n1. Install dependencies\n2. Run server\n3. Test connection"
        },
        "events": [
            ("test_user", "meeting", "Team standup meeting notes"),
            ("test_user", "development", "Working on new feature implementation"),
            ("test_user", "review", "Code review completed for PR #123")
        ]
    }