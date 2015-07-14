package main

import (
	"bufio"
	//"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"syscall"
)

var argMap map[string]string

func main() {
	Authen()

	if goPromote() {
		Run("accurev", "info")
		promote()
		os.Exit(0)
	}

	clearTTY()
	allStreams := queryStreams()
	var candidates []string

	if len(os.Args) > 1 { // when do search with zip file name

		zipFileName := os.Args[1]
		top := topWeight(zipFileName, allStreams) // top match index
		fmt.Println("Recommended base streams for " + zipFileName)
		candidates = getStreams(top)
		dispSlice(candidates)

	} else { // when do manual search

		tMap := make(map[string]string)
		done := make(chan bool)
		pattern := make(chan []byte)

		var pat []byte

		go input(pattern, done)
		go func() {
			for {
				pat = <-pattern
				candidates = filter(pat, allStreams)
				disp(pat, candidates)
				dispatchQuery(pat, candidates, tMap)
			}
		}()

		<-done
	}

	baseStream := pickStream(candidates)
	workSpace := strings.Join([]string{setTCRnum(), baseStream}, "_")
	dir := strings.Join([]string{getPWD(), workSpace}, "/")

	argMap = make(map[string]string)
	argMap["-b"] = baseStream
	argMap["-w"] = workSpace
	argMap["-l"] = dir

	//enc := json.NewEncoder(os.Stdout)
	//enc.Encode(argMap)
	checkOut(argMap)

	//Run( "accurev", "logout" )
}

func dispatchQuery(pat []byte, tgt []string, tNameMap map[string]string) {
	if len(tgt) == 0 {
		return
	}

	if len(tgt) <= 5 {

		jobs := make(chan string, len(tgt))
		for _, s := range tgt {
			jobs <- s
		}

		for i, s := range tgt {
			go func(id int, sn string) {
				if _, has := tNameMap[sn]; !has {
					tNameMap[sn] = fmt.Sprintf("%s\t, %s", sn, getTestProgramName(sn))
				}
				dispMap(pat, tgt, tNameMap)
			}(i, s)
		}
	}
}

func goPromote() bool {

	if TryRun("accurev", "stat", ".") != nil {
		return false
	}

	return true
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

func promote() {
	fmt.Println("External Files:")
	o, _ := Output("accurev", "stat", "-R", ".", "-x")
	if len(o) != 0 {
		fmt.Println(o)
		confirm("Proceed to add new files ? (Y/n) ")
		Run("accurev", "add", "-x")
	}

	fmt.Println("Modified Files:")
	o, _ = Output("accurev", "stat", "-R", ".", "-m")
	if len(o) != 0 {
		fmt.Println(o)
		confirm("Proceed to promote modified files ? (Y/n) ")
		Run("accurev", "keep", "-m", "-c", "automated")
	}

	fmt.Println("Pending Files:")
	o, _ = Output("accurev", "stat", "-R", ".", "-p")
	if len(o) != 0 {
		fmt.Println(o)
		confirm("Proceed to promote pending files ? (Y/n) ")
		Run("accurev", "promote", "-p", "-c", "automated")
	}
}

func confirm(msg string) {

	fmt.Print(msg)
	var b []byte = make([]byte, 2)
again:
	os.Stdin.Read(b)
	switch b[0] {
	case 'Y':
		return
	case 'y':
		fmt.Print(msg)
		goto again
	default:
		os.Exit(0)
	}

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
		fmt.Println("Checking out workspace....")
		args := append([]string{"mkws"}, flat(argMap)...)
		Run("accurev", args...)
		os.Chdir(argMap["-l"])
		Run("accurev", "update")
	default:
		return
	}
}

func getPWD() string {
	dir, e := os.Getwd()
	if e != nil {
		fmt.Println(e)
		os.Exit(1)
	}

	return dir
}

