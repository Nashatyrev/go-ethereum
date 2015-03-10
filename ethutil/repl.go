package ethutil

import (
	"bufio"
	"fmt"
	"os"
	"os/signal"
	"strings"

	"github.com/peterh/liner"
)

type REPLbackend interface {
	Handle(string) (string, error)
}

/*
REPL is a generic console for interactive sessions
 supports history
 REPL passes user input to a backend (e.g., javascript runtime environment)
 implementing the REPLbackend interface
*/
type REPL struct {
	re      REPLbackend
	prompt  string
	lr      *liner.State
	history string
}

func RunREPL(history string, re REPLbackend) {
	repl := &REPL{
		re:      re,
		history: history,
		prompt:  "> ",
	}
	if !liner.TerminalSupported() {
		repl.dumbRead()
	} else {
		lr := liner.NewLiner()
		defer lr.Close()
		lr.SetCtrlCAborts(true)
		repl.withHistory(func(hist *os.File) { lr.ReadHistory(hist) })
		repl.read(lr)
		repl.withHistory(func(hist *os.File) { hist.Truncate(0); lr.WriteHistory(hist) })
	}
}

func (self *REPL) withHistory(op func(*os.File)) {
	hist, err := os.OpenFile(self.history, os.O_RDWR|os.O_CREATE, os.ModePerm)
	if err != nil {
		fmt.Printf("unable to open history file: %v\n", err)
		return
	}
	op(hist)
	hist.Close()
}

func (self *REPL) parseInput(code string) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("[native] error", r)
		}
	}()
	val, err := self.re.Handle(code)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(val)
}

var indentCount = 0
var str = ""

func (self *REPL) setIndent() {
	open := strings.Count(str, "{")
	open += strings.Count(str, "(")
	closed := strings.Count(str, "}")
	closed += strings.Count(str, ")")
	indentCount = open - closed
	if indentCount <= 0 {
		self.prompt = "> "
	} else {
		self.prompt = strings.Join(make([]string, indentCount*2), "..")
		self.prompt += " "
	}
}

func (self *REPL) read(lr *liner.State) {
	for {
		input, err := lr.Prompt(self.prompt)
		if err != nil {
			return
		}
		if input == "" {
			continue
		}
		str += input + "\n"
		self.setIndent()
		if indentCount <= 0 {
			if input == "exit" {
				return
			}
			hist := str[:len(str)-1]
			lr.AppendHistory(hist)
			self.parseInput(str)
			str = ""
		}
	}
}

func (self *REPL) dumbRead() {
	fmt.Println("Unsupported terminal, line editing will not work.")

	// process lines
	readDone := make(chan struct{})
	go func() {
		r := bufio.NewReader(os.Stdin)
	loop:
		for {
			fmt.Print(self.prompt)
			line, err := r.ReadString('\n')
			switch {
			case err != nil || line == "exit":
				break loop
			case line == "":
				continue
			default:
				self.parseInput(line + "\n")
			}
		}
		close(readDone)
	}()

	// wait for Ctrl-C
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, os.Interrupt, os.Kill)
	defer signal.Stop(sigc)

	select {
	case <-readDone:
	case <-sigc:
		os.Stdin.Close() // terminate read
	}
}
