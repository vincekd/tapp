package tapp

import ("regexp")

type SearchTerm struct {
	Original string
	Text string
	Upper string
	Quoted bool
	RegExp *regexp.Regexp
}
