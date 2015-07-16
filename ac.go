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

var (
	argMap map[string]string
	usrNm  chan string
)

func main() {

	if goPromote() {
		Login()
		promote()
		os.Exit(0)
	}

	Authen()
	usrNm = userName()
	allStreams := queryStreams()
	clearTTY()
	var candidates []string

	if len(os.Args) > 1 { // when do search with zip file name

		zipFileName := os.Args[1]
		top := topWeight(zipFileName, allStreams) // top match index
		fmt.Println("Recommended base streams for " + zipFileName)
		candidates = getStreams(top)
		dispSlice(candidates)

	} else { // when do manual search

		pattern, confirmed := input()
		go func() {
			tMap := make(map[string]string) // stream -> tXXXXXX
			alls := <-allStreams
			for {
				pat := <-pattern
				candidates = filter(pat, alls)
				disp(pat, candidates)
				dispatchQuery(pat, candidates, tMap)
			}
		}()

		<-confirmed

	}

	baseStream := pickStream(candidates)
	workSpace := strings.Join([]string{setTCRnum(), baseStream}, "_")
	dir := strings.Join([]string{pwd(), workSpace}, "/")

	argMap = make(map[string]string)
	argMap["-b"] = baseStream
	argMap["-w"] = workSpace
	argMap["-l"] = dir

	//enc := json.NewEncoder(os.Stdout)
	//enc.Encode(argMap)
	checkOut(argMap)

	Logout()
}

func dispatchQuery(pat []byte, tgt []string, tNameMap map[string]string) {
	if len(tgt) == 0 {
		return
	}

	if len(tgt) == 2 { // creates # of goroutines equivalent to slice length
		dispMap(pat, tgt, tNameMap)

		jobs := make(chan string, len(tgt))
		for _, s := range tgt {
			jobs <- s
		}
		close(jobs)

		for s := range jobs {
			if _, has := tNameMap[s]; !has {
				go func(sn string) {
					tNameMap[sn] = fmt.Sprintf("%s\t, %s", sn, getTestProgramName(sn))
					dispMap(pat, tgt, tNameMap)
				}(s)
			}
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
	fmt.Printf("Base Stream : %s\n", m["-b"])
	fmt.Printf("Local Path : %s\n", m["-l"])
	fmt.Printf("\nHi %s", <-usrNm)
	fmt.Printf("\nProceed to create workspace (Y/n) ? ")

	var b []byte = make([]byte, 2)
	os.Stdin.Read(b)
	switch b[0] {
	case 'y':
		Login()
		fmt.Println("Checking out workspace....")
		args := append([]string{"mkws"}, flat(argMap)...)
		Run("accurev", args...)
		os.Chdir(argMap["-l"])
		Run("accurev", "update")
	default:
		return
	}
}

func userName() chan string {
	c := make(chan string)
	go func() {
		o, _ := Output("accurev", "info")
		r, e := regexp.Compile("Principal:.*\n")
		if e != nil {
			c <- ""
		}

		c <- strings.TrimSpace(strings.Split(r.FindString(o), ":")[1])
	}()
	return c
}

func pwd() string {
	dir, e := os.Getwd()
	if e != nil {
		fmt.Println(e)
		os.Exit(1)
	}

	return dir
}

func getTestProgramName(bs string) string {

	patDieConfDir := "[1-9]+D([1-9]+CE)?"
	r, _ := regexp.Compile(patDieConfDir)
	if !r.MatchString(bs) { // void of "xDxCE" in stream name
		return ""
	}

	proNameFile := "./" + r.FindString(bs) + "/proname"
	o, _ := Output("accurev", "cat", "-v", bs, proNameFile)
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
		Login()
	}
}

func Logout() {
	Run("accurev", "logout")
}

func Login() {
	fmt.Println("\nAccuRev Login >>")
	Run("accurev", "login")
}

func filter(pat []byte, allStreams []string) []string {
	if len(pat) == 0 {
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

func dispMap(pat []byte, arr []string, m map[string]string) {
	tgt := append([]string{}, arr...) // defensive copy
	for i, s := range tgt {
		if v, has := m[s]; has {
			tgt[i] = v
		}
	}

	disp(pat, tgt)
}

func disp(pat []byte, arr []string) {

	clearTTY()
	dispSlice(arr)
	fmt.Printf("\nSearch base stream : %s", string(pat))
}

func pickStream(candidates []string) string {
	if len(candidates) == 0 {
		fmt.Printf("\nNothing matches. Quit\n")
		os.Exit(0)
	}

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

func input() (chan []byte, chan bool) {
	patCh := make(chan []byte)
	done := make(chan bool)

	resetTTYonTerm()
	setTTY()
	disp([]byte{}, []string{})

	go func() {
		pat := []byte{}
		b := make([]byte, 1)
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
			patCh <- pat
		}
	}()

	return patCh, done

}

func queryStreams() chan []string {
	c := make(chan []string)
	go func() {
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
		//fmt.Printf("%d Dynamic Streams were queried\n", len(sms))
		c <- sms
	}()

	//go func() {//a8w4db
	//	c<- []string{
	//		"T73_1Znm_128gb_ABL_eX3_2P_SDSIP_768_16D",
	//		"T73_1Ynm_64gb_ABL_eX3_1P_TSOP_768_1D",
	//		"T73_1Znm_64gb_ABL_eX2_4P_SSD-BGA_768_8D",
	//	}
	//}()

	return c
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

func clearTTY() {
	Run("echo", "-e", "\\033c") // clear the screen
}
