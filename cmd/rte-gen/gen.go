package main

import (
	"flag"
	"fmt"
	"go/format"
	"io"
	"log"
	"os"
)

var (
	output     = flag.String("output", "", "where to write the generated code")
	testOutput = flag.String("test-output", "", "where to write the generated tests")
)

type Signature struct {
	Name  string
	Arr   bool
	Count int
}

func (s Signature) PNames() []string {
	var ns []string
	for i := 0; i < s.Count; i++ {
		ns = append(ns, fmt.Sprintf("p%d", i))
	}
	return ns
}

func main() {
	flag.Parse()

	if *output == "" && *testOutput == "" {
		log.Fatalln("Output and/or test output must be provided")
	}

	sigs := generateDefaultSigs()

	if *output != "" {
		o := os.Stdout
		if *output != "-" {
			var err error
			if o, err = os.Create(*output); err != nil {
				log.Fatal(err)
			}
			defer func() {
				_ = o.Close()
			}()
		}

		if err := writeFunctionFile(o, sigs); err != nil {
			log.Fatalf("failed writing output file: %v", err)
		}
	}

	if *testOutput != "" {
		tO := os.Stdout
		if *testOutput != "-" {
			var err error
			if tO, err = os.Create(*testOutput); err != nil {
				log.Fatal(err)
			}
			defer func() {
				_ = tO.Close()
			}()
		}

		if err := writeTestFile(tO, sigs); err != nil {
			log.Fatalf("failed writing test file: %v", err)
		}
	}
}

func generateDefaultSigs() []Signature {
	signatures := []Signature{
		{Name: "Func"},
	}
	for i := 1; i < 5; i++ {
		signatures = append(signatures, Signature{Name: fmt.Sprintf("Func%d", i), Count: i})
	}
	for i := 5; i < 8+1; i++ {
		signatures = append(signatures, Signature{Name: fmt.Sprintf("Func%d", i), Count: i, Arr: true})
	}
	return signatures
}

func writeFormatted(bs []byte, w io.Writer) error {
	if _, ok := os.LookupEnv("SKIP_FORMAT"); !ok {
		var err error
		bs, err = format.Source(bs)
		if err != nil {
			return fmt.Errorf("format.Source: %v", err)
		}
	}

	if _, err := w.Write(bs); err != nil {
		return fmt.Errorf("os.Stdout.Write: %v", err)
	}

	return nil
}
