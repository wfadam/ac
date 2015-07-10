package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"syscall"
)

var sel []string // filtered stream slice by pattern
var argMap map[string]string

func main() {

	resetTTYonTerm()
	if authen() != nil {
		fmt.Println("Program exited")
	}

	allStreams := queryStreams()

	done := make(chan bool)
	pattern := make(chan []byte)

	setTTY()
	go inputProName(pattern, done)
	go search(pattern, allStreams)
	<-done
	resetTTY()

	baseStream := setStream()
	workSpace := strings.Join([]string{setTCRnum(), baseStream}, "_")
	dir := strings.Join([]string{setDir(), workSpace}, "/")

	argMap = make(map[string]string)
	argMap["-b"] = baseStream
	argMap["-w"] = workSpace
	argMap["-l"] = dir

	enc := json.NewEncoder(os.Stdout)
	enc.Encode(argMap)

	checkOut(argMap)

	//Run( "accurev", "logout" )
}

func bizCard() {
	fmt.Println("\n\n\n<< Call 85725 for any help >>\n")
}
func resetTTYonTerm() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		resetTTY()
		bizCard()
		os.Exit(0)
	}()
}

func flat(m map[string]string) []string {
	cmd := []string{}
	for k, v := range m {
		cmd = append(cmd, k)
		cmd = append(cmd, v)
	}
	return cmd
}

func addKeep() {
	os.Chdir(argMap["-l"])

	fmt.Println("External Files:")
	o, e := Output("accurev", "stat", "-R", ".", "-x")
	fmt.Println(o)

	fmt.Println("Modified Files:")
	o, e = Output("accurev", "stat", "-R", ".", "-m")
	fmt.Println(o)

	fmt.Println("Pending Files:")
	o, e = Output("accurev", "stat", "-R", ".", "-m")
	fmt.Println(o)

	fmt.Println(e)

}
func checkOut(m map[string]string) {
	fmt.Printf("\n\n\n")
	fmt.Printf("Base Stream    : %s\n", m["-b"])
	fmt.Printf("WorkSpace Path : %s\n", m["-l"])
	fmt.Printf("\nProceed to create workspace (y/n) ? ")

	var b []byte = make([]byte, 2)
	os.Stdin.Read(b)
	switch b[0] {
	case 'y':
		fmt.Println("Checking out....")
		args := append([]string{"mkws"}, flat(argMap)...)
		Run("accurev", args...)
		os.Chdir(argMap["-l"])
		Run("accurev", "update")
	default:
		return
	}
}

func setDir() string {
	dir, e := os.Getwd()
	if e != nil {
		fmt.Println(e)
		os.Exit(1)
	}

	return dir
}

func setTCRnum() string {
	pat := "^[0-9]+(\\.?[0-9]+)?\n"
	r, err := regexp.Compile(pat)
	if err != nil {
		fmt.Println(err)
	}
	br := bufio.NewReader(os.Stdin)

	prompt := "TCR-"
	for {
		fmt.Printf("Input %s", prompt)
		s, _ := br.ReadString('\n')
		if r.MatchString(s) {
			return strings.TrimSpace(prompt + s)
		} else {
			fmt.Println("Examples:\n\t1024\nor\t1024.1")
		}
	}
}

func TryRun(s string, arg ...string) error { //when only exit value matters
	return exec.Command(s, arg...).Run()
}

func Run(s string, arg ...string) error { //when stdin is needed
	cmd := exec.Command(s, arg...)
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

func Output(s string, arg ...string) (string, error) { //when stdout is needed
	o, e := exec.Command(s, arg...).Output()
	return string(o), e
}

func authen() error {
	if TryRun("accurev", "show", "sessions") != nil {
		fmt.Println("AccuRev Login >>")
		return Run("accurev", "login")
	}
	return nil
}

func search(ch <-chan []byte, allStreams []string) {
	caseIgnore := "(?i)"
	for {
		pat := <-ch
		if len(pat) == 0 {
			sel = []string{}
			disp(pat, sel)
			continue
		}

		r, err := regexp.Compile(caseIgnore + string(pat))
		if err != nil {
			disp(pat, sel)
			continue
		}

		sel = []string{}
		for _, s := range allStreams {
			if r.MatchString(s) {
				sel = append(sel, s)
			}
		}
		disp(pat, sel)
	}
}

func disp(pat []byte, arr []string) {
	if len(arr) > 0 {
		fmt.Print("\n\n\n")
	}
	for i, s := range arr {
		fmt.Printf("\t%d : %s\n", i, s)
	}
	fmt.Printf("\nSearch Base Stream : %s", string(pat))
}

func setStream() string {
	if len(sel) == 0 {
		fmt.Printf("\nNothing matches, quit\n")
		os.Exit(0)
	}
	if len(sel) == 1 {
		return sel[0]
	}

	br := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("\nChoose Stream # : ")
		s, _ := br.ReadString('\n')
		i, e := strconv.ParseInt(strings.TrimSpace(s), 10, 0)
		if e != nil || int(i) >= len(sel) {
			fmt.Printf("\nInput an integer from 0 to %d\n", len(sel)-1)
			continue
		} else {
			return sel[i]
		}
	}
}

func inputProName(ch chan<- []byte, done chan<- bool) {

	str := []byte{}
	var b []byte = make([]byte, 1)

	fmt.Printf("Search Backing Stream : %s", string(str))

outer:
	for {
		os.Stdin.Read(b)
		switch b[0] {
		case 0x7F: // backspace
			if len(str) > 0 {
				str = str[:len(str)-1]
			}
		case '\n':
			done <- true
			break outer
		default:
			str = append(str, b[0])
		}
		ch <- str
	}
}

func queryStreams() []string {

	s, e := Output("accurev", "show", "streams", "-d", "-p", "MT_Production_Test_Programs")
	if e != nil {
		fmt.Println(e)
		os.Exit(1)
	}

	sms := []string{}
	for _, b := range strings.Split(s, "\n") {
		ln := strings.Split(strings.TrimRight(b, " "), " ")
		if len(ln) > 0 && strings.EqualFold("Y", ln[len(ln)-1]) { // true for Dynamic Stream
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
