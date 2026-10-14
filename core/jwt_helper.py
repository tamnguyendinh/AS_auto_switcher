"""
Lightweight JWT payload decoder without third-party dependencies.
Decodes Base64URL encoded payload of Google id_token to extract account email and identifiers.
"""

import base64
import json
from typing import Optional, Dict, Any


def decode_jwt_payload(jwt_token: str) -> Optional[Dict[str, Any]]:
    """Decodes the payload (second segment) of a JWT token."""
    if not jwt_token or not isinstance(jwt_token, str):
        return None

    parts = jwt_token.strip().split(".")
    if len(parts) < 2:
        return None

    payload_segment = parts[1]
    # Replace URL-safe base64 characters
    payload_segment = payload_segment.replace("-", "+").replace("_", "/")

    # Add padding if needed
    rem = len(payload_segment) % 4
    if rem > 0:
        payload_segment += "=" * (4 - rem)

    try:
        raw_bytes = base64.b64decode(payload_segment)
        return json.loads(raw_bytes.decode("utf-8"))
    except Exception:
        return None


def extract_email_from_token_data(token_data: Dict[str, Any]) -> Optional[str]:
    """Extracts the email address from a token dictionary containing id_token."""
    if not token_data or not isinstance(token_data, dict):
        return None

    # Check root level email first
    if "email" in token_data and isinstance(token_data["email"], str):
        return token_data["email"].strip().lower()

    # Check id_token
    id_token = token_data.get("id_token")
    if id_token:
        payload = decode_jwt_payload(id_token)
        if payload and "email" in payload and isinstance(payload["email"], str):
            return payload["email"].strip().lower()

    # Sometimes token is nested in 'token'
    inner_token = token_data.get("token")
    if isinstance(inner_token, dict):
        return extract_email_from_token_data(inner_token)

    return None
