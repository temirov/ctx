#!/usr/bin/env -S uv run -qq
# /// script
# requires-python = ">=3.11"
# dependencies = [
#   "sentencepiece>=0",
#   "huggingface-hub>=0"
# ]
# ///

from pathlib import Path
import sys

try:
    import sentencepiece as spm  # type: ignore
    from huggingface_hub import hf_hub_download  # type: ignore
except Exception as import_error:  # pragma: no cover - network dependent
    sys.stderr.write(
        f"uv runtime missing dependency for llama helper: {import_error}\n"
    )
    sys.stderr.write(
        "install with: uv pip install sentencepiece huggingface-hub\n"
    )
    sys.exit(1)

DEFAULT_REPO = "hf-internal-testing/llama-tokenizer"
DEFAULT_FILE = "tokenizer.model"
DEFAULT_CACHE = Path.home() / ".cache" / "ctx" / "llama-tokenizer"


def resolve_model_path() -> Path:
    DEFAULT_CACHE.mkdir(parents=True, exist_ok=True)
    try:
        downloaded = hf_hub_download(
            repo_id=DEFAULT_REPO,
            filename=DEFAULT_FILE,
            local_dir=str(DEFAULT_CACHE),
        )
        return Path(downloaded)
    except Exception as download_error:  # pragma: no cover
        sys.stderr.write(
            f"failed to download SentencePiece model automatically: {download_error}\n"
        )
        sys.stderr.write(
            "install sentencepiece manually or place tokenizer.model at ~/.cache/ctx/llama-tokenizer\n"
        )
        sys.exit(1)


def main() -> None:
    model_path = resolve_model_path()
    processor = spm.SentencePieceProcessor()
    processor.Load(str(model_path))

    input_text = sys.stdin.read()
    token_ids = processor.EncodeAsIds(input_text)
    sys.stdout.write(f"{len(token_ids)}\n")


if __name__ == "__main__":
    main()
