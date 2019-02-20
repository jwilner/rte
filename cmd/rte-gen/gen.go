package main

import (
	"bytes"
	"fmt"
	"go/format"
	"io"
	"log"
	"os"
	"sort"
	"strings"
)

type (
	paramType struct {
		Name    string
		Initial string
		Conv    [2]string
		Imports []string
		Desc    string
	}

	sig struct {
		FuncName string
		TypeName string
		Params   []param
	}

	param struct {
		Name string
		Type paramType
	}

	// a group of the same type
	paramGroup struct {
		Names []string
		Type  paramType
	}
)

var (
	baseParamTypes = []paramType{
		{
			Initial: "s",
			Name:    "string",
			Desc:    "string",
		},
		{
			Initial: "i",
			Name:    "int64",
			Conv:    [2]string{"strconv.ParseInt(", ", 10, 64)"},
			Imports: []string{"strconv"},
			Desc:    "base-10, max-64 bit integer",
		},
		{
			Initial: "h",
			Name:    "int64",
			Conv:    [2]string{"strconv.ParseInt(", ", 16, 64)"},
			Imports: []string{"strconv"},
			Desc:    "hex, max-64 bit integer",
		},
		{
			Initial: "u",
			Name:    "uint64",
			Conv:    [2]string{"strconv.ParseUint(", ", 10, 64)"},
			Imports: []string{"strconv"},
			Desc:    "base-10, max-64 bit unsigned integer",
		},
	}
)

func (s sig) ParamGroups() []paramGroup {
	var grps []paramGroup
	var cur paramGroup
	for _, p := range s.Params {
		if cur.Type.Name == "" {
			cur.Type = p.Type
		}

		if cur.Type.Name != p.Type.Name {
			grps = append(grps, cur)
			cur = paramGroup{Type: p.Type}
		}

		cur.Names = append(cur.Names, p.Name)
	}
	return append(grps, cur)
}

func main() {
	sigs := generateDefaultSigs()

	if err := writeFunctionFile(os.Stdout, sigs); err != nil {
		log.Fatal(err)
	}

	if err := writeTestFile(nil, sigs); err != nil {
		log.Fatal(err)
	}
}

func generateDefaultSigs() []sig {
	const maxParams = 2

	// numSignatures = sum(len(baseParamTypes) ** i for i in range(1, maxParams + 1))
	var sigs []sig
	for _, pTypes := range product(baseParamTypes, maxParams) {
		if len(pTypes) == 0 { // we don't autogenerate the zero func
			continue
		}

		var s sig
		var nameParts []string
		{
			type count struct {
				Initial string
				Number  int
			}
			c := count{Initial: pTypes[0].Initial}
			for i, p := range pTypes {
				if c.Initial != p.Initial {
					nameParts = append(nameParts, fmt.Sprintf("%v%d", c.Initial, c.Number))
					c = count{Initial: p.Initial}
				}
				c.Number++

				s.Params = append(s.Params, param{Name: fmt.Sprintf("%s%d", c.Initial, i), Type: p})
			}
			nameParts = append(nameParts, fmt.Sprintf("%v%d", c.Initial, c.Number))
		}
		s.FuncName = fmt.Sprintf("Func%v", strings.ToUpper(strings.Join(nameParts, "")))
		s.TypeName = fmt.Sprintf("func%v", strings.ToUpper(strings.Join(nameParts, "")))

		sigs = append(sigs, s)
	}
	return sigs
}


func writeFormatted(bs []byte, w io.Writer) error {
	var err error
	bs, err = format.Source(bs)
	if err != nil {
		return fmt.Errorf("format.Source: %v", err)
	}

	if _, err := w.Write(bs); err != nil {
		return fmt.Errorf("os.Stdout.Write: %v", err)
	}

	return nil
}

func product(choices []paramType, maxLength int) [][]paramType {
	var results [][]paramType

	last := [][]paramType{{}}
	for i := 0; i < maxLength; i++ {
		copies := make([][]paramType, 0, len(last)*len(choices))
		for _, arr := range last {
			for _, choice := range choices {
				arrCopy := make([]paramType, len(arr)+1)
				copy(arrCopy, arr)
				arrCopy[len(arr)] = choice
				copies = append(copies, arrCopy)
			}
		}
		results = append(results, last...)
		last = copies
	}

	return append(results, last...)
}
