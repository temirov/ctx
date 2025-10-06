#!/usr/bin/env -S uv run
# /// script
# requires-python = ">=3.11"
# dependencies = [
#   "sentencepiece>=0"
# ]
# ///

import argparse
import sys


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument(
        "--spm-model",
        required=True,
        help="Path to SentencePiece tokenizer.model",
    )
    parsed = parser.parse_args()

    try:
        import sentencepiece as spm  # type: ignore
    except Exception as import_error:  # pragma: no cover - network dependent
        sys.stderr.write(
            "uv runtime missing dependency sentencepiece: "
            f"{import_error}\n"
        )
        sys.stderr.write("install with: uv pip install sentencepiece\n")
        sys.exit(1)

    processor = spm.SentencePieceProcessor()
    processor.Load(parsed.spm_model)

    input_text = sys.stdin.read()
    token_ids = processor.EncodeAsIds(input_text)
    sys.stdout.write(str(len(token_ids)))


if __name__ == "__main__":
    main()
