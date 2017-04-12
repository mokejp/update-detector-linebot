package updatedetector

import (
	"bytes"

	"github.com/sergi/go-diff/diffmatchpatch"
)

func deleteSlice(s []string, i int) []string {
	s = append(s[:i], s[i+1:]...)
	n := make([]string, len(s))
	copy(n, s)
	return n
}

func diffText(text1, text2 []byte) string {
	dmp := diffmatchpatch.New()
	a, b, c := dmp.DiffLinesToRunes(string(text1), string(text2))
	diffs := dmp.DiffMainRunes(a, b, false)
	result := dmp.DiffCharsToLines(diffs, c)
	return diffsToString(result)
}

func diffsToString(diffs []diffmatchpatch.Diff) string {
	var buf bytes.Buffer
	for _, diff := range diffs {
		text := diff.Text

		switch diff.Type {
		case diffmatchpatch.DiffInsert:
			_, _ = buf.WriteString("+")
			_, _ = buf.WriteString(text)
		case diffmatchpatch.DiffDelete:
			_, _ = buf.WriteString("-")
			_, _ = buf.WriteString(text)
		}
	}

	return buf.String()
}
