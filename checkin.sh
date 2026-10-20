#!/usr/bin/env bash
# checkin.sh — Check-in results and gateway status
#
# Runs cmd/signin directly for per-account results and token refresh.
# Reads config.json to check the gateway status before running check-ins.
# Verbose mode repeats failures at the end; ALREADY is not a failure.
#
# Usage:
#   ./checkin.sh            # Gateway status and check-in table
#   ./checkin.sh -v         # Also show failure details
#   ./checkin.sh auths_dir  # Use a different credentials directory
#
# Dependencies: Go when building, curl, and optionally jq.
# Without jq, the status header uses a basic availability message.
#
# Environment:
#   WB2A_CONFIG  Config path (default ./config.json; listen and api_key)
#   WB2A_URL     Gateway URL override, e.g. http://1.2.3.4:7863
#
# Scheduled gateway check-ins are independent of this manual utility.
#   WB2A_URL     直接指定服务地址（如 http://1.2.3.4:7863），设置后忽略配置文件
set -euo pipefail
cd "$(dirname "$0")"

VERBOSE=0
AUTHS_DIR="auths"
# 参数解析：-v/-h 与可选的 auths 目录。照搬 PR #48 的严格解析风格。
while [[ $# -gt 0 ]]; do
    case "$1" in
        -v|--verbose) VERBOSE=1; shift ;;
        -h|--help)
            sed -n '2,20p' "$0"
            exit 0 ;;
        -*) echo "Unknown argument: $1 (use -v / -h)" >&2; exit 2 ;;
        *) AUTHS_DIR="$1"; shift ;;
    esac
done

if ! command -v curl >/dev/null 2>&1; then
    echo "curl is required" >&2
    exit 1
fi
HAS_JQ=1
command -v jq >/dev/null 2>&1 || HAS_JQ=0

# ─── 解析服务地址与密钥：优先 WB2A_URL，其次 config.json ──────────────────────
CFG=${WB2A_CONFIG:-config.json}
PORT=7863
KEY=""
if [[ -f "$CFG" ]] && [[ "$HAS_JQ" == "1" ]]; then
    LISTEN=$(jq -r '.listen // ":7863"' "$CFG")
    PORT=${LISTEN##*:}
    KEY=$(jq -r '.api_key // ""' "$CFG")
fi
PORT=${PORT:-7863}
BASE=${WB2A_URL:-http://localhost:$PORT}

AUTH=()
[[ -n "$KEY" ]] && AUTH=(-H "Authorization: Bearer $KEY")

# ─── 网关状态头：GET /status 判活不死活（活着 = 定时签到已在网关侧运行）───────
# -s 静默，-w 单独取状态码；连接失败用 || true 兜住，CODE 归一为 000。
GATEWAY_OK=0
GATEWAY_INFO=""
RESP=$(curl -s -w $'\n%{http_code}' ${AUTH+"${AUTH[@]}"} "$BASE/status" || true)
if [[ -n "$RESP" ]]; then
    CODE=${RESP##*$'\n'}
    BODY=${RESP%$'\n'*}
    if [[ "$CODE" == "200" ]]; then
        GATEWAY_OK=1
        if [[ "$HAS_JQ" == "1" ]]; then
            GATEWAY_INFO=$(printf '%s' "$BODY" | jq -r \
                '"Gateway online: total=\(.total) healthy=\(.healthy) cooling=\(.cooling) disabled=\(.disabled)"' 2>/dev/null || true)
        fi
        [[ -z "$GATEWAY_INFO" ]] && GATEWAY_INFO="Gateway online (/status 200)"
    else
        case "$CODE" in
            401) GATEWAY_INFO="Gateway authentication failed (401): api_key does not match config.json" ;;
            000) GATEWAY_INFO="Cannot reach $BASE: service stopped or incorrect port" ;;
            *)   GATEWAY_INFO="Gateway /status error (HTTP $CODE)" ;;
        esac
    fi
else
    GATEWAY_INFO="Cannot reach $BASE: curl failed"
fi

echo "── Gateway status ─────────────────────────────"
echo "$GATEWAY_INFO"
if [[ "$GATEWAY_OK" == "1" ]]; then
    echo "Scheduled check-ins run in the gateway at 09:00/21:00. Running the standalone check-in tool now."
fi
echo

# ─── 核心：构建并执行 signin_bin（复用 signin.sh 的构建逻辑）──────────────────
BIN=./signin_bin
if [[ ! -x "$BIN" ]] || find . \( -name '*.go' -o -name 'go.mod' -o -name 'go.sum' \) -newer "$BIN" -print -quit | grep -q .; then
    echo "build signin_bin ..."
    if ! command -v go >/dev/null 2>&1; then
        echo "Go is required to build signin_bin (or prebuild it with ./signin.sh)" >&2
        exit 1
    fi
    go build -o "$BIN" ./cmd/signin
fi

# signin_bin 已是表格输出 + 汇总行，原样透传。-v 时先落盘再抓 fail 明细追加末尾。
if [[ "$VERBOSE" == "1" ]]; then
    OUT=$("$BIN" "$AUTHS_DIR") || true
    printf '%s\n' "$OUT"
    echo
    echo "── Failure details (excludes already checked in) ──"
    # signin_bin 表格行：uid | nick | status | remain | detail
    # 状态列左对齐填充（%-12s），故状态值后跟若干空格再 |。用 [[:space:]]* 容差。
    # 只挑 FAIL/LOAD_ERR/AUTH_INVALID 行（ALREADY 是幂等成功，不计）。
    printf '%s\n' "$OUT" | grep -E '\| (FAIL|LOAD_ERR|AUTH_INVALID)[[:space:]]*\|' || echo "  (no failures)"
else
    exec "$BIN" "$AUTHS_DIR"
fi
