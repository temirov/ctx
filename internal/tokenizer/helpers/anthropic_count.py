
#!/usr/bin/env -S uv run
# /// script
# requires-python = ">=3.11"
# dependencies = [
#   "anthropic-tokenizer>=0"
# ]
# ///

import argparse
import sys


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument(
        "--model",
        required=False,
        help="Anthropic model name, e.g., claude-3-5-sonnet",
    )
    parser.parse_args()

    try:
        from anthropic_tokenizer import tokenize as anthropic_tokenize  # type: ignore
    except Exception as import_error:  # pragma: no cover - network dependent
        sys.stderr.write(
            "uv runtime missing dependency anthropic_tokenizer: "
            f"{import_error}
"
        )
        sys.stderr.write(
            "install with: uv pip install anthropic-tokenizer
"
        )
        sys.exit(1)

    input_text = sys.stdin.read()
    token_ids = anthropic_tokenize(input_text)
    sys.stdout.write(str(len(token_ids)))


if __name__ == "__main__":
    main()
