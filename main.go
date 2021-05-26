package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"time"
	"strings"
	"math/rand"

	flags "github.com/jessevdk/go-flags"
)

const appName = "ntimes"

var (
	version   = ""
	gitCommit = ""
)

func getDelayGenerator(delay string) func() int {
	if !strings.Contains(delay,"(") {
		d, _ := strconv.Atoi(delay)
		return func() int {return d}
	}
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	spl := strings.FieldsFunc(delay, func(c rune) bool {return c == '(' || c == ')'})
	if spl[0] == "exp" {
		rate, _ := strconv.ParseFloat(spl[1], 64)
		return func() int {return int(r.ExpFloat64()/rate)}
	}
	panic("Cannot define delay generator")
}

type options struct {
	Parallels   int  `short:"p" long:"parallels" description:"Parallel degree of execution" default:"1"`
	Delay   string  `short:"d" long:"delay" description:"Miliseconds to sleep after a job has been started (can use exp(l))" default:"0"`
	ShowVersion bool `short:"v" long:"version" description:"Show version"`
}

func main() {
	var opts options
	parser := flags.NewParser(&opts, flags.Default^flags.PrintErrors)
	parser.Name = appName
	parser.Usage = "N [OPTIONS] -- COMMAND"

	args, err := parser.Parse()
	if err != nil {
		if flagsErr, ok := err.(*flags.Error); ok {
			if flagsErr.Type == flags.ErrHelp {
				parser.WriteHelp(os.Stderr)

				return
			}
		}

		errorf("flag parse error: %s", err)
		os.Exit(1)
	}

	if opts.ShowVersion {
		_, _ = io.WriteString(os.Stdout, fmt.Sprintf("%s v%s, build %s\n", appName, version, gitCommit))

		return
	}

	cnt, err := strconv.Atoi(args[0])
	cmdName := args[1]
	cmdArgs := args[2:]

	if err != nil {
		panic(err)
	}

	ntimes(cnt, cmdName, cmdArgs, os.Stdin, os.Stdout, os.Stderr, opts.Parallels, getDelayGenerator(opts.Delay))
}

func ntimes(cnt int, cmdName string, cmdArgs []string, stdin io.Reader, stdout io.Writer, stderr io.Writer, parallels int, delay func() int) {
	var wg sync.WaitGroup

	sema := make(chan bool, parallels)

	for i := 0; i < cnt; i++ {
		wg.Add(1)

		go func() {
			sema <- true

			defer func() {
				wg.Done()
				<-sema
			}()

			cmd := exec.Command(cmdName, cmdArgs...)
			cmd.Stdin = stdin
			cmd.Stdout = stdout
			cmd.Stderr = stderr

			err := cmd.Run()
			if err != nil {
				panic(err)
			}
		}()
		del := delay()
		//fmt.Println("Delay:",del)
		time.Sleep(time.Duration(del) * time.Millisecond)
	}

	wg.Wait()
	close(sema)
}

func errorf(message string, args ...interface{}) {
	subMessage := fmt.Sprintf(message, args...)
	_, _ = fmt.Fprintf(os.Stderr, "%s: %s\n", appName, subMessage)
}
