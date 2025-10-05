#!/usr/bin/env python3
import sys
import argparse

def main():
    argument_parser = argparse.ArgumentParser()
    argument_parser.add_argument("--model", required=False, help="Anthropic model name, e.g., claude-3-5-sonnet")
    parsed = argument_parser.parse_args()

    try:
        # PyPI package providing Claude-compatible tokenizer
        # pip install anthropic_tokenizer
        from anthropic_tokenizer import tokenize as anthropic_tokenize
    except Exception as import_error:
        sys.stderr.write(f"import error: {import_error}\n")
        sys.stderr.write("please install with: pip install anthropic_tokenizer\n")
        sys.exit(1)

    input_text = sys.stdin.read()
    token_ids = anthropic_tokenize(input_text)
    token_count = len(token_ids)
    sys.stdout.write(str(token_count))

if __name__ == "__main__":
    main()
