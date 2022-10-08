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
	return tv.Value.String()

}

func evalType(s string) string {

	fs := token.NewFileSet()
	tv, err := types.Eval(fs, nil, token.NoPos, s)
	if err != nil {
		return "Var"
	}

	return tv.Value.Kind().String()

}

var variableDict = make(map[string]string)

func insertVariable(variableToken string) {
	newtoken := variableToken
	varToken := strings.Split(newtoken, "=")
	// fmt.Println("Variable type : ", evalType(varToken[1]), "  : variable value : ", varToken[1])
	varToken[0] = strings.ReplaceAll(varToken[0], " ", "")
	if evalType(varToken[1]) == "String" {
		variableDict[string(varToken[0])] = eval(varToken[1])
	} else if evalType(varToken[1]) == "Int" {
		variableDict[string(varToken[0])] = eval(varToken[1])
	} else if evalType(varToken[1]) == "Float" {
		variableDict[string(varToken[0])] = eval(varToken[1])
	} else if evalType(varToken[1]) == "Var" {

		// fmt.Println("----->", varToken[1])
		// variable := ""
		oldvar := varToken[1]
		newVarTok := strings.Split(oldvar, " ")
		for v := range newVarTok {
			// fmt.Println(newVarTok[v])

			_, isPresent := variableDict[newVarTok[v]]
			// fmt.Println(isPresent)
			if isPresent {
				// fmt.Println(variableDict[newVarTok[v]])
				//variable += variableDict[newVarTok[v]]
				newVarTok[v] = variableDict[newVarTok[v]]
			}
		}
		VarTok := strings.Join(newVarTok, " ")
		variableDict[string(varToken[0])] = eval(VarTok)
	}
	// fmt.Println("Dictionary --->", variableDict)

}

// variableDict[string(varToken[0])] = eval(varToken[1])

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

func show(str string) {
	//fmt.Println(str)
	showTok := strings.SplitAfterN(str, "show", 2)
	//fmt.Println(showTok[1])
	if strings.LastIndex(showTok[1], ".") > -1 && strings.Index(showTok[1], "\"") > -1 {
		newShow := showTok[1]
		newShowCheck := showTok[1][0:strings.LastIndex(showTok[1], ".")]
		// Strings and Concatenation
		if evalType(newShowCheck) == "String" {
			expression := ""
			newtok := ""
			inString := 0
			for count := 0; count < strings.LastIndex(showTok[1], "."); count++ {
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
				} else if inString == 0 {
					expression += string(newShow[count])
				} else if inString == 1 {
					newtok += string(newShow[count])
				} else {
					newtok += string(newShow[count])
				}

			}

			newtok = strings.ReplaceAll(newtok, "\\n", "\n")
			fmt.Println(newtok)
		} else {
			expression := ""
			newtok := ""
			inString := 0
			for count := 0; count <= strings.LastIndex(showTok[1], "."); count++ {
				if string(newShow[count]) == "\"" {
					inString += 1
					if inString > 1 {
						inString = 0
					}
					continue

				} else if count <= strings.LastIndex(showTok[1], ".") && string(newShow[count]) == "." {
					if len(expression) > 0 {
						newtok += eval(expression)
						expression = ""
					}
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
				} else if inString == 0 {
					expression += string(newShow[count])
				} else if inString == 1 {
					newtok += string(newShow[count])
				} else if count <= strings.LastIndex(showTok[1], ".") {
					if len(expression) > 0 {
						newtok += eval(expression)
						expression = ""
					}
				} else {
					newtok += string(newShow[count])
				}

			}

			newtok = strings.ReplaceAll(newtok, "\\n", "\n")
			fmt.Println(newtok)

		}

	} else if evalType(showTok[1]) == "Int" {
		fmt.Println(eval(showTok[1]))
	} else if evalType(showTok[1]) == "Float" {
		fmt.Println(eval(showTok[1]))
	} else if evalType(showTok[1]) == "Var" {
		vars := strings.Split(showTok[1], " ")
		for v := range vars {
			// fmt.Println(newVarTok[v])

			_, isPresent := variableDict[vars[v]]
			// fmt.Println(isPresent)
			if isPresent {
				// fmt.Println(variableDict[newVarTok[v]])
				//variable += variableDict[newVarTok[v]]
				vars[v] = variableDict[vars[v]]
			}
		}
		newVar := strings.Join(vars, " ")
		newVar = newVar[0:strings.LastIndex(newVar, ".")]
		fmt.Println(eval(newVar))

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
