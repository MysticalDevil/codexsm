#!/usr/bin/env python3
"""Generate deterministic random Codex session .jsonl files.

Usage:
  python3 scripts/gen_seeded_sessions.py --seed 20260308 --count 50 \
    --time-range-start 2026-03-01T00:00:00Z \
    --time-range-end 2026-03-31T23:59:59Z \
    --output-root ./tmp/sessions

Output:
  Writes session files under <output-root>/YYYY/MM/DD/*.jsonl with session_meta and
  response_item lines. The same seed and args always generate the same content.
"""

from __future__ import annotations

import argparse
import json
import random
from datetime import UTC, datetime, timedelta
from pathlib import Path


HOSTS = [
    "/workspace/proj-alpha",
    "/workspace/proj-beta",
    "/workspace/proj-gamma",
    "/srv/codex/mono-repo",
]

USER_PROMPTS = [
    "请帮我实现会话恢复功能",
    "please add retry logic for scanner",
    "por favor corrige el analisis de sesiones",
    "salve quaeso sessiones refice",
    "セッション一覧の表示を最適化してください",
    "세션 스캐너 성능을 개선해 주세요",
    "يرجى تحسين فحص الجلسات",
    "请修复 list command gracias 日本語も対応 مرحبا",
    "fix flaky test 😄🔥 in scanner",
    "add pagination support for the list command",
    "请增加 --head-width 的边界测试",
    "optimize ScanSessions for large directories",
    "por favor agrega pruebas de mezcla emoji 😅🚀",
    "日本語とEnglishが混在するヘッド抽出を直して",
    "세션 필터에서 대소문자 무시 검색을 추가해 주세요",
    "يرجى إضافة وضع dry-run أكثر وضوحا",
    "latin quaestio: quomodo session id ex filename legitur",
    "need help with sorting by updated_at desc",
    "请检查 corrupted 文件的健康状态判定",
    "añade soporte para exportar csv con columnas personalizadas",
    "テストカバレッジを 80% 以上にしたい",
    "unicode normalization issue in matcher",
    "请把多语言关键字检索做成组合过滤",
    "can we benchmark filter performance with 100k sessions",
    "세션 삭제 전 미리보기 출력 형식을 개선해 주세요",
    "هل يمكن دعم استعادة حسب batch_id",
    "introduce deterministic fixtures for integration tests",
    "请把错误提示文案更具体一些",
    "por favor revisa el flujo de confirmacion interactiva",
    "emoji-only prompt 😄😄😄 should still be searchable",
    "日本語ヘッドの省略表示が崩れます",
    "mixed text: 修复 restore bug por favor 🙏",
    "need robust parsing when json line has extra spaces",
]

ASSISTANT_PROMPTS = [
    "I will inspect scanner and selector code paths first.",
    "我会先补充单元测试再验证覆盖率。",
    "Podemos agregar pruebas para mezcla multilingue y emoji.",
    "了解しました。まず失敗ケースを再現します。",
    "네, 대화 길이가 긴 케이스도 추가하겠습니다.",
    "سأضيف سيناريوهات خاصة ثم اشغل الاختبارات.",
    "I can add deterministic fixtures so results are reproducible across runs.",
    "我会先写一个最小失败用例，再做针对性修复。",
    "Primero validare el comportamiento actual y despues ajustamos la logica.",
    "了解です。境界値ケースから順に確認します。",
    "좋습니다. 성능 회귀가 없는지도 함께 확인하겠습니다.",
    "سأتحقق من التوافق مع تنسيق JSONL الحالي.",
    "I suggest splitting parsing and scoring tests for clearer failure signals.",
    "我会把多语言场景拆成独立子测试，便于定位问题。",
    "Podemos medir impacto con benchmark antes y despues del cambio.",
    "この変更は既存のCLIフラグ互換性を維持します。",
    "필요하면 selector 매칭 로직에 추가 테스트를 넣겠습니다.",
    "سأضيف رسائل خطأ أوضح في حالات الادخال غير الصحيح.",
    "I will keep the patch minimal and avoid touching unrelated files.",
    "我会优先保证 dry-run 和 confirm 的安全语义不变。",
    "Podemos incluir casos con arabe, coreano y japones en la misma conversacion.",
    "日本語ヘッドの切り詰め表示も合わせて確認します。",
    "네, emoji 혼합 문자열 검색도 회귀 테스트에 포함하겠습니다.",
    "سأشغل اختبار الوحدة الاصغر ذي الصلة بعد التعديل.",
    "I can also wire this generator into justfile if you want a shortcut target.",
    "我会确保相同 seed 下输出完全一致。",
    "Podemos extender los prompts sin cambiar el formato de salida.",
    "必要なら start-time を固定して时系列数据を再現します。",
    "성능이 걱정되면 생성 턴 수 범위를 조정할 수 있습니다.",
    "يمكننا اضافة سيناريوهات فساد متعمد للملفات لاحقا.",
    "I will provide command examples for quick local smoke checks.",
]


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Generate deterministic random Codex session files.")
    parser.add_argument("--output-root", default="tmp/generated-sessions", help="output root directory")
    parser.add_argument("--seed", type=int, required=True, help="RNG seed for deterministic output")
    parser.add_argument("--count", type=int, default=20, help="number of session files to generate")
    parser.add_argument("--min-turns", type=int, default=40, help="minimum conversation turns per session")
    parser.add_argument("--max-turns", type=int, default=240, help="maximum conversation turns per session")
    parser.add_argument(
        "--start-time",
        default="2026-03-01T00:00:00Z",
        help="legacy base RFC3339 timestamp in UTC; used when --time-range-start is omitted",
    )
    parser.add_argument(
        "--time-range-start",
        default="",
        help="RFC3339 UTC start timestamp for randomized created_at range",
    )
    parser.add_argument(
        "--time-range-end",
        default="",
        help="RFC3339 UTC end timestamp for randomized created_at range",
    )
    return parser.parse_args()


