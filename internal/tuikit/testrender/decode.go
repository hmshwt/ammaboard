package testrender

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"
)

// ansiRe matches the escape vocabulary worth stripping: CSI sequences, OSC
// sequences, and two-byte escapes.
var ansiRe = regexp.MustCompile("\x1b\\[[0-9;?]*[A-Za-z]|\x1b\\][^\x07]*\x07|\x1b.")

// StripANSI removes every escape sequence, leaving the plain characters —
// the level-1 view of any rendered frame.
func StripANSI(s string) string {
	return ansiRe.ReplaceAllString(s, "")
}

// Style is the full style state a cell can carry — the kit's emitted
// vocabulary is foreground, background, and bold, nothing else. Palette keys
// are Styles.
type Style struct {
	Fg, Bg             [3]int
	HasFg, HasBg, Bold bool
}

// Palette maps every style the rendered frame may carry to a stable Attr
// slot. The host registers its theme here; an unregistered style is a decode
// error, never a silent unknown.
type Palette map[Style]Attr

// Decode turns rendered frame content into a cell grid. Escape sequences are
// interpreted (SGR applied, the rest skipped), styles resolve through the
// palette, and rows are padded to the widest line.
func Decode(s string, pal Palette) (Grid, error) {
	lines := strings.Split(s, "\n")
	width := 0
	for _, l := range lines {
		if n := utf8.RuneCountInString(StripANSI(l)); n > width {
			width = n
		}
	}
	if width == 0 {
		return Grid{}, nil
	}
	var cells []Cell
	var unknown map[Style]bool
	st := Style{}
	for _, line := range lines {
		row := make([]Cell, 0, width)
		i := 0
		for i < len(line) {
			if line[i] == 0x1b {
				params, sgr, n := parseSeq(line[i:])
				if sgr {
					st.apply(params)
				}
				i += n
				continue
			}
			r, size := utf8.DecodeRuneInString(line[i:])
			a, ok := pal[st]
			if !ok {
				if unknown == nil {
					unknown = map[Style]bool{}
				}
				unknown[st] = true
				a = Unknown
			}
			row = append(row, Cell{R: r, A: a})
			i += size
		}
		for len(row) < width {
			row = append(row, Cell{R: ' ', A: Normal})
		}
		cells = append(cells, row[:width]...)
	}
	g := Grid{W: width, H: len(lines), Cells: cells}
	if len(unknown) > 0 {
		keys := make([]string, 0, len(unknown))
		for k := range unknown {
			keys = append(keys, fmt.Sprintf("%+v", k))
		}
		sort.Strings(keys)
		return g, fmt.Errorf("unknown styles — register them in the palette: %s", strings.Join(keys, "; "))
	}
	return g, nil
}

// parseSeq decodes one escape sequence at the start of s: it returns the SGR
// parameters (for 'm' sequences), whether the sequence was SGR at all, and
// the number of bytes consumed.
func parseSeq(s string) (params []string, sgr bool, n int) {
	if len(s) < 2 {
		return nil, false, 1
	}
	if s[1] != '[' {
		if s[1] == ']' { // OSC … BEL (or unterminated: drop the rest)
			if i := strings.IndexByte(s, 0x07); i >= 0 {
				return nil, false, i + 1
			}
			return nil, false, len(s)
		}
		return nil, false, 2 // two-byte escape
	}
	i := 2
	for i < len(s) && s[i] >= 0x20 && s[i] <= 0x3f { // parameter and intermediate bytes
		i++
	}
	if i >= len(s) || s[i] < 0x40 || s[i] > 0x7e {
		return nil, false, i // unterminated: drop what we walked
	}
	if s[i] != 'm' {
		return nil, false, i + 1
	}
	return strings.Split(s[2:i], ";"), true, i + 1
}

// apply folds one SGR sequence into the style state. Unrecognized codes are
// ignored — the vocabulary is the kit's, and it is small.
func (st *Style) apply(params []string) {
	for i := 0; i < len(params); i++ {
		p := params[i]
		if p == "" {
			p = "0"
		}
		switch p {
		case "0":
			*st = Style{}
		case "1":
			st.Bold = true
		case "38", "48":
			if i+4 < len(params) && params[i+1] == "2" {
				rgb := [3]int{atoi(params[i+2]), atoi(params[i+3]), atoi(params[i+4])}
				if p == "38" {
					st.Fg, st.HasFg = rgb, true
				} else {
					st.Bg, st.HasBg = rgb, true
				}
				i += 4
			}
		}
	}
}

func atoi(s string) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return n
}
