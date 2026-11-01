package strictjson

import "testing"

func TestUnicode(t *testing.T) {
	for _, tc := range []struct {
		raw   string
		valid bool
	}{
		{`{"key":"\ud83d\ude00"}`, true}, {`{"key":"\ud83d"}`, false},
		{`{"key":"\ude00"}`, false}, {`{"key":"\ud83d\u0041"}`, false},
		{`{"key":"\\ud83d"}`, true}, {`{"key":"\u0000\r\n x "}`, true},
		{`{"\ud800":1}`, false}, {`{"key":"�"}`, true}, {`"unterminated`, false},
		{"\"\xff\"", false}, {`["a\\","b"]`, true},
	} {
		t.Run(tc.raw, func(t *testing.T) {
			if err := Validate([]byte(tc.raw)); (err == nil) != tc.valid {
				t.Fatalf("valid=%t error=%v", tc.valid, err)
			}
		})
	}
}
func FuzzValidate(f *testing.F) {
	for _, s := range []string{`"a"`, `"\ud800"`, `{"k":"\ud83d\ude00"}`} {
		f.Add([]byte(s))
	}
	f.Fuzz(func(t *testing.T, b []byte) { _ = Validate(b) })
}