def parse_start_time(raw: str) -> datetime:
    value = raw.strip()
    if value.endswith("Z"):
        value = value[:-1] + "+00:00"
    dt = datetime.fromisoformat(value)
    if dt.tzinfo is None:
        dt = dt.replace(tzinfo=UTC)
    return dt.astimezone(UTC)


def make_session_id(rng: random.Random) -> str:
    h = f"{rng.getrandbits(128):032x}"
    return f"{h[:8]}-{h[8:12]}-{h[12:16]}-{h[16:20]}-{h[20:]}"


def random_time_in_range(rng: random.Random, start: datetime, end: datetime) -> datetime:
    if end < start:
        raise ValueError("time range end must be >= start")
    start_ts = int(start.timestamp())
    end_ts = int(end.timestamp())
    picked = rng.randint(start_ts, end_ts)
    return datetime.fromtimestamp(picked, tz=UTC)


def write_json_line(fp, payload: dict) -> None:
    fp.write(json.dumps(payload, ensure_ascii=False, separators=(",", ":")))
    fp.write("\n")


def generate_one_session(
    out_root: Path, rng: random.Random, range_start: datetime, range_end: datetime, min_turns: int, max_turns: int
) -> Path:
    session_id = make_session_id(rng)
    created = random_time_in_range(rng, range_start, range_end)
    day_dir = out_root / created.strftime("%Y") / created.strftime("%m") / created.strftime("%d")
    day_dir.mkdir(parents=True, exist_ok=True)

    file_name = f"rollout-{created.strftime('%Y-%m-%dT%H-%M-%S')}-{session_id}.jsonl"
    file_path = day_dir / file_name
    turns = rng.randint(min_turns, max_turns)

    with file_path.open("w", encoding="utf-8", newline="\n") as fp:
        write_json_line(
            fp,
            {
                "timestamp": created.isoformat().replace("+00:00", "Z"),
                "type": "session_meta",
                "payload": {
                    "id": session_id,
                    "timestamp": created.isoformat().replace("+00:00", "Z"),
                    "cwd": rng.choice(HOSTS),
                },
            },
        )

        for turn in range(turns):
            role = "user" if turn % 2 == 0 else "assistant"
            text = rng.choice(USER_PROMPTS if role == "user" else ASSISTANT_PROMPTS)
            if rng.random() < 0.25:
                text = f"{text} #{turn:03d}"
            write_json_line(
                fp,
                {
                    "timestamp": (created + timedelta(seconds=turn * 9)).isoformat().replace("+00:00", "Z"),
                    "type": "response_item",
                    "payload": {
                        "type": "message",
                        "role": role,
                        "content": [{"type": "input_text", "text": text}],
                    },
                },
            )

    return file_path


def main() -> int:
    args = parse_args()
    if args.count <= 0:
        raise SystemExit("--count must be > 0")
    if args.min_turns <= 0 or args.max_turns <= 0:
        raise SystemExit("--min-turns and --max-turns must be > 0")
    if args.min_turns > args.max_turns:
        raise SystemExit("--min-turns cannot be greater than --max-turns")

    rng = random.Random(args.seed)
    range_start = parse_start_time(args.time_range_start) if args.time_range_start.strip() else parse_start_time(args.start_time)
    if args.time_range_end.strip():
        range_end = parse_start_time(args.time_range_end)
    else:
        range_end = range_start + timedelta(days=30)
    if range_end < range_start:
        raise SystemExit("--time-range-end cannot be earlier than --time-range-start")

    out_root = Path(args.output_root).expanduser().resolve()
    out_root.mkdir(parents=True, exist_ok=True)

    generated: list[Path] = []
    for _ in range(args.count):
        generated.append(generate_one_session(out_root, rng, range_start, range_end, args.min_turns, args.max_turns))

    print(
        "generated="
        f"{len(generated)} seed={args.seed} "
        f"time_range={range_start.isoformat().replace('+00:00', 'Z')}..{range_end.isoformat().replace('+00:00', 'Z')} "
        f"root={out_root}"
    )
    for p in generated[:5]:
        print(f"sample={p}")
    if len(generated) > 5:
        print("sample=...")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
