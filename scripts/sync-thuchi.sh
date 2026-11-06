#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."
src="${THUCHI_SRC:-$HOME/thu-chi}"
test -d "$src/pkg/thuchi" || { echo "Không thấy $src/pkg/thuchi (đặt THUCHI_SRC)"; exit 1; }

rsync -a --delete --exclude '*_test.go' "$src/pkg/thuchi/" pkg/thuchi/
grep -rl 'github.com/mihb123/thu-chi/' pkg/thuchi --include='*.go' | xargs -r sed -i 's#github.com/mihb123/thu-chi/#github.com/mihb123/quanly-phongtro/#g'
gofmt -l pkg/thuchi
echo "Đã đồng bộ pkg/thuchi từ $src. Schema đổi thì thêm migration mới (so với pkg/thuchi/sql/schema.sql)."
