<!-- SPDX-FileCopyrightText: 2026 Marat Zimnurov <zimtir@mail.ru> -->
<!-- SPDX-License-Identifier: BSD-2-Clause -->

# flang-env — parsing environment *values* in flang

[По-русски](README.ru.md)

Two layers, and the boundary between them is not taste — it is a property of
the language.

1. **Value parsing — pure, total functions.** The host hands over strings; the
   library decides how much colour the terminal will take, whether it can host
   a screen at all, what language to speak, and which characters to write
   numbers with. Emitted to **Go** and to **C**, so it travels inside somebody
   else's binary.
2. **A plan that reads the environment itself.** It is executed by `flang io` —
   the same compiler binary that builds with one `cc`. **No JavaScript is
   involved in execution, not one byte.**

The reference for layer 1 is
[digitdisk](https://github.com/digitable-lol/digitdisk),
`host/internal/ui/theme.go` and `term.go`. Diffed over a grid; the numbers
below are run output.

The source is written in flang, whose surface is Russian. File names are
English, module names are English, the prose inside is Russian — the language's
own convention.

---

## The boundary between layers: a quotation, not a claim

Layer 1 emits to all eight targets. Layer 2 does not, and the compiler says so
itself. A run on this repository's `flang/read-env.flang`:

```
$ flang emit flang/read-env.flang --target go --out /tmp/x
flang emit: печать отказала — FLANG_PLAN_UNSUPPORTED: цель «go» не умеет
печатать объявление «план». В программе оно есть: «Прочитать окружение».
Напечатать программу и промолчать про объявление нельзя: модуль собрался бы,
код возврата был бы нулевой, а исполнять план он бы не умел. Сегодня
объявление «план» печатает одна цель — «js»; исполнить его умеет ещё «flang io».
```

("Emission refused — FLANG_PLAN_UNSUPPORTED: the target «go» cannot emit a
`план` declaration. The program has one: «Прочитать окружение». Emitting the
program and staying silent about the declaration is not allowed: the module
would build, the exit code would be zero, and it would not be able to execute
the plan. Today one target emits a `план` — «js»; `flang io` can also execute
it.")

```
$ flang emit flang/read-env.flang --target c    --out /tmp/x   → same refusal, target «c»
$ flang emit flang/read-env.flang --target rust --out /tmp/x   → same refusal, target «rust»
$ flang emit flang/read-env.flang --target js   --out /tmp/x   → 2 files emitted, 145,099 bytes
```

Two conclusions, both load-bearing.

* **`js` is an EMIT TARGET, not the executor.** The plan is executed by
  `flang io`, a binary built by `cc` from `bootstrap/`. Node is needed neither
  to build it nor to run it.
* **Only layer 1 travels into emitted Go and C.** Whoever needs the library to
  read the environment itself calls `flang io`. Whoever needs the parsing to
  live inside their own binary takes the emitted «Env» module and hands it the
  strings. Both routes are real, and both are here.

### What calling `flang io` costs

Measured on this machine, 10–50 cold runs per row:

| What | Per run |
|---|---:|
| `flang io flang/read-env.flang` (six variables, the whole plan) | **1.08 s** |
| `flang check flang/read-env.flang` (the same program, checking only) | 1.17 s |
| `printenv TERM` | 4.0 ms |
| `/bin/true` (the fork/exec floor on this machine) | 1.5 ms |
| A Go program reading the same six variables | 24 ms |

The conclusion, as a number rather than an opinion: **`flang io` must not be
called from a hot loop** — it is 270× a single `printenv` and 700× the
fork/exec floor. Almost all of the cost is not reading but *checking*: before
executing, `flang io` judges the program the same way `flang check` does, which
the second row makes visible. `flang io` belongs at startup, at install time,
in CI, in a one-off report. Emitted layer 1 belongs everywhere else: it costs
nothing beyond an ordinary function call.

### Why the plan uses `printenv` and not the «Прочитать переменную среды» order

An order by that name **does exist** in the language's trunk — it is the
twenty-first of twenty-two (`flang/self/parser.flang`, `«Варианты поручения»`),
answered by «Значение среды» or «Переменной среды нет». But the released
compiler (the one `make -C bootstrap` builds, version 0.6.2) does not carry it
yet, and says so itself, verbatim:

```
FLANG_UNKNOWN_NAME: неизвестный вариант «Прочитать переменную среды»
```

Checked against the seed sources too: `getenv` appears seven times in
`bootstrap/flang_repl.c` (the file that executes plans), and not one of them
serves an order — all seven are about install paths and the compiler's own
variables.

So the plan asks the environment through the «Запустить процесс» order and the
`printenv` program: **exit code 0 — the variable is set, exit code 1 — it is
not**, and an empty value is told apart from an absent one by the *code*, not
by empty output. This matters: `NO_COLOR` acts by its mere presence
(no-color.org), and the reference asks about it with `os.LookupEnv`, not
`os.Getenv`. When the order reaches a release, two functions change here and
`printenv` leaves the machine.

---

## What layer 1 does

| What | Function | Reference |
|---|---|---|
| Colour depth from `NO_COLOR`, `TERM`, `COLORTERM` | `«Глубина цвета»` | `ui/theme.go` `detectDepth` |
| Whether the terminal can host a screen | `«Терминал годен»` | `ui/term.go` `UsableTERM` |
| Palette by name | `«Палитра по имени»` | `ui/theme.go` `PaletteByName` |
| Parsing `lang[_TERRITORY][.codeset][@modifier]` | `«Разобрать язык»` | none |
| `LC_ALL` → `LC_MESSAGES` → `LANG`, the POSIX order | `«Язык среды»` | none |
| Decimal point and digit separator by language | `«Правила языка»` | none |
| One-line summary (what the plan uses) | `«Сводка среды»` | none |

Plus the string plumbing without which none of it can be written: whitespace
trimming, ASCII lowercasing, splitting on a character.

**A variable is a sum, not a string.** `вариант «Задано» с значение равным …`
against `вариант «Не задано»`: an empty string and an absent variable are
different things, and the type knows it. `NO_COLOR` is exactly why.

## Getting it

```sh
brew install flang        # or: asdf plugin add flang
# or from a clone of the language:  make -C bootstrap
```

```sh
make проверка   # flang check + flang test
make печать     # emit layer 1 to Go and C, build both
make план       # check the plan and run it through flang io
make лицензии   # the licence guard
```

The plan, live:

```sh
$ TERM=xterm-256color COLORTERM=truecolor LANG=ru_RU.UTF-8 flang io flang/read-env.flang
{"plan":"Прочитать окружение",
 "result":"цвет: истинный цвет; язык: ru; область: RU; дробный: «,»; разрядный: « »",
 "orders":6, "log":[ … ]}

$ env -i PATH=/usr/bin:/bin flang io flang/read-env.flang
{"plan":"Прочитать окружение",
 "result":"цвет: без цвета; язык: ; область: ; дробный: «.»; разрядный: «»",
 "orders":6, "log":[ … ]}
```

A bare environment yields "no colour" — that is the right answer, not a
failure: an empty `TERM` means there is nothing to promise colour with.

---

## Diff against digitdisk

Clone of digitdisk at `7ea03ed` (0.5.0); the differ lives inside the
`internal/ui` package (there is no other way to reach the unexported
`detectDepth`) and, for every input, sets the process's real environment with
`t.Setenv`/`os.Unsetenv` while handing flang the same triple of values.

| Piece | Reference | Inputs | Divergences |
|---|---|---:|---:|
| Colour depth | `ui.detectDepth` | 384 | 0 |
| Terminal usable | `ui.UsableTERM` | 6 | 0 |
| Palette by name | `ui.PaletteByName` | 12 | 0 |
| **Total** | | **402** | **0** |

The colour-depth grid walks the presence *and* the value of `NO_COLOR`, eleven
`TERM` values (including `""`, `dumb`, `Dumb`, `256`, `direct`,
`xterm-256color`, `screen-256color`, `xterm-direct`) and seven `COLORTERM`
values (including `TrueColor` and `24BIT` — case matters), each in both the
"set" and the "unset" variety.

**Language parsing and number rules have NOTHING to diff against, and it is
said out loud.** Go's `fmt` is locale-blind: digitdisk writes `2.4 ГиБ` with a
dot, always, and never groups digits. The table below is ours, drawn from
glibc's locales, and the ledger lists it as **"declared, not proved"**; the
only thing that could check it is glibc itself, and no such diff was run.

* 36 languages write the fraction with a comma (`ru uk be kk de fr es it pt nl
  pl tr sv fi da nb nn cs sk hu bg ro el id vi lv lt et sl hr sr sq az hy ka
  mk`);
* 20 of those group digits with a space (`ru uk be kk pl fr cs sk sv fi nb nn
  hu bg lv lt et hy az ka`), the rest with a dot;
* everything else gets a dot and a comma, as in English;
* `C` and `POSIX` get a dot and *no* grouping at all — exactly what Go prints.

The space here is an ordinary U+0020; glibc uses a no-break or narrow no-break
space for several of these languages. Named, not glossed over.

---

## The ledger

**"proved"** — about *all* inputs. **"grid of N"** — computed on N of the
author's values; not a proof. **"declared, not proved"** — the runtime computes
it on whatever inputs arrive. Numbers are `flang check --proof` output.

| Module | Functions | All total | Claims | proved | grid | declared, not proved | Examples |
|---|---:|---|---:|---:|---:|---:|---:|
| Env (layer 1) | 25 | yes | 9 | 4 | 5 | 0 | 62 |
| ReadEnv (the plan) | 8 own (+25 imported) | yes | 4 | — | — | — | 14 own (76 with imports) |
| Licensing (the guard) | 21 | yes | 4 | — | — | — | 27 |

No ledger is printed for the last two, and the binary explains why: the program
declares a `план`, whose laws it does not judge, and an empty ledger section
would read as "no laws declared" — which is false. Their claims are checked by
examples and by running them, not by a ledger.

---

## Emission

| Target | Layer 1 | Layer 2 (the plan) |
|---|---|---|
| Go | emitted; `gofmt -l` empty, `go vet` clean, `go build` clean | **refused**, `FLANG_PLAN_UNSUPPORTED` |
| C | emitted; `cc -std=c99 -Wall -Wextra -Werror -pedantic -O2 -flto`, **0 warnings about our code** | **refused**, `FLANG_PLAN_UNSUPPORTED` |
| js | — | 2 files, 145,099 bytes |

`cc (Ubuntu 15.2.0-16ubuntu1) 15.2.0`, Linux, flang 0.6.2, `make all` exiting 0.
A caveat, so that "zero warnings" is not a lie: the linker prints its own
housekeeping line — `lto-wrapper: warning: using serial compilation of 2 LTRANS
jobs`; that is about parallelising LTO, not about our code, and `-Werror` would
not have let a diagnostic about the sources through.

```sh
echo '{"fn":"Сводка среды","args":[{"l":[{"s":"-"},{"s":"=xterm-256color"},{"s":"-"},{"s":"-"},{"s":"-"},{"s":"=ru_RU.UTF-8"}]}]}' | out-c/env/flang_cli
{"ok":true,"value":{"s":"цвет: 256; язык: ru; область: RU; дробный: «,»; разрядный: « »"}}
```

---

## Postconditions and their price

Postconditions are written here, and they are recomputed in emitted code. The
sibling repository
([flang-tui](https://github.com/digitable-lol/flang-tui), the speed section)
prices them: **5.3×** on a 200×50 frame — 55.9 ms against 10.1 ms. Environment
parsing happens once at startup, not on every repaint, so the price does not
burn here; but the rule is the same: a working build is emitted without
postconditions, and checking lives in `flang check` and `flang test`.

The emit flag that drops postconditions is being written in the language right
now; until it lands, the number was obtained by stripping `обеспечивает` from a
copy of the source with one line and emitting it with the same command.

## Licence

BSD-2-Clause (`LICENSE`; a Russian translation in `LICENSE-RU.md`). The tree is
written from scratch; layer 1's behaviour is diffed against digitdisk through
its open source, but not one line of anyone else's code is here. There is no
Python and no JavaScript in this tree — one of the checks in
`tools/licensing.flang`.
