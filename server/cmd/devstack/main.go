// Command devstack reserves this worktree's own local stack — database port and
// volume, server and web ports, daemon home and token — so two agents working
// at once cannot land on each other (WORKFLOW.md, Local isolation).
//
//	devstack env [-dir .] [-format dotenv|powershell|json]   this worktree's environment
//	devstack list                                            every reserved stack
//	devstack release [-dir .]                                give this worktree's slot back
//
// scripts/dev.ps1 and the Makefile both start by running `env`, so neither of
// them decides a port and the two cannot drift apart.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/sparstrow/sparstrowgen/server/internal/devstack"
)

func main() {
	if len(os.Args) < 2 {
		usage()
	}
	switch os.Args[1] {
	case "env":
		env(os.Args[2:])
	case "list":
		list()
	case "release":
		release(os.Args[2:])
	default:
		usage()
	}
}

func env(args []string) {
	fs := flag.NewFlagSet("env", flag.ExitOnError)
	dir := fs.String("dir", ".", "the worktree to reserve a stack for")
	format := fs.String("format", "dotenv", "dotenv, powershell or json")
	_ = fs.Parse(args)

	stack, err := devstack.Reserve(*dir)
	exitOn(err)
	vars, err := stack.Env()
	exitOn(err)

	switch *format {
	case "dotenv":
		for _, v := range vars {
			fmt.Printf("%s=%s\n", v.Key, v.Value)
		}
	case "powershell":
		for _, v := range vars {
			fmt.Printf("$env:%s = '%s'\n", v.Key, strings.ReplaceAll(v.Value, "'", "''"))
		}
	case "json":
		out := map[string]string{}
		for _, v := range vars {
			out[v.Key] = v.Value
		}
		body, err := json.MarshalIndent(out, "", "  ")
		exitOn(err)
		fmt.Println(string(body))
	default:
		usage()
	}
}

func list() {
	stacks, err := devstack.List()
	exitOn(err)
	path, err := devstack.RegistryPath()
	exitOn(err)
	if len(stacks) == 0 {
		fmt.Printf("no stacks reserved yet (%s)\n", path)
		return
	}
	fmt.Printf("%-4s  %-6s %-6s %-6s  %s\n", "SLOT", "DB", "API", "WEB", "WORKTREE")
	for _, s := range stacks {
		fmt.Printf("%-4d  %-6d %-6d %-6d  %s\n", s.Slot, s.DBPort, s.APIPort, s.WebPort, s.Worktree)
	}
	fmt.Printf("\n%s\n", path)
}

func release(args []string) {
	fs := flag.NewFlagSet("release", flag.ExitOnError)
	dir := fs.String("dir", ".", "the worktree whose slot to give back")
	_ = fs.Parse(args)
	exitOn(devstack.Release(*dir))
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: devstack env [-dir .] [-format dotenv|powershell|json] | devstack list | devstack release [-dir .]")
	os.Exit(2)
}

func exitOn(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
