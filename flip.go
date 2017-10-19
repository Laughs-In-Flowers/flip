package flip

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"sort"
)

type Flip interface {
	Commander
	Instructer
	Executer
	Cleaner
}

type flip struct {
	name string
	*commander
	*instructer
	*cleaner
}

func New(name string) *flip {
	f := &flip{
		name:    name,
		cleaner: newCleaner(),
	}
	f.commander = newCommander(f)
	f.instructer = newInstructer(f, os.Stdout)
	f.SetCleanup(ExitUsageError, f.Instruction)
	f.SetGroup("", 0)
	return f
}

type Grouper interface {
	GetGroup(string) *group
	SetGroup(string, int, ...Command) Flip
}

type Commander interface {
	Grouper
	GetCommand(string) Command
	SetCommand(...Command) Flip
	AddCommand(string, ...string) Flip
}

type commander struct {
	f      Flip
	groups *groups
}

func newCommander(f Flip) *commander {
	return &commander{f, newGroups()}
}

func (c *commander) GetGroup(name string) *group {
	for _, g := range c.groups.has {
		if name == g.name {
			return g
		}
	}
	return nil
}

func (c *commander) SetGroup(name string, priority int, cmds ...Command) Flip {
	c.groups.has = append(c.groups.has, NewGroup(name, priority))
	for _, v := range cmds {
		v.SetGroup(name)
	}
	c.SetCommand(cmds...)
	return c.f
}

func (c *commander) GetCommand(k string) Command {
	for _, g := range c.groups.has {
		for _, cmd := range g.commands {
			if k == cmd.Tag() {
				return cmd
			}
		}
	}
	return nil
}

func (c *commander) SetCommand(cmds ...Command) Flip {
	for _, cmd := range cmds {
		g := c.GetGroup(cmd.Group())
		g.commands = append(g.commands, cmd)
	}
	return c.f
}

func (f *flip) AddCommand(nc string, args ...string) Flip {
	switch nc {
	case "help":
		return f.addHelp()
	case "version":
		return f.addVersion(args...)
	}
	return f
}

type CommandFunc func(context.Context, []string) (context.Context, ExitStatus)

type Command interface {
	Group() string
	SetGroup(string)
	Tag() string
	Priority() int
	Escapes() bool
	Use(io.Writer)
	Execute(context.Context, []string) (context.Context, ExitStatus)
	Flagger
}

type command struct {
	group, tag string
	use        string
	priority   int
	escapes    bool
	cfn        CommandFunc
	*FlagSet
}

func NewCommand(group, tag, use string,
	priority int,
	escapes bool,
	cfn CommandFunc,
	fs *FlagSet) Command {
	return &command{group, tag, use, priority, escapes, cfn, fs}
}

func (c *command) SetGroup(k string) {
	c.group = k
}

func (c *command) Group() string {
	return c.group
}

func (c *command) Tag() string {
	return c.tag
}

func (c *command) Priority() int {
	return c.priority
}

func (c *command) Escapes() bool {
	return c.escapes
}

func (c *command) useHead(o io.Writer) {
	white(o, fmt.Sprintf("-----\n%s [<flags>]:\n", c.tag))
}

func (c *command) useString(o io.Writer) {
	white(o, fmt.Sprintf("\t%s\n\n", c.use))
}

func (c *command) Use(o io.Writer) {
	c.useHead(o)
	c.useString(o)
	c.Usage(o)
	fmt.Fprint(o, "\n")
}

func (c *command) Execute(ctx context.Context, v []string) (context.Context, ExitStatus) {
	if c.cfn != nil {
		return c.cfn(ctx, v)
	}
	return ctx, ExitFailure
}

type groups struct {
	sortBy string
	has    []*group
}

func newGroups() *groups {
	return &groups{"default", make([]*group, 0)}
}

func (g groups) Len() int { return len(g.has) }

func (g groups) Less(i, j int) bool { return g.has[i].priority < g.has[j].priority }

func (g groups) Swap(i, j int) { g.has[i], g.has[j] = g.has[j], g.has[i] }

type group struct {
	name     string
	priority int
	sortBy   string
	commands []Command
}

func NewGroup(name string, priority int, cs ...Command) *group {
	return &group{name, priority, "", cs}
}

func (g group) Len() int { return len(g.commands) }

func (g group) Less(i, j int) bool {
	switch g.sortBy {
	case "alpha":
		return g.commands[i].Tag() < g.commands[j].Tag()
	default:
		return g.commands[i].Priority() < g.commands[j].Priority()
	}
	return false
}

func (g group) Swap(i, j int) {
	g.commands[i], g.commands[j] = g.commands[j], g.commands[i]
}

func (g *group) SortBy(s string) {
	g.sortBy = s
	sort.Sort(g)
}

func (g *group) Use(o io.Writer) {
	g.SortBy("default")
	for _, cmd := range g.commands {
		cmd.Use(o)
	}
}

type Instructer interface {
	Instruction(context.Context)
	SubsetInstruction(c ...Command) func(context.Context)
	Writer
}

