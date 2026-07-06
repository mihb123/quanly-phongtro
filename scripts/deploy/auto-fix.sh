#!/usr/bin/env bash
# auto-fix.sh — Khi deploy thất bại, nhờ một AI coding agent có sẵn trên self-hosted
# server tự phân tích log, sửa lỗi, rồi TỰ commit + push để kích hoạt deploy lại.
# Thứ tự fallback: claude -> agy -> codex. Agent ĐẦU TIÊN chạy xong (exit 0) thì dừng.
#
# An toàn:
#   - Chống vòng lặp: nếu commit đang deploy đã là commit auto-fix (có marker AUTOFIX_MARKER)
#     thì KHÔNG sửa tiếp, chỉ báo cần can thiệp tay.
#   - Commit do agent tạo được đánh dấu rõ (author bot + marker trong subject) để user nhận ra.
#   - Chỉ commit đúng những file agent thay đổi (so snapshot working tree trước/sau).
#
# Biến môi trường tùy chọn:
#   PROJECT_DIR             — thư mục dự án (đích deploy, repo git). Mặc định = repo root của script.
#   DEPLOY_LOG_FILE         — file log bước deploy bị lỗi. Mặc định /tmp/deploy_step.log.
#   DEPLOY_AUTOFIX_SUMMARY  — file agent ghi tóm tắt nguyên nhân. Mặc định /tmp/deploy_autofix_summary.md.
#   AUTOFIX_TIMEOUT         — timeout mỗi agent, định dạng của `timeout`. Mặc định 15m.
#   AUTOFIX_PUSH_URL        — URL remote để push (ép về SSH vì actions/checkout đặt origin là https).
#                             Mặc định git@github.com:mihb123/quanly-phongtro.git.
#
# Cố ý KHÔNG bật `set -e`: cần bắt exit code của từng agent/git để fallback thủ công.
set -uo pipefail

# --- 1. CONFIGURATION ---

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

FIX_DIR="${PROJECT_DIR:-$(cd "$SCRIPT_DIR/../.." && pwd)}"
LOG_FILE="${DEPLOY_LOG_FILE:-/tmp/deploy_step.log}"
SUMMARY_FILE="${DEPLOY_AUTOFIX_SUMMARY:-/tmp/deploy_autofix_summary.md}"
AGENT_TIMEOUT="${AUTOFIX_TIMEOUT:-15m}"
FAIL_REASON=""  # lý do lỗi chính xác, do commit_and_push_fix điền.

# Marker nhận diện commit do auto-fix tạo; dùng cho cả loop-guard lẫn subject commit.
AUTOFIX_MARKER="[auto-fix]"
BOT_NAME="deploy-autofix[bot]"
BOT_EMAIL="deploy-autofix@quanly-phongtro.local"
TARGET_BRANCH="develop"
# actions/checkout đặt origin về https+token (push bằng token KHÔNG trigger workflow),
# nên ép origin về SSH để push kích hoạt được deploy lại.
PUSH_REMOTE_URL="${AUTOFIX_PUSH_URL:-git@github.com:mihb123/quanly-phongtro.git}"

# --- 2. HELPERS & PROMPT ---

# Gửi thông báo Telegram, bỏ qua lỗi để không làm hỏng luồng fallback.
notify() {
  PROJECT_DIR="$FIX_DIR" bash "$FIX_DIR/scripts/notify-telegram.sh" "$1" 2>/dev/null || true
}

# Chụp danh sách file đang thay đổi trong working tree (sort theo LC_ALL=C để khớp comm).
snapshot_worktree() {
  git -C "$FIX_DIR" status --porcelain --untracked-files=all | cut -c4- | LC_ALL=C sort -u
}

