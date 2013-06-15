package main

import (
	"encoding/csv"
	"fmt"
	"github.com/alecthomas/mph"
	"github.com/ogier/pflag"
	"io"
	"os"
	"time"
)

var (
	keyFlag    = pflag.Int("key", 0, "CSV column to use for key")
	valueFlag  = pflag.Int("value", 1, "CSV column to use for value")
	verifyFlag = pflag.Bool("verify", true, "verify CSV map and MPH validity")
)

func main() {
	pflag.Parse()
	input := pflag.Arg(0)
	r, err := os.Open(input)
	if err != nil {
		panic(err)
	}
	cdh := mph.NewCHDBuilder()
	start := time.Now()
	reader := csv.NewReader(r)
	m := map[string]string{}
	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if *verifyFlag {
			if _, ok := m[row[*keyFlag]]; ok {
				panic(fmt.Sprintf("duplicate key %s", row[*keyFlag]))
			}
			m[row[*keyFlag]] = row[*valueFlag]
		}
		cdh.Add([]byte(row[*keyFlag]), []byte(row[*valueFlag]))
	}
	fmt.Printf("Load took %s\n", time.Now().Sub(start))
	start = time.Now()
	h, err := cdh.Build()
	if err != nil {
		panic(err)
	}
	fmt.Printf("Build took %s\n", time.Now().Sub(start))
	if *verifyFlag {
		for k, v := range m {
			if string(h.Get([]byte(k))) != v {
				panic(fmt.Sprintf("MPH did not validate: key %s did not map to value %s", k, v))
			}
		}
		println("validated OK")
	}
}
