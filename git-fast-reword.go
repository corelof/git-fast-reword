package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	git "github.com/libgit2/git2go/v28"
)

type rewordParam struct {
	hash    string
	message string
}

type logger struct {
	Verbose bool
}

func (l *logger) Write(b []byte) (int, error) {
	if l.Verbose {
		return fmt.Fprint(os.Stdout, string(b))
	}
	return 0, nil
}

// As program runs inside of a repo, function goes up by a directory tree until it meets .git directory
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

func parseRewordParams(repo *git.Repository, content string) ([]rewordParam, error) {
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
		commit, err := parseCommit(repo, rr[0])
		if err != nil {
			return nil, err
		}
		res = append(res, rewordParam{commit, rr[1]})
	}
	return res, nil
}

func main() {
	filePtr := flag.String("file", "",
		"Path to file containing reword options in format \"<hash> <new_message>\\n\"",
	)
	dateOptimizationPtr := flag.Bool("date-optimization", false,
		"Optimize graph building using commit dates. Use it with caution, it can damage your "+
			"repo if invariant 'date(child) > date(parent)' is broken for at least one pair (parent, child)",
	)
	headOnlyPtr := flag.Bool("head-only", false,
		"Reword only commit chain that current HEAD points to. If any rewordable commit is not reachable from HEAD, returns an error",
	)
	verbosePtr := flag.Bool("verbose", false,
		"Verbose logging",
	)
	helpPtr := flag.Bool("help", false,
		"Print this help",
	)
	flag.Parse()

	filePath := *filePtr
	dateOptimization := *dateOptimizationPtr
	headOnly := *headOnlyPtr

	if *helpPtr {
		printHelpAndExit()
	}

	log.SetOutput(&logger{*verbosePtr})

	wd, err := getRepoRoot()
	if err != nil {
		exitWithError("error while getting repo root: %s", err.Error())
	}
	repo, err := git.OpenRepository(wd)
	if err != nil {
		exitWithError("error while opening repository: %s", err.Error())
	}

	flagOffset := 0
	if *verbosePtr {
		flagOffset++
	}
	if dateOptimization {
		flagOffset++
	}
	if headOnly {
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
			exitWithError("error while reading reword file: %s", err.Error())
		}
		f.Close()
		params, err = parseRewordParams(repo, string(cont))
		if err != nil {
			exitWithError("error while parsing reword file: %s", err.Error())
		}
	} else {
		if len(os.Args) != 3+flagOffset {
			exitWithError("command line arguments are invalid, run program with -help flag to get additional info\n")
		}
		commit, err := parseCommit(repo, os.Args[flagOffset+1])
		if err != nil {
			exitWithError("error while getting commit hash %s: %s", os.Args[1], err.Error())
		}
		newMessage := os.Args[flagOffset+2]
		params = []rewordParam{{commit, newMessage}}
	}

	if err = fastReword(repo, params, dateOptimization, headOnly); err != nil {
		exitWithError("error during fast reword: %s", err.Error())
	}
	log.Println("Reworded successfuly")
}

func exitWithError(format string, a ...interface{}) {
	if format[len(format)-1] != '\n' {
		format += "\n"
	}
	fmt.Fprintf(os.Stderr, format, a...)
	os.Exit(1)
}

func printHelpAndExit() {
	flag.Usage()
	fmt.Println("Format:")
	fmt.Println("./git-fast-reword [-date-optimization] [-head-only] -file <path>")
	fmt.Println("or")
	fmt.Println("./git-fast-reword [-date-optimization] [-head-only] <commit hash | position relative to the HEAD> <new message>")
	os.Exit(0)
}
