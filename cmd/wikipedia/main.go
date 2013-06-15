package main

import (
	"bufio"
	"fmt"
	"github.com/alecthomas/mph"
	"io"
	"os"
	"strings"
	"time"
)

func main() {
	rf, err := os.Open("wikipedia.tsv")
	if err != nil {
		panic(err)
	}
	startTime := time.Now()
	r := bufio.NewReader(rf)
	chd := mph.NewCHDBuilder()
	last := ""
	offset := int64(0)
	start := offset
	n := 0
	for {
		line, err := r.ReadString('\n')
		if err == io.EOF {
			break
		}
		offset += int64(len(line))
		line = strings.TrimRight(line, "\n")
		row := strings.Split(line, "\t")
		n++
		if n%10000 == 0 {
			print(".")
		}
		name := row[1]
		if name != last {
			v := fmt.Sprintf("%d", start)
			chd.Add([]byte(last), []byte(v))
			// fmt.Printf("added %s\n", last)
			last = name
			start = offset
		}
	}
	println()
	fmt.Printf("load: %s\n", time.Now().Sub(startTime))
	println("finished")
	startTime = time.Now()
	m, err := chd.Build()
	if err != nil {
		panic(err)
	}
	fmt.Printf("build: %s\n", time.Now().Sub(startTime))
	for i := m.Iterate(); i != nil; i = i.Next() {
		v := i.Get()
		fmt.Printf("%s\n", v.Key())
	}
}