# Dựng yêu cầu cho agent: log lỗi, nhiệm vụ, và yêu cầu ghi tóm tắt ra file.
build_prompt() {
  local log_excerpt
  log_excerpt="$(tail -n 200 "$LOG_FILE" 2>/dev/null || echo '(không đọc được log)')"
  cat <<EOF
Quy trình deploy CI/CD của dự án vừa THẤT BẠI trên self-hosted server.
Thư mục dự án: $FIX_DIR

Log của bước deploy bị lỗi:
--- BẮT ĐẦU LOG ---
$log_excerpt
--- KẾT THÚC LOG ---

Nhiệm vụ:
1. Phân tích log để tìm nguyên nhân gốc của lỗi (build / lint / test / migration ...).
2. Sửa trực tiếp source trong repo để bước deploy này chạy qua được.
3. Giữ thay đổi tối thiểu, chỉ chạm những file liên quan tới lỗi.
4. Ghi một bản tóm tắt NGẮN GỌN ra đúng file: $SUMMARY_FILE
   Gồm: (a) nguyên nhân gốc 1-3 câu; (b) từng file đã sửa và lý do ngắn.

Ràng buộc BẮT BUỘC:
- KHÔNG chạy git commit / git push / git reset / git clean.
  Script deploy sẽ tự commit & push bản sửa của bạn — bạn chỉ cần sửa file và ghi tóm tắt.
EOF
}

# --- 3. AGENT RUNNERS ---

# Nhờ Claude Code sửa lỗi (ưu tiên đầu tiên).
try_claude() {
  command -v claude >/dev/null 2>&1 || { echo "claude: không tìm thấy trên server"; return 127; }
  timeout "$AGENT_TIMEOUT" claude -p "$PROMPT" \
    --add-dir "$FIX_DIR" \
    --dangerously-skip-permissions
}

# Fallback sang Antigravity (Gemini).
try_agy() {
  command -v agy >/dev/null 2>&1 || { echo "agy: không tìm thấy trên server"; return 127; }
  timeout "$AGENT_TIMEOUT" agy -p "$PROMPT" \
    --add-dir "$FIX_DIR" \
    --dangerously-skip-permissions \
    --print-timeout "$AGENT_TIMEOUT"
}

# Fallback cuối cùng sang Codex CLI.
try_codex() {
  command -v codex >/dev/null 2>&1 || { echo "codex: không tìm thấy trên server"; return 127; }
  timeout "$AGENT_TIMEOUT" codex exec "$PROMPT" \
    --dangerously-bypass-approvals-and-sandbox
}

# --- 4. COMMIT & PUSH ---

