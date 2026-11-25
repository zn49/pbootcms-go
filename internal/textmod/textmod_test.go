package textmod

import "testing"

func TestTruthHome(t *testing.T) {
	if !Truth("0==''") {
		t.Fatal("empty scode should match home")
	}
	if Truth("0=='1'") {
		t.Fatal("scode 1 is not home")
	}
	if !Truth("'1'=='1'") {
		t.Fatal("equal strings")
	}
	if !Truth("3>0") {
		t.Fatal("rows")
	}
	if Truth("") {
		t.Fatal("empty")
	}
	if !Truth("in_array('R102','R101,R102')") {
		t.Fatal("in_array hit")
	}
	if Truth("in_array('R109','R101,R102')") {
		t.Fatal("in_array miss")
	}
}

func TestCut(t *testing.T) {
	got := Apply("你好世界", "lencn=2", "")
	if got != "你好" {
		t.Fatalf("%s", got)
	}
}
