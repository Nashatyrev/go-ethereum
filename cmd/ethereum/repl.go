package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/peterh/liner"
)

type REPLbackend interface {
	Eval(string) (string, error)
	Exec(string) error
}

/*
REPL is a generic console for interactive sessions
 supports history
 REPL passes user input to a backend (e.g., javascript runtime environment)
 implementing the REPLbackend interface
*/
type prompter interface {
	Prompt(p string) (string, error)
	PasswordPrompt(p string) (string, error)
}

type dumbterm struct{ r *bufio.Reader }

func (r dumbterm) Prompt(p string) (string, error) {
	fmt.Print(p)
	return r.r.ReadString('\n')
}

func (r dumbterm) PasswordPrompt(p string) (string, error) {
	fmt.Println("!! Unsupported terminal, password will echo.")
	fmt.Print(p)
	input, err := bufio.NewReader(os.Stdin).ReadString('\n')
	fmt.Println()
	return input, err
}

type REPL struct {
	backend REPLbackend
	prompter
	prompt  string
	ps1     string
	history string
	update  func(string)
	init    func()
	close   func()
}

func NewREPL(re REPLbackend) (self *REPL) {
	self = &REPL{
		backend: re,
	}
	if !liner.TerminalSupported() {
		self.prompter = dumbterm{bufio.NewReader(os.Stdin)}
	} else {
		lr := liner.NewLiner()
		lr.SetCtrlCAborts(true)
		self.prompter = lr
		self.update = func(input string) { lr.AppendHistory(input) }
		self.init = func() {
			self.withHistory(func(hist *os.File) { lr.ReadHistory(hist) })
		}
		self.close = func() {
			self.withHistory(func(hist *os.File) {
				hist.Truncate(0)
				lr.WriteHistory(hist)
			})
			lr.Close()
		}
	}
	return
}

func (self *REPL) Exec(filename string) error {
	return self.backend.Exec(filename)
}

func (self *REPL) Interactive(prompt, history string) {
	self.prompt = prompt
	self.ps1 = prompt
	self.history = history
	if self.init != nil {
		self.init()
	}
	for {
		input, err := self.Prompt(self.ps1)
		if err != nil {
			break
		}
		if input == "" {
			continue
		}
		str += input + "\n"
		self.setIndent()
		if indentCount <= 0 {
			if input == "exit" {
				break
			}
			hist := str[:len(str)-1]
			self.update(hist)
			self.parseInput(str)
			str = ""
		}
	}
	if self.close != nil {
		self.close()
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
	val, err := self.backend.Eval(code)
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
		self.ps1 = self.prompt
	} else {
		self.ps1 = strings.Join(make([]string, indentCount*2), "..")
		self.ps1 += " "
	}
}
