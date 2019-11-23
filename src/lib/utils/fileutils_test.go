package utils

import "testing"

var testSlashClean = []struct {
	Value  string
	Result string
}{
	// Already clean
	{"", "/"},
	{"/abc", "/abc"},
	{"/abc/def", "/abc/def"},
	{"/a/b/c", "/a/b/c"},
	{".", "/"},
	{"..", "/"},
	{"../..", "/"},
	{"../../abc", "/abc"},
	{"/abc", "/abc"},
	{"/", "/"},

	// Remove trailing slash
	{"abc/", "/abc"},
	{"abc/def/", "/abc/def"},
	{"a/b/c/", "/a/b/c"},
	{"./", "/"},
	{"../", "/"},
	{"../../", "/"},
	{"/abc/", "/abc"},

	// Remove doubled slash
	{"abc//def//ghi", "/abc/def/ghi"},
	{"//abc", "/abc"},
	{"///abc", "/abc"},
	{"//abc//", "/abc"},
	{"abc//", "/abc"},

	// Remove . elements
	{"abc/./def", "/abc/def"},
	{"/./abc/def", "/abc/def"},
	{"abc/.", "/abc"},

	// Remove .. elements
	{"abc/def/ghi/../jkl", "/abc/def/jkl"},
	{"abc/def/../ghi/../jkl", "/abc/jkl"},
	{"abc/def/..", "/abc"},
	{"abc/def/../..", "/"},
	{"/abc/def/../..", "/"},
	{"abc/def/../../..", "/"},
	{"/abc/def/../../..", "/"},
	{"abc/def/../../../ghi/jkl/../../../mno", "/mno"},

	// Combinations
	{"abc/./../def", "/def"},
	{"abc//./../def", "/def"},
	{"abc/../../././../def", "/def"},
}

func TestSlashClean(t *testing.T) {
	for _, test := range testSlashClean {
		val := SlashClean(test.Value)
		if val != test.Result {
			t.Errorf("Incorrect value on SlashClean for %v; want: %v; got: %v", test.Value, test.Result, val)
		}
	}
}
func BenchmarkGetOnExtension(b *testing.B) {
	for n := 0; n < b.N; n++ {
		GetFileType("f.jpg")
	}
}
