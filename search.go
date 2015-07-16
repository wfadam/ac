package main

import (
	"sort"
	"fmt"
	"regexp"
	"strings"
)

type ByValue []int 
var unfreqMap map[int][]string

func (v ByValue) Len() int {
	return len(v)
}

func (v ByValue) Swap(i,j int) {
	v[i], v[j] = v[j], v[i] 
}

func (v ByValue) Less(i,j int) bool {
	return v[i] > v[j] // from large to small
}

func topWeight( zipFileName string, allStreams chan []string) int {
	freqMap := make(map[string]int) 
	allKeys := getKeyWords( zipFileName )
	
	// map stream to key words hit count
	reduced := <-allStreams
	for _, key := range allKeys {
		part := filter( []byte(key), reduced )

		if weight(key)>1 && len(part)==0 { // when the keyword is a must-have one but it matches nothing
			return 0
		}

		if weight(key)>1 && len(part)>0 { // when the keyword is a must-have one and it matches, shrink the search space for next match
			reduced = part
		}

		for _, s := range part { // add weight value on trivial keywords
			freqMap[ s ] += weight( key )
		}
	}

	if len( freqMap ) == 0 { // no match against a series of keywords
		return 0
	}

	// map key words hit count to slice of streams
	unfreqMap = make(map[int][]string)
	for k,v := range freqMap {
		if v == 0 {
			continue
		}
		unfreqMap[ v ] = append( unfreqMap[ v ], k )
	}

	wt := []int{}
	for i := range unfreqMap {
		wt = append( wt, i )
	}

	sort.Sort( ByValue( wt ) )	

	topWt := wt[0]
	fmt.Printf("Top matching index %d / %d\n", topWt, sumWeight(allKeys))
	return topWt
}

func weight( s string ) int {
	caseIgnore := "(?i)"

	r, _ := regexp.Compile( caseIgnore + "(32|24|19|1Y|1X|1Z)nm|Bics" )
	if r.MatchString( s ) {
		return 10
	}

	r, _ = regexp.Compile( caseIgnore + "[124]P" )
	if r.MatchString( s ) {
		return 10
	}

	return 1
}

func sumWeight( arr []string ) int {
	tw := 0
	for _, s := range arr {
		tw += weight( s )
	}
	return tw
}

func getStreams( f int ) []string {
	return unfreqMap[ f ]
}

func getKeyWords( s string ) []string {
	r, e := regexp.Compile( "[^0-9a-zA-Z]|([1-9]+CE)" )
	if e != nil {
		fmt.Println( e )
		return []string{}
	}

	keywords := []string{}
	for _, k := range r.Split( s, -1) {
		if len(strings.TrimSpace( k ) ) == 0 {
			continue
		}
		keywords = append( keywords, k )
	}
	return keywords
}
