package core

import (
	"fmt"
	"regexp"
	"strconv"
)

var pointerRE = regexp.MustCompile(`(?m)^version kk-lfs-1\.0\.0\r?\noid sha256:([a-f0-9]{64})\r?\nsize ([0-9]+)\r?\n?$`)

func FormatPointer(p Pointer) string {
	return fmt.Sprintf("version %s\noid sha256:%s\nsize %d\n", PointerVersion, p.OID, p.Size)
}

func ParsePointerText(s string) (Pointer, bool) {
	m := pointerRE.FindStringSubmatch(s)
	if len(m) != 3 {
		return Pointer{}, false
	}
	size, err := strconv.ParseInt(m[2], 10, 64)
	if err != nil {
		return Pointer{}, false
	}
	return Pointer{OID: m[1], Size: size}, true
}

func ParsePointerBytes(b []byte) (Pointer, bool) {
	return ParsePointerText(string(b))
}
