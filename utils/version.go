package utils

import "github.com/hashicorp/go-version"

func IsVersionLessThan(ver, other string) bool {
	if ver == "" || ver == "0.0" || ver == "0.0.0" {
		return true
	}

	v, _ := version.NewVersion(ver)
	v2, _ := version.NewVersion(other)
	if v == nil || v2 == nil {
		return true
	}

	return v.LessThan(v2)
}
