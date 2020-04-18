// Code generated by "stringer -type Status -linecomment"; DO NOT EDIT.

package runner

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[Queued-0]
	_ = x[Running-1]
	_ = x[Passed-2]
	_ = x[PassedWithFailures-3]
	_ = x[Failed-4]
	_ = x[Killed-5]
	_ = x[TimedOut-6]
}

const _Status_name = "queuedrunningpassedpassed_with_failuresfailedkilledtimed_out"

var _Status_index = [...]uint8{0, 6, 13, 19, 39, 45, 51, 60}

func (i Status) String() string {
	if i >= Status(len(_Status_index)-1) {
		return "Status(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _Status_name[_Status_index[i]:_Status_index[i+1]]
}