// Code generated by "stringer -type Level -linecomment"; DO NOT EDIT.

package alert

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[Success-1]
	_ = x[Warn-2]
	_ = x[Danger-3]
}

const _Level_name = "successwarndanger"

var _Level_index = [...]uint8{0, 7, 11, 17}

func (i Level) String() string {
	i -= 1
	if i >= Level(len(_Level_index)-1) {
		return "Level(" + strconv.FormatInt(int64(i+1), 10) + ")"
	}
	return _Level_name[_Level_index[i]:_Level_index[i+1]]
}
