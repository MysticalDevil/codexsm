package migrate

import (
	"bufio"

	"github.com/MysticalDevil/codexsm/util"
)

const maxSessionMetaLineBytes = 1 << 20

type metaLine struct {
	Type    string `json:"type"`
	Payload struct {
		ID  string `json:"id"`
		Cwd string `json:"cwd"`
	} `json:"payload"`
}

func readBoundedLine(r *bufio.Reader, maxBytes int) (line []byte, truncated bool, err error) {
	return util.ReadBoundedLine(r, maxBytes)
}
