package game

import (
	"fmt"
	"strings"
)

// stateLine — состояние игры в том же формате, что и дамп из оригинала.
func (g *Game) stateLine() string {
	b2i := func(b bool) int {
		if b {
			return 1
		}
		return 0
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "lvl=%d cols=%d cur=%d lives=%d score=%d goal=%d adv=%d bur=[%d %d %d %d] f=[%d %d]",
		g.level, g.cols, g.cur, g.lives, g.score, g.goal, g.adventure,
		b2i(g.burActive[0]), g.burPad[0], b2i(g.burActive[1]), g.burPad[1], g.burFrame[0], g.burFrame[1])
	sb.WriteString(" cnt=")
	for i := 0; i < g.npads; i++ {
		fmt.Fprintf(&sb, "%d,", g.counter[i])
	}
	sb.WriteString(" sz=")
	for i := 0; i < g.npads; i++ {
		fmt.Fprintf(&sb, "%d,", g.padSize[i])
	}
	sb.WriteString(" items=")
	for i := 0; i < g.npads; i++ {
		c := byte('.')
		switch {
		case g.goods[i]:
			c = 'g'
		case g.tax[i]:
			c = 't'
		case g.currency[i]:
			c = 'c'
		}
		sb.WriteByte(c)
	}
	return sb.String()
}
