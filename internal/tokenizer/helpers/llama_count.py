#!/usr/bin/env python3
import sys
import argparse

def main():
    argument_parser = argparse.ArgumentParser()
    argument_parser.add_argument("--spm-model", required=True, help="Path to SentencePiece tokenizer.model")
    parsed = argument_parser.parse_args()

    try:
        # pip install sentencepiece
        import sentencepiece as spm
    except Exception as import_error:
        sys.stderr.write(f"import error: {import_error}\n")
        sys.stderr.write("please install with: pip install sentencepiece\n")
        sys.exit(1)

    input_text = sys.stdin.read()
    processor = spm.SentencePieceProcessor()
    processor.Load(parsed.spm_model)
    token_ids = processor.EncodeAsIds(input_text)
    token_count = len(token_ids)
    sys.stdout.write(str(token_count))

if __name__ == "__main__":
    main()
