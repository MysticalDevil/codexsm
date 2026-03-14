package scanner

import (
	"bufio"

	"github.com/MysticalDevil/codexsm/util"
)

const (
	maxSessionMetaLineBytes = 1 << 20
	maxSessionHeadLineBytes = 256 << 10
)

func readBoundedLine(r *bufio.Reader, maxBytes int) (line []byte, truncated bool, err error) {
	return util.ReadBoundedLine(r, maxBytes)
}
