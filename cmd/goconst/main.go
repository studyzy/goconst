package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/jgautheron/goconst"
)

const usageDoc = `goconst: find repeated strings that could be replaced by a constant

Usage:

  goconst ARGS <directory>

Flags:

  -ignore            exclude files matching the given regular expression
  -ignore-tests      exclude tests from the search (default: true)
  -min-occurrences   report from how many occurrences (default: 2)
  -match-constant    look for existing constants matching the strings
  -numbers           search also for duplicated numbers
  -min          	   minimum value, only works with -numbers
  -max          	   maximum value, only works with -numbers
  -output            output formatting (text or json)

Examples:

  goconst ./...
  goconst -ignore "yacc|\.pb\." $GOPATH/src/github.com/cockroachdb/cockroach/...
  goconst -min-occurrences 3 -output json $GOPATH/src/github.com/cockroachdb/cockroach
  goconst -numbers -min 60 -max 512 .
`

var (
	flagIgnore         = flag.String("ignore", "", "ignore files matching the given regular expression")
	flagIgnoreTests    = flag.Bool("ignore-tests", true, "exclude tests from the search")
	flagMinOccurrences = flag.Int("min-occurrences", 2, "report from how many occurrences")
	flagMatchConstant  = flag.Bool("match-constant", false, "look for existing constants matching the strings")
	flagNumbers        = flag.Bool("numbers", false, "search also for duplicated numbers")
	flagMin            = flag.Int("min", 0, "minimum value, only works with -numbers")
	flagMax            = flag.Int("max", 0, "maximum value, only works with -numbers")
	flagOutput         = flag.String("output", "text", "output formatting")
)

func main() {
	flag.Usage = func() {
		fmt.Fprint(os.Stderr, usage)
	}
	flag.Parse()
	log.SetPrefix("goconst: ")

	args := flag.Args()
	if len(args) != 1 {
		usage()
	}
	path := args[0]

	gco := goconst.New(
		path,
		*flagIgnore,
		*flagIgnoreTests,
		*flagMatchConstant,
		*flagNumbers,
	)
	strs, consts, err := gco.ParseTree()
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	printOutput(strs, consts, *flagOutput, *flagMinOccurrences, *flagMin, *flagMax)
}

func usage() {
	fmt.Fprintf(os.Stderr, usageDoc)
	os.Exit(1)
}

func printOutput(strs goconst.Strings, consts goconst.Constants, output string, minOccurrences, min, max int) {
	for str, item := range strs {
		// Filter out items whose occurrences don't match the min value
		if len(item) < minOccurrences {
			delete(strs, str)
		}

		// If the value is a number
		if i, err := strconv.Atoi(str); err == nil {
			if min != 0 && i < min {
				delete(strs, str)
			}
			if max != 0 && i > max {
				delete(strs, str)
			}
		}
	}

	switch output {
	case "json":
		enc := json.NewEncoder(os.Stdout)
		err := enc.Encode(struct {
			Strings   goconst.Strings   `json:"strings,omitEmpty"`
			Constants goconst.Constants `json:"constants,omitEmpty"`
		}{
			strs, consts,
		})
		if err != nil {
			log.Fatal(err)
		}
	case "text":
		for str, item := range strs {
			for _, xpos := range item {
				fmt.Printf(
					`%s:%d:%d:%d other occurrence(s) of "%s" found in: %s`,
					xpos.Filename,
					xpos.Line,
					xpos.Column,
					len(item)-1,
					str,
					occurrences(item, xpos),
				)
				fmt.Print("\n")
			}

			if len(consts) == 0 {
				continue
			}
			if cst, ok := consts[str]; ok {
				// const should be in the same package and exported
				fmt.Printf(`A matching constant has been found for "%s": %s`, str, cst.Name)
				fmt.Printf("\n\t%s\n", cst.String())
			}
		}
	default:
		fmt.Printf(`Unsupported output format: %s`, output)
	}
}

func occurrences(item []goconst.ExtendedPos, current goconst.ExtendedPos) string {
	occurrences := []string{}
	for _, xpos := range item {
		if xpos == current {
			continue
		}
		occurrences = append(occurrences, fmt.Sprintf(
			"%s:%d:%d", xpos.Filename, xpos.Line, xpos.Column,
		))
	}
	return strings.Join(occurrences, " ")
}
