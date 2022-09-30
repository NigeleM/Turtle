package main

import (
	"bufio"
	"fmt"
	"go/token"
	"go/types"
	"os"
	"strings"
)

func contains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}
	return false
}

func show(tokens string) {

	//fmt.Println(tokens)
	checktoken := strings.Split(tokens, " ")
	showSet := strings.Split(tokens, "show ")
	if "show" == checktoken[0] && contains(checktoken, ".") == true {
		// String logic
		//fmt.Println(showSet)
		for i := range showSet {
			if showSet[i] == "" {
				continue
			} else {
				fmt.Println("set:", i, showSet[i])
			}
		}
	} else {

		//fmt.Println(checktoken)
		tokens = strings.Trim(tokens, "show")
		fs := token.NewFileSet()
		tv, err := types.Eval(fs, nil, token.NoPos, tokens)
		if err != nil {
			panic(err)
		}
		println("token : ", tv.Value.String())

	}

}

func check(err error) {
	if err != nil {
		fmt.Println(err)
		return
	}
}

func main() {

	file, err := os.Open(os.Args[1])
	check(err)

	defer file.Close()

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {

		//fmt.Println(scanner.Text())
		tok := scanner.Text()
		if len(tok) == 0 {
			continue
		} else {
			show(tok)
		}
	}

	// fmt.Println("hello world")
	// show("hello world")
	// show("1 + 2")

}
