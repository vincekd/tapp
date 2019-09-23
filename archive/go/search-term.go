package tapp

import ("regexp")

type SearchTerm struct {
	Text string
	Upper string
	Quoted bool
	RegExp *regexp.Regexp
}
