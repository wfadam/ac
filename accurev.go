package main

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
)

var sel []string

func main() {
	// disable input buffering
	exec.Command("stty", "-F", "/dev/tty", "cbreak", "min", "1").Run()
	// do not display entered characters on the screen
	exec.Command("stty", "-F", "/dev/tty", "-echo").Run()

	allStreams := []string{
		"T73_1Znm_128gb_ABL_eX3_2P_SDSIP_768_16D",
		"T73_1Ynm_64gb_ABL_eX3_1P_TSOP_768_1D",
		"T73_1Znm_64gb_ABL_eX2_4P_SSD-BGA_768_8D",
	}

	done := make(chan bool)
	ch := make(chan string)
	go inputProName(ch, done)
	go search(ch, allStreams )

	<-done

	baseStream := setStream()
	fmt.Printf( "\n%s\n", baseStream )

}

func getStream(ch <-chan string) {
	fmt.Printf("\nBase Stream is %s\n", <-ch)
}

func search(ch <-chan string, allStreams []string ) {
	caseIgnore := "(?i)"
	for {
		pat := <-ch
		if len(pat) == 0 {
			continue
		}

		r, err := regexp.Compile(caseIgnore + pat)
		if err != nil {
			fmt.Println(err)
		}

		sel = []string{}
		for _, s := range allStreams {
			if r.MatchString(s) {
				sel = append(sel, s)
			}
		}

		fmt.Println()
		for i, s := range sel {
			if r.MatchString(s) {
				fmt.Printf("\t%d : %s\n", i, s)
			}
		}
		fmt.Println("Press <Enter> to proceed")
	}
}

func setStream() string {
	if len( sel ) == 0 {
		fmt.Printf("\nNothing is selected, so quit\n" )
		os.Exit(1)
	}

	exec.Command("stty", "-F", "/dev/tty", "echo").Run()

tryAgain:
	fmt.Print("\nSelect Stream : ")
	var b []byte = make([]byte, 1)

	os.Stdin.Read(b)
	i,err := strconv.ParseInt( string( b[0] ), 10, 8 )
	if err != nil {
		fmt.Printf( "\nInput an integer from 0 to %d\n", len(sel)-1)
		goto tryAgain
	}

	return sel[i]

}

func inputProName(ch chan<- string, done chan<- bool) {
	str := []byte{}
	var b []byte = make([]byte, 1)
outer:
	for {
		fmt.Printf("\nSearch Stream : %s", string(str))
		os.Stdin.Read(b)
		switch b[0] {
		case 0x7F:
			if len(str) > 0 {
				str = str[:len(str)-1]
			}
		case 0x0A:
			done <- true
			break outer
		default:
			str = append(str, b[0])
		}
		ch <- string(str)
	}
}
