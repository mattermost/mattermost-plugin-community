package main

import "sort"

func contains(list sort.StringSlice, name string) bool {
	for _, e := range list {
		if e == name {
			return true
		}
	}
	return false
}

func union(l1 sort.StringSlice, l2 sort.StringSlice) sort.StringSlice {
	for _, e := range l2 {
		if !contains(l1, e) {
			l1 = append(l1, e)
		}
	}
	return l1
}
