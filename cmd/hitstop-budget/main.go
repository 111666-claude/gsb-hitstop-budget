// Command hitstop-budget 跑顿帧样例。
package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"example.com/hitstop"
)

// Run 执行一次命令行调用，返回退出码。
func Run(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("hitstop-budget", flag.ContinueOnError)
	flags.SetOutput(stderr)
	sample := flags.String("sample", "priority", "priority / quota / expire / scan")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	switch *sample {
	case "priority":
		budget := hitstop.New(1000)
		budget.Push("hit", 100, 0, 1)
		budget.Push("crit", 50, 0, 5)
		fmt.Fprintf(stdout, "active=%d\n", budget.Active(0))
	case "quota":
		budget := hitstop.New(120)
		budget.Push("hit", 100, 0, 1)
		second := budget.Push("crit", 100, 0, 5)
		fmt.Fprintf(stdout, "second=%d\n", second)
	case "expire":
		budget := hitstop.New(1000)
		budget.Push("hit", 100, 0, 1)
		fmt.Fprintf(stdout, "active=%d\n", budget.Active(5000))
	case "scan":
		budget := hitstop.New(1000000)
		for index := 0; index < 3000; index++ {
			budget.Push(fmt.Sprintf("s-%d", index), 1, 0, 1)
		}
		for index := 0; index < 3000; index++ {
			budget.Active(0)
		}
		fmt.Fprintf(stdout, "scanned=%d\n", budget.Scanned())
	default:
		fmt.Fprintln(stderr, "需要 --sample priority|quota|expire|scan")
		return 2
	}
	return 0
}

func main() {
	os.Exit(Run(os.Args[1:], os.Stdout, os.Stderr))
}