func getTestProgramName(bs string) string {
	//fmt.Printf("\nQuerying Test Program Name ")

	patDieConfDir := "[1-9]+D([1-9]+CE)?"
	r, _ := regexp.Compile("_" + patDieConfDir)
	if !r.MatchString(bs) { // void of "_xDxCE" in stream name
		//fmt.Printf("[not applicable]\n")
		return ""
	}

	r, _ = regexp.Compile("./" + patDieConfDir)
	o, e := Output("accurev", "files", "-s", bs)
	if e != nil || !r.MatchString(o) { // void of file ./xDxCE/proname on server
		//fmt.Printf("[nothing found]\n")
		return ""
	}

	proNameFile := r.FindString(o)[2:] + "/proname"
	o, _ = Output("accurev", "cat", "-v", bs, proNameFile)
	//fmt.Printf(" : %s\n", o)
	return strings.TrimSpace(fmt.Sprintf("%s", o))

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
		fmt.Printf("\nInput %s", prompt)
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
	cmd.Stderr = os.Stderr

	e := cmd.Run()
	if e != nil {
		fmt.Printf("Exit on execution of %s %s\n", s, arg)
		os.Exit(1)
	}
	return e
}

func Output(s string, arg ...string) (string, error) { //when stdout is needed
	o, e := exec.Command(s, arg...).Output()
	return string(o), e
}

func Authen() {
	if TryRun("accurev", "show", "sessions") != nil {
		fmt.Println("AccuRev Login >>")
		Run("accurev", "login")
	}
}

func filter(pat []byte, allStreams []string) []string {
	if len(pat) == 0 { // when no pattern is input
		return []string{}
	}

	caseIgnore := "(?i)"
	r, e := regexp.Compile(caseIgnore + string(pat))
	if e != nil {
		return []string{}
	}

	opt := []string{}
	for _, s := range allStreams {
		if r.MatchString(s) {
			opt = append(opt, s)
		}
	}
	return opt
}

func dispSlice(arr []string) {
	for i, s := range arr {
		fmt.Printf("\t%d : %s\n", i, s)
	}
}

func clearTTY() {
	Run("echo", "-e", "\\033c") // clear the screen
}

func dispMap(pat []byte, arr []string, m map[string]string) {
	clearTTY()
	//if len(arr) > 0 {
	//	fmt.Print("\n\n\n")
	//}

	tgt := append([]string{}, arr...) // defensive copy
	for i, s := range tgt {
		v, has := m[s]
		if has {
			tgt[i] = v
		}
	}
	dispSlice(tgt)
	fmt.Printf("\nSearch base stream : %s", string(pat))
}

func disp(pat []byte, arr []string) {
	clearTTY()
	//if len(arr) > 0 {
	//	fmt.Print("\n\n\n")
	//}
	dispSlice(arr)
	fmt.Printf("\nSearch base stream : %s", string(pat))
}

func pickStream(candidates []string) string {
	if len(candidates) == 0 {
		fmt.Printf("\nNothing matches. Quit\n")
		os.Exit(0)
	}
	//if len(candidates) == 1 {
	//	return candidates[0]
	//}

	maxIdx := int64(len(candidates) - 1)
	msg := fmt.Sprintf("\nChoose stream from 0 to %d : ", maxIdx)
	br := bufio.NewReader(os.Stdin)
	for {
		fmt.Print(msg)
		s, _ := br.ReadString('\n')
		i, e := strconv.ParseInt(strings.TrimSpace(s), 10, 0)
		if e != nil || i > maxIdx {
			continue
		} else {
			return candidates[i]
		}
	}
}

func input(ch chan<- []byte, done chan<- bool) {

	resetTTYonTerm()
	setTTY()

	pat := []byte{}
	b := make([]byte, 1)

	disp(pat, []string{})

	for {
		os.Stdin.Read(b)
		switch b[0] {
		case 0x7F: // backspace
			if len(pat) > 0 {
				pat = pat[:len(pat)-1]
			}
		case '\n':
			resetTTY()
			done <- true
			return
		default:
			pat = append(pat, b[0])
		}
		ch <- pat
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
	fmt.Printf("%d Dynamic Streams were queried\n", len(sms))
	return sms

	//return []string{//a8w4db
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
