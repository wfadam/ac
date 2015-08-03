package main

import (
	"bufio"
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

	Authen()
	usrNm = userName()

	if goPromote() {
		fmt.Println("Calculating changes ...")
		promote()
		os.Exit(0)
	}

	allStreams := queryStreams()

	clearTTY()
	var candidates []string

	if len(os.Args) > 1 { // when search by zip file name

		zipFileName := os.Args[1]
		top := topWeight(zipFileName, allStreams) // top match index
		fmt.Println("Recommended base streams for " + zipFileName)
		candidates = getStreams(top)
		dispSlice(candidates)

	} else { // when search by pattern

		pattern, confirmed := input()
		tMap := make(map[string]string) // stream -> tXXXXXX
		wk := workers(5, tMap)
		alls := <-allStreams
		go func() {
			for {
				pat := <-pattern
				candidates = filter(pat, alls)
				dispMap(pat, candidates, tMap)
				dispatchQuery(candidates, wk)
			}
		}()
		<-confirmed

	}

	baseStream := pickStream(candidates)

	optMsg := `
0 : Check out a workspace
1 : Make a dynamic stream
`
	fmt.Println(optMsg)
	var b []byte = make([]byte, 2)
	for {
		fmt.Print("Choose option # : ")
		os.Stdin.Read(b)
		switch b[0] {
		case '0':
			goto doCheckOut
		case '1':
			fmt.Printf("\nBacking stream : %s\n", baseStream)
			stream := readInput("New Stream Name : ")
			makeStream(stream, baseStream)
			os.Exit(0)
		default:
			continue
		}
	}
doCheckOut:

	workSpace := strings.Join([]string{setTCRnum(), baseStream}, "_")
	dir := strings.Join([]string{pwd(), workSpace}, "/")

	argMap = make(map[string]string)
	argMap["-b"] = baseStream
	argMap["-w"] = workSpace
	argMap["-l"] = dir

	checkOut(argMap)

}

func workers(cnt int, m map[string]string) chan string {

	c := make(chan string, cnt)

	for i := 0; i < cnt; i++ {
		go func() {
			for {
				sn := <-c
				if _, has := m[sn]; !has {
					m[sn] = fmt.Sprintf("%s\t, %s", sn, getTestProgramName(sn))
				}
			}
		}()
	}

	return c
}

func dispatchQuery(jobs []string, wk chan string) {
	if len(jobs) == 0 {
		return
	}

	if len(jobs) > 2*cap(wk) { // job count is limited
		return
	}

	go func() {
		for _, j := range jobs {
			wk <- j
		}
	}()
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
	oa := []string{}

	o, _ := Output("accurev", "stat", "-R", ".", "-x")
	oa = append(oa, o)

	o, _ = Output("accurev", "stat", "-R", ".", "-m")
	oa = append(oa, o)

	if len(strings.TrimSpace(strings.Join(oa, ""))) == 0 {
		fmt.Printf("Nothing changed. Quit\n")
		os.Exit(0)
	}

	comments := comment(oa)
	commentFile := fmt.Sprintf("@%s", genCommentFile(comments))

	clearTTY()
	fmt.Println(">>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>Comments Begin")
	fmt.Println(comments)
	fmt.Println(">>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>Comments End")

	fmt.Println()
	for _, s := range oa {
		fmt.Println(strings.TrimSpace(s))
	}
	fmt.Println()

	confirm(fmt.Sprintf("Promote all changes on behalf of __%s__ ? (Y/n) ", <-usrNm))
	TryRun("accurev", "add", "-x")
	TryRun("accurev", "keep", "-m", "-c", commentFile)
	Run("accurev", "promote", "-p", "-c", commentFile)
}

func comment(modFiles []string) string {
	msg := []string{}
	msg = append(msg, fmt.Sprintf("Workspace Path: %s", pwd()))
	msg = append(msg, lastRevHist(modFiles))
	return strings.TrimSpace(strings.Join(msg, "\n"))
}

func readInput(prompt string) string {
	fmt.Print(prompt)

	var b []byte = make([]byte, 80)
	os.Stdin.Read(b)

	rtn := strings.TrimSpace(string(b))

	if i := strings.Index(rtn, "\n"); i >= 0 {
		return rtn[:i]
	}
	return rtn
}

func confirm(msg string) {

	fmt.Print(msg)
	var b []byte = make([]byte, 2)
	for {
		os.Stdin.Read(b)
		switch b[0] {
		case 'Y':
			return
		case 'y':
			fmt.Print(msg)
			continue
		default:
			os.Exit(0)
		}
	}
}

func makeStream(stream, backingStream string) {
	confirm(fmt.Sprintf("Create a new stream on behalf of __%s__ ? (Y/n) ", <-usrNm))

	Run("accurev", "mkstream", "-s", stream, "-b", backingStream)
}

func checkOut(m map[string]string) {

	fmt.Printf("\n\n\n")
	fmt.Printf("Base Stream : %s\n", m["-b"])
	fmt.Printf("Local Path  : %s\n", m["-l"])
	msg := fmt.Sprintf("\nCreate a workspace on behalf of ___%s___  (Y/n) ? ", <-usrNm)
	fmt.Print(msg)

	var b []byte = make([]byte, 2)
	for {
		os.Stdin.Read(b)
		switch b[0] {
		case 'Y':
			//Login()
			fmt.Println("Checking out workspace....")
			args := append([]string{"mkws"}, flat(argMap)...)
			Run("accurev", args...)
			os.Chdir(argMap["-l"])
			Run("accurev", "update")
			return
		case 'y':
			fmt.Print(msg)
			continue
		default:
			os.Exit(0)
		}
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

	folderName := r.FindString(bs)
	var proNameFile string
	if strings.Contains(folderName, "CE") {
		proNameFile = "./" + folderName + "/proname"
	} else {
		proNameFile = "./" + folderName + "1CE/proname"
	}

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
		fmt.Printf("Error out on execution of %s %s\n", s, arg)
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
	fmt.Printf("\nSearch streams : %s", string(pat))
}

func pickStream(candidates []string) string {
	if len(candidates) == 0 {
		fmt.Printf("\nNothing matches. Quit\n")
		os.Exit(0)
	}

	maxIdx := int64(len(candidates) - 1)
	msg := fmt.Sprintf("\nChoose a stream from 0 to %d : ", maxIdx)
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
		s, e := Output("accurev", "show", "streams", "-p", "MT_Production_Test_Programs")
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
