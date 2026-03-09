#!/usr/bin/env python3
"""Validate doctor risk JSON output for smoke checks."""

from __future__ import annotations

import argparse
import json
from pathlib import Path


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Validate codexsm doctor risk JSON output.")
    parser.add_argument("path", help="path to the JSON payload")
    parser.add_argument("--sample-limit", type=int, required=True, help="expected sample_limit value")
    parser.add_argument("--min-high", type=int, default=1, help="minimum expected high-risk count")
    parser.add_argument("--min-risk-total", type=int, default=1, help="minimum expected risk_total count")
    return parser.parse_args()


def main() -> int:
    args = parse_args()
    payload = json.loads(Path(args.path).read_text(encoding="utf-8"))

    sessions_total = int(payload["sessions_total"])
    risk_total = int(payload["risk_total"])
    high = int(payload["high"])
    sample_limit = int(payload["sample_limit"])

    if sessions_total < risk_total:
        raise SystemExit(f"sessions_total {sessions_total} < risk_total {risk_total}")
    if risk_total < args.min_risk_total:
        raise SystemExit(f"risk_total {risk_total} < {args.min_risk_total}")
    if high < args.min_high:
        raise SystemExit(f"high {high} < {args.min_high}")
    if sample_limit != args.sample_limit:
        raise SystemExit(f"sample_limit {sample_limit} != {args.sample_limit}")

    print("doctor risk json payload validated")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
