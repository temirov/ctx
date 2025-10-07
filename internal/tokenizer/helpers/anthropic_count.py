#!/usr/bin/env -S uv run -qq
# /// script
# requires-python = ">=3.11"
# dependencies = [
#   "anthropic>=0"
# ]
# ///

import argparse
import os
import sys
from typing import List

def main() -> None:
    argument_parser = argparse.ArgumentParser(description="Count tokens for Anthropic Claude models.")
    argument_parser.add_argument(
        "--model",
        default="claude-3-5-sonnet-20241022",
        help="Anthropic model name (default: claude-3-5-sonnet-20241022).",
    )
    argument_parser.add_argument(
        "--system",
        default=None,
        help="Optional system prompt to include in counting.",
    )
    parsed_args = argument_parser.parse_args()

    try:
        from anthropic import Anthropic  # type: ignore
    except Exception as import_error:
        sys.stderr.write(f"Missing dependency 'anthropic': {import_error}\n")
        sys.stderr.write("Install with: uv pip install anthropic\n")
        sys.exit(1)

    api_key = os.getenv("ANTHROPIC_API_KEY")
    if not api_key:
        sys.stderr.write("ANTHROPIC_API_KEY is not set in the environment.\n")
        sys.exit(1)

    client = Anthropic(api_key=api_key)

    user_text = sys.stdin.read()

    messages_payload = build_user_messages(user_text)
    system_prompt = parsed_args.system

    request_args = {
        "model": parsed_args.model,
        "messages": messages_payload,
    }
    if system_prompt:
        request_args["system"] = system_prompt

    try:
        token_count = client.messages.count_tokens(**request_args)
    except Exception as count_error:
        handle_count_error(client, parsed_args.model, count_error)

    # Print only the integer with a newline
    sys.stdout.write(f"{token_count.input_tokens}\n")


def build_user_messages(user_text: str) -> List[dict]:
    """Construct Claude-compatible user message payload."""
    return [
        {
            "role": "user",
            "content": [
                {
                    "type": "text",
                    "text": user_text,
                }
            ],
        }
    ]


def handle_count_error(client, requested_model: str, count_error: Exception) -> None:
    """Emit a helpful error and exit, adding model suggestions when available."""
    try:
        from anthropic import APIStatusError, NotFoundError  # type: ignore
    except Exception:  # anthropic < 0.18 compatibility
        APIStatusError = NotFoundError = tuple()  # type: ignore

    if isinstance(count_error, NotFoundError):  # pragma: no cover - network dependent
        suggestions = discover_claude_models(client)
        sys.stderr.write(
            f"Model {requested_model!r} was not found by the Anthropic API.\n"
        )
        if suggestions:
            sys.stderr.write(
                "Available Claude models: " + ", ".join(suggestions) + "\n"
            )
        sys.exit(1)

    if APIStatusError and isinstance(count_error, APIStatusError):  # pragma: no cover
        sys.stderr.write(
            f"Failed to count tokens via Anthropic API: {count_error}\n"
        )
        sys.exit(1)

    sys.stderr.write(
        f"Failed to count tokens via Anthropic API: {count_error}\n"
    )
    sys.exit(1)


def discover_claude_models(client) -> List[str]:
    """Return a short, sorted list of Claude models for better guidance."""
    try:
        models = client.models.list()
    except Exception:  # pragma: no cover - network dependent
        return []

    names: List[str] = []
    for model_info in getattr(models, "data", []) or []:
        model_id = getattr(model_info, "id", None)
        if isinstance(model_id, str) and model_id.startswith("claude-"):
            names.append(model_id)

    names.sort()
    return names[:10]

if __name__ == "__main__":
    main()
