
#!/usr/bin/env -S uv run
# /// script
# requires-python = ">=3.11"
# dependencies = [
#   "sentencepiece>=0",
#   "huggingface-hub>=0"
# ]
# ///

import argparse
import os
from pathlib import Path
import sys

try:
    import sentencepiece as spm  # type: ignore
    from huggingface_hub import hf_hub_download  # type: ignore
except Exception as import_error:  # pragma: no cover - network dependent
    sys.stderr.write(
        "uv runtime missing dependency for llama helper: "
        f"{import_error}
"
    )
    sys.stderr.write(
        "install with: uv pip install sentencepiece huggingface-hub
"
    )
    sys.exit(1)

DEFAULT_REPO = os.getenv("CTX_SPM_MODEL_REPO", "hf-internal-testing/llama-tokenizer")
DEFAULT_FILE = os.getenv("CTX_SPM_MODEL_FILE", "tokenizer.model")
DEFAULT_CACHE = Path(os.getenv("CTX_SPM_CACHE_DIR", Path.home() / ".cache/ctx/llama-tokenizer"))


def resolve_model_path(path_argument: str | None) -> Path:
    candidates = [path_argument, os.getenv("CTX_SPM_MODEL")]
    for candidate in candidates:
        if not candidate:
            continue
        resolved = Path(candidate).expanduser()
        if resolved.is_file():
            return resolved
        if candidate == path_argument:
            sys.stderr.write(
                f"provided --spm-model path {resolved} does not exist
"
            )
            sys.exit(1)

    try:
        DEFAULT_CACHE.mkdir(parents=True, exist_ok=True)
        downloaded = hf_hub_download(
            repo_id=os.getenv("CTX_SPM_MODEL_REPO", DEFAULT_REPO),
            filename=os.getenv("CTX_SPM_MODEL_FILE", DEFAULT_FILE),
            local_dir=str(DEFAULT_CACHE),
            local_dir_use_symlinks=False,
        )
        return Path(downloaded)
    except Exception as download_error:  # pragma: no cover
        sys.stderr.write(
            "failed to download SentencePiece model: "
            f"{download_error}
"
        )
        sys.stderr.write(
            "set CTX_SPM_MODEL to a local tokenizer.model or configure CTX_SPM_MODEL_REPO/FILE
"
        )
        sys.exit(1)


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument(
        "--spm-model",
        required=False,
        help="Path to SentencePiece tokenizer.model",
    )
    parser.add_argument(
        "--model",
        required=False,
        help="Requested model name (informational)",
    )
    args = parser.parse_args()

    model_path = resolve_model_path(args.spm_model)
    processor = spm.SentencePieceProcessor()
    processor.Load(str(model_path))

    input_text = sys.stdin.read()
    token_ids = processor.EncodeAsIds(input_text)
    sys.stdout.write(str(len(token_ids)))


if __name__ == "__main__":
    main()
