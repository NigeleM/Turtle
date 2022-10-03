package main

import (
	"bufio"
	"fmt"
	"go/token"
	"go/types"
	"os"
	"strings"
)

// eval function
// used for evaluation expression

func eval(s string) string {

	fs := token.NewFileSet()
	tv, err := types.Eval(fs, nil, token.NoPos, s)
	if err != nil {
		panic(err)
	}
	//println("token---:", tv.Value.String())
	// println("token---:", tv.Value.String())
	// println("token---type:", tv.Value.Kind().String())
	// newInt, err := strconv.Atoi(tv.Value.String())

	// intVar := 0
	// for i := 0; i <= newInt; i++ {
	// 	if i == newInt {
	// 		intVar = i
	// 	}
	// }
	//fmt.Println("intVar : ", intVar, "string version : ", string(intVar))
	// return strconv.FormatInt(int64(intVar), 10)
	return tv.Value.String()

}

var variableDict = make(map[string]string)

func insertVariable(variableToken string) {
	newtoken := variableToken
	varToken := strings.Split(newtoken, "=")
	variableDict[string(varToken[0])] = eval(varToken[1])
	fmt.Println(varToken, newtoken, variableDict, eval(varToken[1]))

}

func indexList(s []int, str string) []int {
	for i := range str {
		if string(str[i]) == "," {
			s = append(s, i)
		}

	}

	return s

}

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
				expression := ""
				if showSet[i] == "show" {
					continue
				} else if containsStr(showSet[i], "show") == true || containsStr(showSet[i], "show ") == true {
					continue
				} else if containsStr(showSet[i], ".") {
					if strings.LastIndex(showSet[i], ".") > 0 {
						newShow := showSet[i]
						inString := 0
						for count := 0; count < strings.LastIndex(showSet[i], "."); count++ {
							if string(newShow[count]) == "\"" {
								inString += 1
								if inString > 1 {
									inString = 0
								}
								continue
							} else if string(newShow[count]) == " " && inString == 0 {
								continue

							} else if string(newShow[count]) == "+" && inString == 0 {
								expression += string(newShow[count])
								continue

							} else if string(newShow[count]) == "," && inString == 0 {
								if len(expression) > 0 {
									newtok += eval(expression)
									expression = ""

								} else {
									continue
								}
								//continue

							} else if inString == 0 {
								expression += string(newShow[count])
							} else if inString == 1 {
								newtok += string(newShow[count])
							} else {
								newtok += string(newShow[count])
							}

						}
					}
					newtok = strings.ReplaceAll(newtok, "\\n", "\n")
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
		println(tv.Value.String())

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
		//fmt.Println(tok)
		if len(tok) == 0 {
			continue
		}

		if strings.Contains(tok, "//") && strings.Contains(tok, "\"") && strings.Index(tok, "//") < strings.Index(tok, "\"") {
			//comments := strings.SplitAfter(tok, "//")
			//fmt.Println(comments)
			continue
		}

		if strings.Contains(tok, "//") {
			comments := strings.SplitAfter(tok, "//")
			//fmt.Println(comments)
			if strings.Contains(comments[0], "//") {
				continue
			}
		}

		if strings.Contains(tok, "show") {
			showTok := strings.SplitAfter(tok, "show")
			if strings.Contains(showTok[0], "show") {
				show(tok)
			}
		}

		if strings.Contains(tok, "=") {
			varTok := strings.SplitAfter(tok, "=")
			if strings.Contains(varTok[0], "=") {
				insertVariable(tok)
			}
		}

	}

	// fmt.Println("hello world")
	// show("hello world")
	// show("1 + 2")

}
