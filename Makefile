# SPDX-FileCopyrightText: 2026 Marat Zimnurov <zimtir@mail.ru>
# SPDX-License-Identifier: BSD-2-Clause
#
# flang-env: проверка, примеры, печать в Go и в C.
#
# Компилятор flang — ОДИН двоичный файл, которому нужен только `cc`:
#
#   brew install flang        asdf plugin add flang        make -C bootstrap
#
# FLANG=путь/к/flang переопределяет команду; FLANG_HOME указывает на клон flang
# с собранным семенем.
ifdef FLANG_HOME
FLANG ?= $(FLANG_HOME)/bootstrap/flang
else
FLANG ?= flang
endif

МОДУЛИ = env

# Имя модуля Go у всех восьми целей одно — «flangprogram». Модуль здесь
# один, но правило то же, что у соседнего репозитория: после печати имя
# модуля переписывается на «flang<имя>». Это ЕДИНСТВЕННАЯ правка напечатанного,
# и делает её `sed`, а не рука.
.PHONY: all проверка печать план лицензии чисто

all: проверка печать план лицензии

проверка:
	@for m in $(МОДУЛИ); do \
	  $(FLANG) check flang/$$m.flang || exit 1; \
	  $(FLANG) test flang/$$m.flang || exit 1; \
	done
	$(FLANG) check tools/licensing.flang
	$(FLANG) test tools/licensing.flang

печать:
	rm -rf out-go out-c
	@for m in $(МОДУЛИ); do \
	  $(FLANG) emit flang/$$m.flang --target go --out out-go/$$m || exit 1; \
	  grep -rl flangprogram out-go/$$m | while read f; do sed -i.bak "s|flangprogram|flang$$m|g" "$$f" && rm -f "$$f.bak"; done; \
	  ( cd out-go/$$m && test -z "$$(gofmt -l .)" && go vet ./... && go build ./... ) || exit 1; \
	  $(FLANG) emit flang/$$m.flang --target c --out out-c/$$m || exit 1; \
	  $(MAKE) -C out-c/$$m || exit 1; \
	done

# План читает окружение сам и исполняется тем же двоичным. Печать его в Go и в
# C отказывает нарочно — цель «план» печатать не умеет, и об этом сказано в
# README дословной цитатой.
план:
	$(FLANG) check flang/read-env.flang
	$(FLANG) test flang/read-env.flang
	$(FLANG) io flang/read-env.flang

лицензии:
	cd tools && $(FLANG) io licensing.flang

чисто:
	rm -rf out-go out-c
