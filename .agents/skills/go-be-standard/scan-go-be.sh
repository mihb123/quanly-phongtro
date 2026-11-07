#!/usr/bin/env bash
set -uo pipefail
export LC_ALL=C

diff_mode=0
collect() {
  if [ $# -eq 0 ]; then
    { git diff --name-only --diff-filter=ACMR HEAD 2>/dev/null; git ls-files --others --exclude-standard 2>/dev/null; } | grep -E '\.go$'
  else
    for p in "$@"; do
      if [ -d "$p" ]; then find "$p" -name '*.go' -not -path '*/vendor/*'; else echo "$p"; fi
    done
  fi
}

[ $# -eq 0 ] && diff_mode=1
mapfile -t files < <(collect "$@" | grep -vE '(_test\.go|/mock[^/]*/|\.pb\.go|_gen\.go)$' | sort -u)
[ ${#files[@]} -eq 0 ] && { echo "go-be scan: không có file .go nào để soát"; exit 0; }

declare -A allowed
load_allowed() {
  allowed=()
  [ "$diff_mode" -eq 1 ] || return 0
  git ls-files --error-unmatch "$1" >/dev/null 2>&1 || return 0
  local n
  while read -r n; do allowed[$n]=1; done < <(
    git diff -U0 HEAD -- "$1" 2>/dev/null | awk '/^@@/ {
      split($3, a, ","); s = substr(a[1], 2); c = (a[2] == "") ? 1 : a[2]
      for (i = 0; i < c; i++) print s + i
    }')
  allowed[__tracked]=1
}

found=0
report() {
  local file=$1 rule=$2 msg=$3 pattern=$4 out line num
  out=$(grep -nE "$pattern" "$file" 2>/dev/null | grep -vE '^[0-9]+:\s*//') || return 0
  while IFS= read -r line; do
    num=${line%%:*}
    if [ -n "${allowed[__tracked]:-}" ] && [ -z "${allowed[$num]:-}" ]; then
      continue
    fi
    echo "$file:$num [$rule] $msg"
    echo "    ${line#*:}" | sed 's/^\s\+/    /'
    found=$((found + 1))
  done <<< "$out"
}

for f in "${files[@]}"; do
  [ -f "$f" ] || continue
  load_allowed "$f"
  layer=other
  case "$f" in
    */logger/*|*_adapter.go|*/cmd/*|*/pkg/*|*_cron.go|*/jobs/*) layer=other ;;
    */handler/httpx/*) layer=other ;;
    */handler/*|*_handler.go) layer=handler ;;
    */service/*|*_service.go) layer=service ;;
    */repository/*|*_repository.go) layer=repository ;;
  esac

  report "$f" SQL "ghép chuỗi SQL bằng Sprintf, dùng placeholder ? (cột sort thì whitelist)" \
    'fmt\.Sprintf\("[^"]*\b([Ss][Ee][Ll][Ee][Cc][Tt]|WHERE|where|ORDER BY|order by|INSERT|UPDATE|DELETE)\b'
  report "$f" ERRSTR "so sánh chuỗi lỗi, dùng errors.Is/errors.As (unique: *pq.Error Code 23505)" \
    '\.Error\(\)\s*(==|!=)|strings\.(Contains|HasPrefix)\(err\.Error\(\)'
  report "$f" MONEY "tiền dùng float: bảng/field MỚI dùng int64 VND; field cũ đang float thì giữ, không trộn" \
    '\b(Amount|Price|Total|Balance|Fee|Money|Rent|Cost)[A-Za-z]*\s+\*?float(32|64)\b'
  report "$f" OMITEMPTY "omitempty trên số tiền/số lượng làm mất giá trị 0 trong JSON" \
    '\b(Amount|Price|Total|Balance|Fee|Count|Quantity)[A-Za-z]*\s+\*?(u?int(8|16|32|64)?|float(32|64))\s+`json:"[^"]*,omitempty'
  report "$f" EN-MSG "message trả client bằng tiếng Anh, dùng tiếng Việt" \
    'Write(Error|JSON)\([^,]+,[^,]+,[^"]*"[A-Za-z][a-z]+( [a-z]+)*"\)'

  case "$layer" in
    handler)
      report "$f" H-QUERY "handler query DB trực tiếp, chuyển xuống repository" \
        '\.New(Select|Insert|Update|Delete|Raw)\(|\.(QueryContext|ExecContext|QueryRowContext)\(|\bdb\.(Query|Exec)'
      report "$f" H-LEAK "trả err.Error() cho client: dùng httpx.Fail (apperr ra message, lỗi hệ thống log + message chung)" \
        '(Write[A-Za-z]*|Error|JSON|Fail[A-Za-z]*)\(.*\berr\.Error\(\)'
      report "$f" H-DECODE "tự decode body/query, dùng httpx.Bind + request struct" \
        'json\.NewDecoder\(r\.Body\)|r\.URL\.Query\(\)\.Get\(|chi\.URLParam\('
      report "$f" CTX "context.Background/TODO trong luồng request, dùng r.Context() (goroutine nền: context.WithoutCancel + timeout)" \
        'context\.(Background|TODO)\(\)'
      report "$f" GO "goroutine trong handler: không dùng r.Context()/ResponseWriter sau khi handler trả về, có recover + timeout" \
        '\bgo func\('
      report "$f" PANIC "panic trong luồng request, trả error" '\bpanic\('
      ;;
    service)
      report "$f" S-HTTP "service biết HTTP, trả (T, error) và để handler ghi response" \
        'http\.ResponseWriter|\*http\.Request\b|net/http"|http\.Status[A-Z]'
      report "$f" CTX "context.Background/TODO trong service, nhận ctx từ tầng trên" \
        'context\.(Background|TODO)\(\)'
      report "$f" S-LOG "service log lỗi rồi trả lên = log 2 lần; chỉ httpx.Fail log (job nền/cron thì được)" \
        '\blog\.(Print|Printf|Println)\(|logger\.(Error|Warn)\('
      report "$f" PANIC "panic cho nghiệp vụ, trả apperr" '\bpanic\('
      ;;
    repository)
      report "$f" R-VALIDATE "repository parse/trim/validate input, việc này của request struct ở handler" \
        'strings\.TrimSpace|strconv\.(Atoi|Parse[A-Za-z]+)\(|time\.Parse(InLocation)?\('
      report "$f" R-DATE "lọc ngày đến 23:59:59, dùng [from, nextDay) với <" '23:59:59'
      report "$f" R-TX "query trực tiếp r.db: dùng database.Executor(ctx, r.db) để chạy được trong transaction" \
        'r\.db\.(New(Select|Insert|Update|Delete|Raw)|ExecContext|QueryContext)\('
      ;;
  esac
done

if [ "$found" -eq 0 ]; then
  echo "go-be scan: sạch (${#files[@]} file)"
  exit 0
fi
echo "go-be scan: $found chỗ cần xem lại trong ${#files[@]} file"
exit 1
