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

    # Build messages payload; count_tokens expects the same shape as messages.create
    messages_payload = [{"role": "user", "content": user_text}]
    system_prompt = parsed_args.system

    try:
        token_count = client.messages.count_tokens(
            model=parsed_args.model,
            messages=messages_payload,
            system=system_prompt,
        )
    except Exception as count_error:
        sys.stderr.write(f"Failed to count tokens via Anthropic API: {count_error}\n")
        sys.exit(1)

    # Print only the integer with a newline
    sys.stdout.write(f"{token_count.input_tokens}\n")

if __name__ == "__main__":
    main()