# Commit đúng những file agent đã sửa rồi push lên develop để trigger deploy lại.
# Trả về 0 nếu đã push; !=0 nếu lỗi — kèm lý do CHÍNH XÁC trong biến toàn cục FAIL_REASON.
commit_and_push_fix() {
  local agent="$1"
  shift
  local changed_files=("$@")
  local git_err

  if [[ ${#changed_files[@]} -eq 0 ]]; then
    FAIL_REASON="Agent '$agent' chạy xong nhưng KHÔNG chỉnh sửa file nào (git không thấy thay đổi trong $FIX_DIR)."
    return 1
  fi

  if ! git_err="$(git -C "$FIX_DIR" add -- "${changed_files[@]}" 2>&1)"; then
    FAIL_REASON="git add thất bại:
$git_err"
    return 1
  fi

  local subject="🤖 auto-fix(deploy): sửa lỗi deploy bằng $agent $AUTOFIX_MARKER"
  local root_cause
  root_cause="$(cat "$SUMMARY_FILE" 2>/dev/null || echo '(agent không cung cấp tóm tắt)')"

  if ! git_err="$(git -C "$FIX_DIR" \
    -c "user.name=$BOT_NAME" \
    -c "user.email=$BOT_EMAIL" \
    commit --no-verify -m "$subject" -m "$root_cause" 2>&1)"; then
    FAIL_REASON="git commit thất bại:
$git_err"
    return 1
  fi

  # Ép origin về SSH: actions/checkout đặt lại origin về https+token mỗi lần chạy.
  if ! git_err="$(git -C "$FIX_DIR" remote set-url origin "$PUSH_REMOTE_URL" 2>&1)"; then
    FAIL_REASON="git remote set-url origin '$PUSH_REMOTE_URL' thất bại:
$git_err"
    return 1
  fi

  if ! git_err="$(git -C "$FIX_DIR" push origin "HEAD:$TARGET_BRANCH" 2>&1)"; then
    FAIL_REASON="git push lên nhánh '$TARGET_BRANCH' thất bại (remote: $(git -C "$FIX_DIR" remote get-url origin 2>/dev/null)):
$git_err"
    return 1
  fi

  return 0
}

# --- 5. EXECUTION FLOW ---

cd "$FIX_DIR" || { echo "Không vào được thư mục dự án: $FIX_DIR"; exit 1; }

# Phải là git repo mới có thể commit/push bản sửa.
if ! git -C "$FIX_DIR" rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  notify "⚠️ Deploy thất bại nhưng $FIX_DIR không phải git repo — không thể tự sửa & push."
  echo "Không phải git repo: $FIX_DIR"
  exit 1
fi

# Loop-guard: nếu commit đang deploy đã do auto-fix tạo mà vẫn lỗi thì dừng, tránh vòng lặp.
if git -C "$FIX_DIR" log -1 --pretty=%s 2>/dev/null | grep -qF "$AUTOFIX_MARKER"; then
  notify "⚠️ Commit đang deploy là bản auto-fix trước đó nhưng deploy VẪN lỗi. Dừng tự sửa để tránh vòng lặp — cần can thiệp tay."
  echo "Loop-guard: commit trước là auto-fix, dừng."
  exit 1
fi

rm -f "$SUMMARY_FILE"
PROMPT="$(build_prompt)"
BEFORE_SNAPSHOT="$(snapshot_worktree)"
AGENT_LOG="$(mktemp -t autofix-agent-XXXXXX.log)"
AGENT_ERRORS=()  # gom lý do thất bại của từng agent để báo cuối cùng.

notify "🔧 Deploy thất bại — đang thử tự động sửa lỗi bằng AI agent (claude → agy → codex)..."

# Chạy lần lượt các agent theo thứ tự fallback; dừng ngay khi có agent chạy xong.
for agent in claude agy codex; do
  echo "===== Thử agent: $agent ====="

  # Lưu output agent ra log để trích lý do lỗi chính xác; vẫn hiện trên log workflow.
  "try_$agent" 2>&1 | tee "$AGENT_LOG"
  status=${PIPESTATUS[0]}

  if [[ "$status" -ne 0 ]]; then
    reason="exit $status"
    [[ "$status" -eq 124 ]] && reason="hết thời gian (timeout $AGENT_TIMEOUT)"
    [[ "$status" -eq 127 ]] && reason="không cài trên server"
    AGENT_ERRORS+=("• $agent ($reason): $(tail -n 3 "$AGENT_LOG" | tr '\n' ' ')")
    echo "Agent '$agent' thất bại ($reason), thử agent kế tiếp..."
    continue
  fi

  # Tách đúng file agent vừa sửa so với baseline trước khi chạy (LC_ALL=C khớp collation).
  mapfile -t CHANGED_FILES < <(LC_ALL=C comm -13 \
    <(printf '%s\n' "$BEFORE_SNAPSHOT") \
    <(snapshot_worktree) | sed '/^$/d')

  if ! commit_and_push_fix "$agent" "${CHANGED_FILES[@]}"; then
    notify "🤖 '$agent' đã chạy xong nhưng KHÔNG hoàn tất auto-fix.

❌ Lý do chính xác:
$FAIL_REASON

📁 Thư mục: $FIX_DIR
Hãy kiểm tra & xử lý thủ công."
    echo "$FAIL_REASON"
    exit 1
  fi

  ROOT_CAUSE="$(cat "$SUMMARY_FILE" 2>/dev/null || echo '(agent không cung cấp tóm tắt)')"
  FILE_LIST="$(printf -- '- %s\n' "${CHANGED_FILES[@]}")"
  COMMIT_HASH="$(git -C "$FIX_DIR" rev-parse --short HEAD)"

  notify "📦 Auto-fix deploy bằng '$agent' — ĐÃ commit & push lên $TARGET_BRANCH, deploy sẽ chạy lại.

🔎 Nguyên nhân & thay đổi:
$ROOT_CAUSE

📝 File đã sửa:
$FILE_LIST
🔗 Commit: $COMMIT_HASH"
  echo "Agent '$agent' hoàn tất, đã push $COMMIT_HASH."
  exit 0
done

notify "⚠️ Cả claude, agy và codex đều KHÔNG sửa được lỗi deploy. Cần can thiệp thủ công.

❌ Lý do từng agent:
$(printf '%s\n' "${AGENT_ERRORS[@]}")"
echo "Tất cả agent đều thất bại."
exit 1
