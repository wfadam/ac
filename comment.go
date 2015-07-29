package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

func lastRevHist(modFiles []string) string {
	r, _ := regexp.Compile("\\./[1-9]+D([1-9]+CE)?/_Revision_TestPrg.txt\\b")

	var fName string
	for _, s := range modFiles {
		fName = r.FindString(s)
		if len(fName) > 0 {
			break
		}
	}

	if len(fName) == 0 {
		return "./xDxCE/_Revision_TestPrg.txt is not found or updated"
	}

	content, _ := Output("cat", fName)
	return lastMsg(content)
}

func lastMsg(s string) string {
	if len(s) == 0 {
		return ""
	}

	lines := strings.Split(s, "\n")

	ignCase := "(?i)"
	r, e := regexp.Compile(ignCase + "\\bREV.*\\bTCR.*[0-9.]+")
	checkError(e)

	ln := lastMatchLn(r, lines)
	if ln >= 0 {
		return strings.Join(lines[ln:], "\n")
	}

	return ""
}

func lastMatchLn(r *regexp.Regexp, lines []string) int {
	for i := len(lines) - 1; i >= 0; i-- {
		if r.MatchString(lines[i]) {
			return i
		}
	}
	return -1
}

func checkError(e error) {
	if e == nil {
		return
	}

	fmt.Println(e)
	os.Exit(1)
}
