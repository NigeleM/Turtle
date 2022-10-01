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

func containsStr(s string, str string) bool {
	for i := range s {
		if string(s[i]) == str {
			return true
		}
	}
	return false
}

func show(tokens string) {

	checktoken := strings.Split(tokens, " ")
	if "show" == checktoken[0] && contains(checktoken, ".") == true {
		showSet := strings.SplitAfterN(tokens, "show", 2)
		for i := range showSet {
			if i == 0 && showSet[i] == "show" {
				continue
			} else {
				//fmt.Println("my set : ", showSet)
				newtok := ""
				if showSet[i] == "show" {
					continue
				} else if containsStr(showSet[i], "show") == true || containsStr(showSet[i], "show ") == true {
					continue
				} else if containsStr(showSet[i], ".") {
					if strings.LastIndex(showSet[i], ".") > 0 {
						newShow := showSet[i]
						inString := 0
						for count := 0; count < strings.LastIndex(showSet[i], "."); count++ {
							// fmt.Println("old token -->", newtok)
							if string(newShow[count]) == "\"" {
								inString += 1
								if inString > 1 {
									inString = 0
								}
								continue
							} else if string(newShow[count]) == "+" && inString == 0 {
								continue

							} else if string(newShow[count]) == " " && inString == 0 {
								continue

							} else {
								newtok += string(newShow[count])
							}
						}
					}
					fmt.Println(newtok)
				}
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
