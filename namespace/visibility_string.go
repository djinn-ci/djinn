// Code generated by "stringer -type Visibility -linecomment"; DO NOT EDIT.

package namespace

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[Private-0]
	_ = x[Internal-1]
	_ = x[Public-2]
}

const _Visibility_name = "privateinternalpublic"

var _Visibility_index = [...]uint8{0, 7, 15, 21}

func (i Visibility) String() string {
	if i >= Visibility(len(_Visibility_index)-1) {
		return "Visibility(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _Visibility_name[_Visibility_index[i]:_Visibility_index[i+1]]
}
