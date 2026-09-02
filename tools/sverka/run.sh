#!/bin/sh
# SPDX-FileCopyrightText: 2026 Marat Zimnurov <zimtir@mail.ru>
# SPDX-License-Identifier: BSD-2-Clause
#
# Сверка слоя 1 flang-env с эталоном digitdisk на сетке входов.
#
# ЗАЧЕМ ОТДЕЛЬНЫЙ СКРИПТ. Сверщику нужно дотянуться до незаэкспортированного
# `detectDepth`, а в Go это возможно только ИЗНУТРИ его пакета. Поэтому файл
# отсюда кладётся во временный клон digitdisk, туда же через `go.mod`
# подставляется напечатанный в Go модуль «Env», и `go test` идёт там. В дерево
# digitdisk ничего не коммитится: клон одноразовый и лежит вне репозитория.
#
#   ./tools/sverka/run.sh
#
# Нужны: flang на PATH (или FLANG=путь), go, git, сеть для клона digitdisk.
# Клон берётся по DIGITDISK_REF (по умолчанию — закреплённый ниже отпечаток),
# чтобы числа в README сверялись с тем же деревом, на котором их сняли.
#
# ЧЕГО ЭТОТ ПРОГОН НЕ ПРОВЕРЯЕТ: разбор локали и таблицу правил чисел. Эталона
# у них нет вовсе — `fmt` в Go к языку нечувствителен, — и в ведомости README
# они стоят как «объявлено, не доказано». Прогон их и не трогает.
set -eu

FLANG="${FLANG:-flang}"
DIGITDISK_REF="${DIGITDISK_REF:-7ea03ed}"
DIGITDISK_URL="${DIGITDISK_URL:-https://github.com/digitable-lol/digitdisk.git}"

ROOT=$(cd "$(dirname "$0")/../.." && pwd)
WORK="${WORK:-${TMPDIR:-/tmp}/flang-env-sverka}"

echo "== клон digitdisk $DIGITDISK_REF в $WORK"
rm -rf "$WORK"
mkdir -p "$WORK"
git clone --quiet "$DIGITDISK_URL" "$WORK/digitdisk"
git -C "$WORK/digitdisk" checkout --quiet "$DIGITDISK_REF"
git -C "$WORK/digitdisk" --no-pager log --oneline -1

echo "== печать слоя 1 в Go"
# Печать пишет пояснения в stderr; они не беда, поэтому копятся в журнал и
# показываются только если печать отказала.
"$FLANG" emit "$ROOT/flang/env.flang" --target go --out "$WORK/go/env" >/dev/null 2>>"$WORK/emit.log" \
  || { cat "$WORK/emit.log" >&2; exit 1; }
grep -rl flangprogram "$WORK/go/env" | while read -r f; do
  sed -i.bak "s|flangprogram|flangenv|g" "$f" && rm -f "$f.bak"
done

echo "== подстановка модуля и сверщика в клон"
H="$WORK/digitdisk/host"
{
  echo
  echo "// СВЕРКА (одноразовый клон): библиотека flang-env."
  echo "require flangenv v0.0.0"
  echo "replace flangenv => $WORK/go/env"
} >> "$H/go.mod"
cp "$ROOT/tools/sverka/env_test.go" "$H/internal/ui/zzsverka_env_test.go"
( cd "$H" && go mod tidy >/dev/null 2>&1 )

echo
echo "== сверка"
( cd "$H" && go test -count=1 -run TestSverka -v ./internal/ui/ ) \
  | grep -E 'СВЕРКА|FAIL|^ok ' | sed 's/^ *zz[a-z_]*\.go:[0-9]*: //'

echo
echo "клон остался в $WORK — убрать: rm -rf $WORK"
