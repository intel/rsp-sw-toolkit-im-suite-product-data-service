package helper

import "strings"

func Contains(source []string, target string) bool {
	for _, item := range source {
		if item == target {
			return true
		}
	}

	return false
}

func ContainsWithInAny(source []string, target string) bool {
	for _, item := range source {
		if strings.Contains(item, target) {
			return true
		}
	}

	return false
}

func AreEqual(source []string, target []string) bool {

	for _, item := range source {
		if !Contains(target, item) {
			return false
		}
	}

	return true
}
