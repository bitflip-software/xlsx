package xlsx

import (
	"fmt"
	"testing"
)

func TestShParserBasics(t *testing.T) {
	tn := "TestShParserBasics"
	rawData := topen(Mac1621)
	zs, err := zopen(rawData)

	if err != nil {
		t.Errorf("received error from zopen: '%s'", err.Error())
		return
	}

	tbl, err := shparse(zs, 0)

	if err != nil {
		t.Errorf("received error from shparse: '%s'", err.Error())
		return
	}

	use(zs)
	use(tn)
	use(tbl)
}

func TestShFindRow(t *testing.T) {
	tn := "TestShFindRow"
	rawData := topen(Mac1621)
	zs, err := zopen(rawData)

	if err != nil {
		t.Errorf("received error from zopen: '%s'", err.Error())
		return
	}

	got := itos(len(zs.info.sheetMeta))
	stmt := "len(zs.info.sheetMeta"
	want := itos(3)
	if got != want {
		t.Errorf(tfail(tn, stmt, got, want))
		return // could panic if not correct
	}

	data, err := shload(zs.info.sheetMeta[0])

	if err != nil {
		t.Error(terr(tn, "data, err := shload(zs.info.sheetMeta[0])", err.Error()))
	}

	next := 0
	first := -1
	last := -1
	chunk := ""

	first, last = shfindRow(data, next)
	stmt = fmt.Sprintf("first, last = shfindRow(data, %d)", next)
	got = itos(first)
	want = itos(999)

	if got != want {
		t.Error(tfail(tn, stmt+"; first", got, want))
	}

	got = itos(last)
	want = itos(1273)

	if got != want {
		t.Error(tfail(tn, stmt+"; last", got, want))
	}

	if first < 0 || first >= len(data) || last < 0 || last >= len(data) || first > last {
		return // avoid a panic
	}

	chunk = string(data[first:last])
	if len(chunk) < 7 {
		return // avoid a panic
	}

	got = chunk[0:5]
	stmt = "chunk[0:5]"
	want = "<row>"

	if got != want {
		t.Error(tfail(tn, stmt, got, want))
	}

	got = chunk[len(chunk)-6:]
	stmt = "chunk[len(chunk)-6:]"
	want = "<row>"

	if got != want {
		t.Error(tfail(tn, stmt, got, want))
	}
}

func TestShAdvanceBad(t *testing.T) {
	tn := "TestShAdvanceBad"
	runes := ""
	start := -1
	r := '<'
	expected := -1
	actual := shadvance([]rune(runes), start, r)
	stmt := fmt.Sprintf("shadvance('%s', %d, '%s')", runes, start, string(r))
	got := itos(actual)
	want := itos(expected)

	if got != want {
		t.Error(tfail(tn, stmt, got, want))
	}
}

func TestShAdvanceGood(t *testing.T) {
	tn := "TestShAdvanceGood"
	runes := " ü Hello Günter"
	start := 2
	r := 'ü'
	expected := 10
	actual := shadvance([]rune(runes), start, r)
	stmt := fmt.Sprintf("shadvance('%s', %d, '%s')", runes, start, string(r))
	got := itos(actual)
	want := itos(expected)

	if got != want {
		t.Error(tfail(tn, stmt, got, want))
	}
}

func TestShAdvanceNotFound(t *testing.T) {
	tn := "TestShAdvanceGood"
	runes := " ü Hello Günter"
	start := 2
	r := 'x'
	expected := -1
	actual := shadvance([]rune(runes), start, r)
	stmt := fmt.Sprintf("shadvance('%s', %d, '%s')", runes, start, string(r))
	got := itos(actual)
	want := itos(expected)

	if got != want {
		t.Error(tfail(tn, stmt, got, want))
	}
}

func TestShBadA(t *testing.T) {
	tn := "TestShBadA"
	ix := -1
	runes := "abc"
	expected := true
	actual := shbad([]rune(runes), ix)
	stmt := fmt.Sprintf("shbad('%s', %d)", runes, ix)
	got := btos(actual)
	want := btos(expected)

	if got != want {
		t.Error(tfail(tn, stmt, got, want))
	}
}

func TestShBadB(t *testing.T) {
	tn := "TestShBadB"
	ix := 3
	runes := "abc"
	expected := true
	actual := shbad([]rune(runes), ix)
	stmt := fmt.Sprintf("shbad('%s', %d)", runes, ix)
	got := btos(actual)
	want := btos(expected)

	if got != want {
		t.Error(tfail(tn, stmt, got, want))
	}
}

func TestShBadC(t *testing.T) {
	tn := "TestShBadC"
	ix := 2
	runes := "abc"
	expected := false
	actual := shbad([]rune(runes), ix)
	stmt := fmt.Sprintf("shbad('%s', %d)", runes, ix)
	got := btos(actual)
	want := btos(expected)

	if got != want {
		t.Error(tfail(tn, stmt, got, want))
	}
}

func TestShTagStart(t *testing.T) {
	type input struct {
		xml      string
		tagName  string
		expectedOpen int
		expepectedClose int
	}

	inputs := []input{
		input{"sg07< bloopsgn<jk:bloop >dfsg978sg9<><><><SFG", "bloop", 15, 24}
	}
}