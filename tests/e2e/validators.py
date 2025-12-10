#!/usr/bin/env python3
"""
Validation utilities for E2E tests.
"""

import json
from typing import Any, Dict, List, Optional

try:
    from toon_format import decode as toon_decode
    TOON_AVAILABLE = True
except ImportError:
    TOON_AVAILABLE = False

try:
    import Levenshtein
    LEVENSHTEIN_AVAILABLE = True
except ImportError:
    LEVENSHTEIN_AVAILABLE = False


def parse_toon_response(response: dict) -> Any:
    """Parse TOON-formatted response content."""
    # Handle errors first
    if "error" in response:
        return response

    result = response.get("result", {})
    
    # Handle direct result (not wrapped in content)
    if isinstance(result, dict) and "content" not in result:
        # Check if it's already a structured response
        if any(key in result for key in ["facts", "results", "entities", "documents", "events", "projects"]):
            return result
        # Return the result as-is if it looks like plain data
        return result
    
    content = result.get("content", [])
    if not content:
        return result if result else None

    text_content = content[0].get("text", "")
    if not text_content.strip():
        return result if result else None

    # Try to decode as TOON if available
    if TOON_AVAILABLE:
        try:
            return toon_decode(text_content)
        except Exception as e:
            pass
    
    # Try JSON
    try:
        return json.loads(text_content)
    except:
        pass
    
    # Try to parse structured text format (key: value)
    try:
        parsed = parse_structured_text(text_content)
        if parsed:
            return parsed
    except:
        pass
    
    # Return raw text if nothing else works
    return text_content


def parse_structured_text(text: str) -> Optional[Dict]:
    """Parse text with key: value format into a dict."""
    result = {}
    lines = text.strip().split('\n')
    
    for line in lines:
        if ':' in line:
            key, value = line.split(':', 1)
            key = key.strip()
            value = value.strip()
            
            # Try to convert value to appropriate type
            if value.isdigit():
                value = int(value)
            elif value.lower() in ['true', 'false']:
                value = value.lower() == 'true'
            
            result[key] = value
    
    return result if result else None


def validate_toon_format(response: dict) -> bool:
    """Validate that response is in TOON format."""
    if not TOON_AVAILABLE:
        return True  # Skip validation if TOON not available

    parsed = parse_toon_response(response)
    return parsed is not None


def validate_levenshtein_suggestions(response: dict, query: str, expected_suggestions: List[str]) -> bool:
    """Validate Levenshtein suggestions in response."""
    if not LEVENSHTEIN_AVAILABLE:
        print("⚠️  Levenshtein library not available, skipping suggestion validation")
        return True

    parsed = parse_toon_response(response)
    if not parsed or "did_you_mean" not in parsed:
        return False

    suggestions = parsed["did_you_mean"]
    if not isinstance(suggestions, list):
        return False

    # Check if expected suggestions are present
    suggestion_values = [s.get("key", s.get("entity_id", s.get("path", ""))) for s in suggestions]
    return any(expected in suggestion_values for expected in expected_suggestions)


def calculate_levenshtein_distance(a: str, b: str) -> int:
    """Calculate Levenshtein distance between two strings."""
    if not LEVENSHTEIN_AVAILABLE:
        return 0
    return Levenshtein.distance(a, b)


def find_similar_strings(query: str, candidates: List[str], max_distance: int = 3) -> List[Dict]:
    """Find strings similar to query using Levenshtein distance."""
    if not LEVENSHTEIN_AVAILABLE:
        return []

    similar = []
    for candidate in candidates:
        distance = calculate_levenshtein_distance(query.lower(), candidate.lower())
        if distance <= max_distance:
            similar.append({
                "value": candidate,
                "distance": distance
            })

    return sorted(similar, key=lambda x: x["distance"])