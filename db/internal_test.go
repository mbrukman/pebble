// Copyright 2012 The LevelDB-Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package db

import (
	"strconv"
	"strings"
	"testing"
)

func (k InternalKey) encodedString() string {
	buf := make([]byte, k.Size())
	k.Encode(buf)
	return string(buf)
}

func TestInternalKey(t *testing.T) {
	k := MakeInternalKey([]byte("foo"), 0x08070605040302, 1)
	if got, want := k.encodedString(), "foo\x01\x02\x03\x04\x05\x06\x07\x08"; got != want {
		t.Fatalf("k = %q want %q", got, want)
	}
	if !k.Valid() {
		t.Fatalf("invalid key")
	}
	if got, want := string(k.UserKey), "foo"; got != want {
		t.Errorf("ukey = %q want %q", got, want)
	}
	if got, want := k.Kind(), InternalKeyKind(1); got != want {
		t.Errorf("kind = %d want %d", got, want)
	}
	if got, want := k.SeqNum(), uint64(0x08070605040302); got != want {
		t.Errorf("seqNum = %d want %d", got, want)
	}
}

func TestInvalidInternalKey(t *testing.T) {
	testCases := []string{
		"",
		"\x01\x02\x03\x04\x05\x06\x07",
		"foo",
		"foo\x08\x07\x06\x05\x04\x03\x02",
		"foo\x12\x07\x06\x05\x04\x03\x02\x01",
	}
	for _, tc := range testCases {
		k := DecodeInternalKey([]byte(tc))
		if k.Valid() {
			t.Errorf("%q is a valid key, want invalid", tc)
		}
	}
}

func TestInternalKeyComparer(t *testing.T) {
	// keys are some internal keys, in sorted order.
	keys := []string{
		// The empty key is not a valid internal key, but it still must
		// sort lower than any other key. It is used as a zero value when
		// checking that a sequence of internal keys are in sorted order.
		"",
		// The next two keys are also invalid internal keys. They are 'less
		// than' any valid internal key, and 'greater than' the empty key.
		"A",
		"B",
		// The remaining test keys are all valid.
		"" + "\x01\xff\xff\xff\xff\xff\xff\xff",
		"" + "\x00\xff\xff\xff\xff\xff\xff\xff",
		"" + "\x01\x01\x00\x00\x00\x00\x00\x00",
		"" + "\x00\x01\x00\x00\x00\x00\x00\x00",
		"" + "\x01\x00\x00\x00\x00\x00\x00\x00",
		"" + "\x00\x00\x00\x00\x00\x00\x00\x00",
		"\x00" + "\x00\x00\x00\x00\x00\x00\x00\x00",
		"\x00blue" + "\x01\x11\x00\x00\x00\x00\x00\x00",
		"bl\x00ue" + "\x01\x11\x00\x00\x00\x00\x00\x00",
		"blue" + "\x01\x11\x00\x00\x00\x00\x00\x00",
		"blue\x00" + "\x01\x11\x00\x00\x00\x00\x00\x00",
		"green" + "\x01\x11\x00\x00\x00\x00\x00\x00",
		"red" + "\x01\xff\xff\xff\xff\xff\xff\xff",
		"red" + "\x01\x72\x73\x74\x75\x76\x77\x78",
		"red" + "\x01\x00\x00\x00\x00\x00\x00\x11",
		"red" + "\x01\x00\x00\x00\x00\x00\x11\x00",
		"red" + "\x01\x00\x00\x00\x00\x11\x00\x00",
		"red" + "\x01\x00\x00\x00\x11\x00\x00\x00",
		"red" + "\x01\x00\x00\x11\x00\x00\x00\x00",
		"red" + "\x01\x00\x11\x00\x00\x00\x00\x00",
		"red" + "\x01\x11\x00\x00\x00\x00\x00\x00",
		"red" + "\x00\x11\x00\x00\x00\x00\x00\x00",
		"red" + "\x00\x00\x00\x00\x00\x00\x00\x00",
		"\xfe" + "\x01\xff\xff\xff\xff\xff\xff\xff",
		"\xfe" + "\x00\x00\x00\x00\x00\x00\x00\x00",
		"\xff" + "\x01\xff\xff\xff\xff\xff\xff\xff",
		"\xff" + "\x00\x00\x00\x00\x00\x00\x00\x00",
		"\xff\x40" + "\x01\xff\xff\xff\xff\xff\xff\xff",
		"\xff\x40" + "\x00\x00\x00\x00\x00\x00\x00\x00",
		"\xff\xff" + "\x01\xff\xff\xff\xff\xff\xff\xff",
		"\xff\xff" + "\x00\x00\x00\x00\x00\x00\x00\x00",
	}
	c := DefaultComparer.Compare
	for i := range keys {
		for j := range keys {
			ik := DecodeInternalKey([]byte(keys[i]))
			jk := DecodeInternalKey([]byte(keys[j]))
			got := InternalCompare(c, ik, jk)
			want := 0
			if i < j {
				want = -1
			} else if i > j {
				want = +1
			}
			if got != want {
				t.Errorf("i=%d, j=%d, keys[i]=%q, keys[j]=%q: got %d, want %d",
					i, j, keys[i], keys[j], got, want)
			}
		}
	}
}

func TestInternalKeySeparator(t *testing.T) {
	var makeIkeyKinds = map[string]InternalKeyKind{
		"DEL": InternalKeyKindDelete,
		"MAX": InternalKeyKindMax,
		"SET": InternalKeyKindSet,
	}

	makeIkey := func(s string) InternalKey {
		x := strings.Split(s, ".")
		ukey := x[0]
		kind := makeIkeyKinds[x[1]]
		seqNum, _ := strconv.ParseUint(x[2], 10, 64)
		return MakeInternalKey([]byte(ukey), seqNum, kind)
	}

	testCases := []struct {
		a        string
		b        string
		expected string
	}{
		{"foo.SET.100", "foo.SET.99", "foo.SET.100"},
		{"foo.SET.100", "foo.SET.100", "foo.SET.100"},
		{"foo.SET.100", "foo.DEL.100", "foo.SET.100"},
		{"foo.SET.100", "foo.SET.101", "foo.SET.100"},
		{"foo.SET.100", "bar.SET.99", "foo.SET.100"},
		{"foo.SET.100", "hello.SET.200", "g.MAX.72057594037927935"},
		{"ABC1AAAAA.SET.100", "ABC2ABB.SET.200", "ABC2.MAX.72057594037927935"},
		{"AAA1AAA.SET.100", "AAA2AA.SET.200", "AAA2.MAX.72057594037927935"},
		{"AAA1AAA.SET.100", "AAA4.SET.200", "AAA2.MAX.72057594037927935"},
		{"AAA1AAA.SET.100", "AAA2.SET.200", "AAA1B.MAX.72057594037927935"},
		{"AAA1AAA.SET.100", "AAA2A.SET.200", "AAA2.MAX.72057594037927935"},
		{"AAA1.SET.100", "AAA2.SET.200", "AAA1.SET.100"},
		{"foo.SET.100", "foobar.SET.200", "foo.SET.100"},
		{"foobar.SET.100", "foo.SET.200", "foobar.SET.100"},
	}
	d := DefaultComparer
	for _, c := range testCases {
		t.Run("", func(t *testing.T) {
			a := makeIkey(c.a)
			b := makeIkey(c.b)
			expected := makeIkey(c.expected)
			result := a.Separator(d.Compare, d.Separator, nil, b)
			if cmp := InternalCompare(d.Compare, expected, result); cmp != 0 {
				t.Fatalf("expected %s, but found %s", expected, result)
			}
		})
	}
}
