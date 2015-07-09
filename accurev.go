package main

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

var sel []string
var str []byte

func main() {

	allStreams := queryStreams()

	done := make(chan bool)
	ch := make(chan string)
	go inputProName(ch, done)
	go search(ch, allStreams)

	<-done

	baseStream := setStream()
	fmt.Printf("\n%s\n", baseStream)

}

func getStream(ch <-chan string) {
	fmt.Printf("\nBase Stream is %s\n", <-ch)
}

func search(ch <-chan string, allStreams []string) {
	caseIgnore := "(?i)"
	for {
		pat := <-ch
		if len(pat) == 0 {
			sel = []string{}
			disp(str, sel)
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
		disp(str, sel)
	}
}

func disp(str []byte, sel []string) {
	fmt.Print("\n\n\n")
	for i, s := range sel {
		fmt.Printf("\t%d : %s\n", i, s)
	}
	fmt.Printf("\nSearch Base Stream : %s", string(str))
}

func setStream() string {
	if len(sel) == 0 {
		fmt.Printf("\nNothing is selected, so quit\n")
		os.Exit(1)
	}
	if len(sel) == 1 {
		return sel[0]
	}

	exec.Command("stty", "-F", "/dev/tty", "echo").Run()
	//resetTTY()

tryAgain:
	fmt.Print("\nChoose Stream# : ")
	var b []byte = make([]byte, 1)

	os.Stdin.Read(b)
	i, err := strconv.ParseInt(string(b[0]), 10, 8)
	if err != nil || int(i) >= len(sel) {
		fmt.Printf("\nInput an integer from 0 to %d\n", len(sel)-1)
		goto tryAgain
	}

	return sel[i]

}

func inputProName(ch chan<- string, done chan<- bool) {
	setTTY()

	str = []byte{}
	var b []byte = make([]byte, 1)
	
	fmt.Printf("Search Backing Stream : %s", string(str))

outer:
	for {
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

func queryStreams() []string {
	fmt.Println("Connecting to Accurev Server ...")
	cmd := "accurev"
	args := []string{"show", "streams", "-d", "-p", "MT_Production_Test_Programs"}
	o, e := exec.Command(cmd, args...).Output()
	if e != nil {
		fmt.Println(e)
		os.Exit(1)
	}

	sms := []string{}
	for _, b := range strings.Split(string(o), "\n") {
		ln := strings.Split(strings.TrimRight(b, " "), " ")
		if len(ln) > 0 && strings.EqualFold( "Y", ln[len(ln)-1]) { // true for Dynamic Stream
			sms = append(sms, ln[0])
		}
	}
	fmt.Printf("%d Dynamic Streams are queried\n", len(sms))
	return sms

	//return []string{
	//	"T73_1Znm_128gb_ABL_eX3_2P_SDSIP_768_16D",
	//	"T73_1Ynm_64gb_ABL_eX3_1P_TSOP_768_1D",
	//	"T73_1Znm_64gb_ABL_eX2_4P_SSD-BGA_768_8D",
	//}
}

func setTTY() {
	// disable input buffering
	exec.Command("stty", "-F", "/dev/tty", "cbreak", "min", "1").Run()
	// do not display entered characters on the screen
	exec.Command("stty", "-F", "/dev/tty", "-echo").Run()
}

func resetTTY() {
	exec.Command("stty", "-F", "/dev/tty", "sane").Run()
}


