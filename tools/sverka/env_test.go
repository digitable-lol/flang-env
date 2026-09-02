// Сверка flang-env с эталоном digitdisk. Кладётся внутрь пакета
// digitdisk/internal/ui: иначе не дотянуться до незаэкспортированного
// detectDepth. Запускается tools/sverka/run.sh из репозитория flang-env.
//
// SPDX-FileCopyrightText: 2026 Marat Zimnurov <zimtir@mail.ru>
// SPDX-License-Identifier: BSD-2-Clause
package ui

import (
	"fmt"
	"os"
	"testing"

	fe "flangenv/flang"
	rte "flangenv/flangrt"
)

type tallyEnv struct {
	name   string
	inputs int
	diffs  int
	first  string
}

func (t *tallyEnv) eq(in string, want, got string) {
	t.inputs++
	if want != got {
		t.diffs++
		if t.first == "" {
			t.first = fmt.Sprintf("вход %s: эталон %q, flang %q", in, want, got)
		}
	}
}

func (t *tallyEnv) strict(tb testing.TB) {
	if t.first != "" {
		tb.Logf("СВЕРКА %-34s входов %8d расхождений %6d  первое: %s", t.name, t.inputs, t.diffs, t.first)
	} else {
		tb.Logf("СВЕРКА %-34s входов %8d расхождений %6d", t.name, t.inputs, t.diffs)
	}
	if t.diffs != 0 {
		tb.Errorf("%s: расхождений %d из %d", t.name, t.diffs, t.inputs)
	}
}

// t.Setenv умеет только ставить; снятие переменной — здесь, с откатом.
func unsetEnvSverka(t *testing.T, name string) {
	t.Helper()
	old, had := os.LookupEnv(name)
	if err := os.Unsetenv(name); err != nil {
		t.Fatalf("не снять %s: %v", name, err)
	}
	t.Cleanup(func() {
		if had {
			_ = os.Setenv(name, old)
		} else {
			_ = os.Unsetenv(name)
		}
	})
}

// ───────────────────────────── Env ─────────────────────────────

func envVal(set bool, v string) rte.Value {
	if set {
		return fe.VariantZadano(rte.Text(v))
	}
	return fe.VariantNeZadano()
}

func TestSverkaDepth(t *testing.T) {
	ta := &tallyEnv{name: "Глубина цвета / detectDepth"}
	ctx := fe.NewContext()
	nameOf := func(d depth) string {
		switch d {
		case depthNone:
			return "без цвета"
		case depth16:
			return "16"
		case depth256:
			return "256"
		default:
			return "истинный цвет"
		}
	}
	terms := []string{"", "dumb", "xterm", "xterm-256color", "screen-256color", "xterm-direct", "linux", "vt100", "Dumb", "256", "direct"}
	colorterms := []string{"", "truecolor", "TrueColor", "24BIT", "24bit", "yes", "нет"}
	for _, noColorSet := range []bool{false, true} {
		for _, noColorVal := range []string{"", "1", "0"} {
			if !noColorSet && noColorVal != "" {
				continue
			}
			for _, term := range terms {
				for _, ct := range colorterms {
					for _, termSet := range []bool{false, true} {
						for _, ctSet := range []bool{false, true} {
							if !termSet && term != "" {
								continue
							}
							if !ctSet && ct != "" {
								continue
							}
							if noColorSet {
								t.Setenv("NO_COLOR", noColorVal)
							} else {
								t.Setenv("NO_COLOR", "")
								_ = ""
							}
							t.Setenv("TERM", term)
							t.Setenv("COLORTERM", ct)
							// t.Setenv не умеет «снять переменную», поэтому
							// снятое состояние проверяется отдельно ниже.
							if !noColorSet {
								unsetEnvSverka(t, "NO_COLOR")
							}
							if !termSet {
								unsetEnvSverka(t, "TERM")
							}
							if !ctSet {
								unsetEnvSverka(t, "COLORTERM")
							}
							v, err := fe.GlubinaCveta(ctx, envVal(noColorSet, noColorVal), envVal(termSet, term), envVal(ctSet, ct))
							if err != nil {
								t.Fatalf("flang: %v", err)
							}
							got, err := fe.NazvanieGlubiny(ctx, v)
							if err != nil {
								t.Fatalf("flang: %v", err)
							}
							ta.eq(fmt.Sprintf("NO_COLOR=%v/%q TERM=%v/%q COLORTERM=%v/%q", noColorSet, noColorVal, termSet, term, ctSet, ct),
								nameOf(detectDepth()), got.Str)
						}
					}
				}
			}
		}
	}
	ta.strict(t)
}

func TestSverkaUsableTERM(t *testing.T) {
	ta := &tallyEnv{name: "Терминал годен / UsableTERM"}
	ctx := fe.NewContext()
	for _, set := range []bool{false, true} {
		for _, term := range []string{"", "dumb", "xterm", "DUMB", " "} {
			if !set && term != "" {
				continue
			}
			t.Setenv("TERM", term)
			if !set {
				unsetEnvSverka(t, "TERM")
			}
			v, err := fe.TerminalGoden(ctx, envVal(set, term))
			if err != nil {
				t.Fatalf("flang: %v", err)
			}
			ta.eq(fmt.Sprintf("%v/%q", set, term), fmt.Sprint(UsableTERM()), fmt.Sprint(v.Flag))
		}
	}
	ta.strict(t)
}

func TestSverkaPalette(t *testing.T) {
	ta := &tallyEnv{name: "Палитра по имени / PaletteByName"}
	ctx := fe.NewContext()
	names := []string{"", " ", "paper", "Paper", " PAPER ", "signal", "SIGNAL", "carbon", "Carbon", "неон", "papers", "\tsignal\n"}
	nameOf := func(p Palette) string {
		switch p.Name {
		case Paper.Name:
			return "Бумага"
		case Signal.Name:
			return "Сигнал"
		default:
			return "Уголь"
		}
	}
	for _, n := range names {
		v, err := fe.PalitraPoImeni(ctx, rte.Text(n))
		if err != nil {
			t.Fatalf("flang: %v", err)
		}
		ta.eq(fmt.Sprintf("%q", n), nameOf(PaletteByName(n)), v.Str)
	}
	ta.strict(t)
}
