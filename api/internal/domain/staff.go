package domain

import "strings"

// WeakPIN reports PINs that are easy to guess: one repeated digit (1111), a straight run (1234, 9876),
// a repeated pair (1212) or a few very common choices. Staff choose their own PIN, so we refuse these.
func WeakPIN(pin string) bool {
	if len(pin) != 4 {
		return true
	}
	if strings.Count(pin, pin[:1]) == 4 {
		return true
	}
	up, down := true, true
	for i := 1; i < 4; i++ {
		if pin[i] != pin[i-1]+1 {
			up = false
		}
		if pin[i] != pin[i-1]-1 {
			down = false
		}
	}
	if up || down {
		return true
	}
	if pin[:2] == pin[2:] {
		return true
	}
	switch pin {
	case "2580", "0852", "1004", "2000", "1122", "6969", "1010":
		return true
	}
	return false
}
