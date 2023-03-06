package module

import "strings"

type Requires []*Module

func (requires Requires) Len() int {
	return len(requires)
}

func (requires Requires) Less(i, j int) (ok bool) {
	on := strings.Split(requires[i].Path, "/")
	tn := strings.Split(requires[j].Path, "/")
	n := len(on)
	if len(on) > len(tn) {
		n = len(tn)
	}
	x := 0
	for x = 0; x < n; x++ {
		if on[x] != tn[x] {
			break
		}
	}
	if x < n {
		ok = on[x] > tn[x]
	} else {
		ok = len(on) < len(tn)
	}
	return
}

func (requires Requires) Swap(i, j int) {
	requires[i], requires[j] = requires[j], requires[i]
	return
}