type instructer struct {
	titleFmtString string
	output         io.Writer
	ifn            Cleanup
}

func newInstructer(f *flip, o io.Writer) *instructer {
	i := &instructer{"%s [OPTIONS...] {COMMAND} ...\n\n", o, nil}
	i.ifn = defaultInstruction(f, i)
	return i
}

func (i *instructer) Instruction(c context.Context) {
	i.ifn(c)
}

func (i *instructer) SubsetInstruction(cs ...Command) func(context.Context) {
	return func(c context.Context) {
		out := i.Out()
		b := new(bytes.Buffer)
		for _, cmd := range cs {
			cmd.Use(b)
		}
		fmt.Fprint(out, b)
		os.Exit(-2)
	}
}

func titleString(titleFmtString, name string, b *bytes.Buffer) {
	title := Color(Bold, FgHiWhite)
	title(b, fmt.Sprintf(titleFmtString, name))
}

func defaultInstruction(f *flip, i *instructer) Cleanup {
	return func(c context.Context) {
		out := i.Out()
		b := new(bytes.Buffer)
		titleString(i.titleFmtString, f.name, b)

		sort.Sort(f.groups)
		for _, g := range f.groups.has {
			g.Use(b)
		}

		fmt.Fprint(out, b)
		os.Exit(-2)
	}
}

func (i *instructer) Out() io.Writer {
	return i.output
}

func (i *instructer) SetOut(w io.Writer) {
	i.output = w
}

type Executer interface {
	Execute(context.Context, []string) int
}

type ExitStatus int

const (
	ExitNo         ExitStatus = iota // continue processing commands
	ExitSuccess                      // return 0
	ExitFailure                      // return -1
	ExitUsageError                   // return -2
	ExitAny                          // status for cleaning function setup, never return
)

type pop struct {
	start, stop int
	c           Command
	v           []string
}

type pops []*pop

func (p pops) Len() int { return len(p) }

func (p pops) Less(i, j int) bool { return p[i].c.Priority() < p[j].c.Priority() }

func (p pops) Swap(i, j int) { p[i], p[j] = p[j], p[i] }

func isCommand(c *commander, s string) (Command, bool, bool) {
	for _, g := range c.groups.has {
		for _, cmd := range g.commands {
			if s == cmd.Tag() {
				return cmd, true, cmd.Escapes()
			}
		}
	}
	return nil, false, false
}

func queue(f *flip, arguments []string) *pops {
	var ps pops

	for i, v := range arguments {
		if cmd, exists, escapes := isCommand(f.commander, v); exists {
			a := &pop{i, 0, cmd, nil}
			ps = append(ps, a)
			if escapes {
				break
			}
		}
	}

	li := len(ps) - 1
	la := len(arguments)
	for i, v := range ps {
		if i+1 <= li {
			nx := ps[i+1]
			v.stop = nx.start
		} else {
			v.stop = la
		}
	}

	for _, p := range ps {
		p.v = arguments[p.start:p.stop]
	}

	sort.Sort(ps)

	return &ps
}

func execute(f *flip, ctx context.Context, cmd Command, arguments []string) (context.Context, ExitStatus) {
	err := cmd.Parse(arguments)
	if err != nil {
		return ctx, ExitUsageError
	}
	return cmd.Execute(ctx, arguments)
}

func (f *flip) Execute(ctx context.Context, arguments []string) int {
	var exit ExitStatus
	switch {
	case len(arguments) < 1:
		goto INSTRUCTION
	default:
		q := queue(f, arguments)
		for _, p := range *q {
			cmd := p.c
			args := p.v[1:]
			ctx, exit = execute(f, ctx, cmd, args)
			switch exit {
			case ExitSuccess:
				f.RunCleanup(exit, ctx)
				return 0
			case ExitFailure:
				f.RunCleanup(exit, ctx)
				return -1
			case ExitUsageError:
				goto INSTRUCTION
			default:
				continue
			}
		}
	}

INSTRUCTION:
	f.RunCleanup(ExitUsageError, ctx)
	return -2
}

type Cleanup func(context.Context)

type Cleaner interface {
	SetCleanup(ExitStatus, ...Cleanup)
	RunCleanup(ExitStatus, context.Context)
}

type cleaner struct {
	cfns map[ExitStatus][]Cleanup
}

func newCleaner() *cleaner {
	return &cleaner{make(map[ExitStatus][]Cleanup, 0)}
}

func (c *cleaner) SetCleanup(e ExitStatus, cfns ...Cleanup) {
	if c.cfns[e] == nil {
		c.cfns[e] = make([]Cleanup, 0)
	}
	c.cfns[e] = append(c.cfns[e], cfns...)
}

func (c *cleaner) RunCleanup(e ExitStatus, ctx context.Context) {
	for _, cfn := range c.cfns[e] {
		cfn(ctx)
	}
	for _, afn := range c.cfns[ExitAny] {
		afn(ctx)
	}
}
