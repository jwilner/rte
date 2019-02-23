package main

import (
	"fmt"
	"go/format"
	"io"
	"log"
	"os"
)

type Signature struct {
	Name   string
	ArrLen int
	Params []string
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

func generateDefaultSigs() []Signature {
	signatures := []Signature{
		{Name: "Func"},
	}
	for i := 1; i < 5; i++ {
		s := Signature{Name: fmt.Sprintf("Func%d", i)}
		for j := 0; j < i; j++ {
			s.Params = append(s.Params, fmt.Sprintf("p%d", j))
		}
		signatures = append(signatures, s)
	}
	for i := 5; i < 8+1; i++ {
		signatures = append(signatures, Signature{Name: fmt.Sprintf("Func%d", i), ArrLen: i})
	}
	return signatures
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
