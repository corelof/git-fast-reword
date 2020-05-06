package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

type rewordParam struct {
	hash    string
	message string
}

func getRepoRoot() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		_, err := os.Open(filepath.Join(wd, ".git"))
		if err == nil {
			break
		}
		wd = filepath.Dir(wd)
		if len(wd) <= 1 {
			return "", fmt.Errorf("programm must be launched inside a git repository")
		}
	}
	return wd, nil
}

func parseRewordParams(wd string, content string) ([]rewordParam, error) {
	res := make([]rewordParam, 0)
	lines := strings.Split(content, "\n")
	for _, v := range lines {
		if len(v) < 1 {
			continue
		}
		rr := strings.SplitN(v, " ", 2)
		if len(rr) != 2 {
			return nil, fmt.Errorf("%s has wrong format", v)
		}
		commit, err := parseCommit(wd, rr[0])
		if err != nil {
			return nil, err
		}
		res = append(res, rewordParam{commit, rr[1]})
	}
	return res, nil
}

func main() {
	wd, err := getRepoRoot()
	if err != nil {
		exitWithError("error while getting repo root: %s", err.Error())
	}

	filePtr := flag.String("file", "", "path to file in format \"<hash> <new_message>\n\"")
	dateOptimizationPtr := flag.Bool("date", false, "optimize graph building using commit dates. Use it with caution, if invariant 'date(child) > date(parent)' is broken for at least one pair (parent, child), program can behave undefined")
	flag.Parse()
	filePath := *filePtr
	dateOptimization := *dateOptimizationPtr
	flagOffset := 0
	if dateOptimization {
		flagOffset++
	}

	params := make([]rewordParam, 0)
	if filePath != "" {
		f, err := os.Open(filePath)
		if err != nil {
			exitWithError("error while opening reword file: %s", err.Error())
		}
		cont, err := ioutil.ReadAll(f)
		if err != nil {
			exitWithError("error while opening reword file: %s", err.Error())
		}
		f.Close()
		params, err = parseRewordParams(wd, string(cont))
		if err != nil {
			exitWithError("error while parsing reword file: %s", err.Error())
		}
	} else {
		if len(os.Args) != 3+flagOffset {
			fmt.Fprint(os.Stderr, "command line arguments are invalid\n")
			os.Exit(1)
		}
		commit, err := parseCommit(wd, os.Args[flagOffset+1])
		if err != nil {
			exitWithError("error while getting commit hash %s: %s", os.Args[1], err.Error())
		}
		newMessage := os.Args[flagOffset+2]
		params = []rewordParam{{commit, newMessage}}
	}

	if err = fastReword(wd, params, dateOptimization); err != nil {
		exitWithError("error during fast reword: %s", err.Error())
	}
}

func exitWithError(format string, a ...interface{}) {
	if format[len(format)-1] != '\n' {
		format += "\n"
	}
	fmt.Fprintf(os.Stderr, format, a...)
	os.Exit(0)
}
