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

const (
	plus     = "+"
	minus    = "-"
	divide   = "/"
	multiply = "*"
)

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

		oldvar := varToken[1]
		hold := ""
		for v := range oldvar {
			if string(oldvar[v]) == plus || string(oldvar[v]) == minus || string(oldvar[v]) == multiply || string(oldvar[v]) == divide {
				hold += " "
				hold += string(oldvar[v])
				hold += " "

			} else {
				hold += string(oldvar[v])

			}

		}

		newVarTok := strings.Split(hold, " ")
		for v := range newVarTok {
			_, isPresent := variableDict[newVarTok[v]]
			if isPresent {
				newVarTok[v] = variableDict[newVarTok[v]]
			}
		}
		VarTok := strings.Join(newVarTok, " ")
		variableDict[string(varToken[0])] = eval(VarTok)

	}
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
						if evalType(expression) == "Var" {
							vars := ""
							if len(expression) > 1 {
								// turn this into a function
								for v := range expression {
									if string(expression[v]) == plus || string(expression[v]) == minus || string(expression[v]) == multiply || string(expression[v]) == divide || v <= strings.LastIndex(expression, ".") && string(expression[v]) == "." || string(expression[v]) == " " {
										_, isPresent := variableDict[vars]
										//fmt.Println(vars)
										if isPresent {
											vars += strings.Replace(vars, vars, variableDict[vars], 1)
											vars += string(expression[v])
										}
									} else {
										if string(expression[v]) == " " {
											continue
										} else {
											vars += string(expression[v])
										}
									}
								}
								// fmt.Println(evalType(variableDict[vars]))
								if evalType(variableDict[vars]) == "String" {
									newtok += variableDict[vars][strings.Index(variableDict[vars], "\"")+1 : strings.LastIndex(variableDict[vars], "\"")]
									expression = ""
								} else if evalType(variableDict[vars]) == "Int" {
									newtok += eval(variableDict[vars])
								} else if evalType(variableDict[vars]) == "Float" {
									newtok += eval(variableDict[vars])
								}

								expression = ""
							} else {
								_, isPresent := variableDict[expression]
								if isPresent {
									fmt.Println(variableDict[expression])
									vars = strings.Replace(expression, expression, variableDict[expression], 1)
								}
								newtok += eval(vars)
								expression = ""
							}

						}
					}

				} else if string(newShow[count]) == " " && inString == 0 {
					//expression += string(newShow[count])
					continue

				} else if string(newShow[count]) == "+" && inString == 0 {
					expression += string(newShow[count])
					continue
				} else if string(newShow[count]) == "," && inString == 0 {
					if len(expression) > 0 {
						//fmt.Println("code : ", expression)
						// fmt.Println(evalType(expression))
						if evalType(expression) == "Var" {
							vars := ""
							if len(expression) > 1 {
								for v := range expression {
									if string(expression[v]) == plus || string(expression[v]) == minus || string(expression[v]) == multiply || string(expression[v]) == divide {
										_, isPresent := variableDict[vars]
										// fmt.Println(isPresent)
										if isPresent {
											// fmt.Println(variableDict[newVarTok[v]])
											//variable += variableDict[newVarTok[v]]
											fmt.Println(variableDict[vars])
											vars = strings.Replace(vars, vars, variableDict[vars], 1)

										}
										vars += string(expression[v])
									} else {
										vars += string(expression[v])
										//fmt.Println("vars : ", string(expression[v]), vars)
									}

								}

								expression = vars
								// fmt.Println("EVAL: ")
								newtok += eval(expression)
								expression = ""
							} else {

								_, isPresent := variableDict[expression]
								// fmt.Println(isPresent)
								if isPresent {
									// fmt.Println(variableDict[newVarTok[v]])
									//variable += variableDict[newVarTok[v]]
									fmt.Println(variableDict[expression])
									vars = strings.Replace(expression, expression, variableDict[expression], 1)
								}
								newtok += eval(vars)
								expression = ""
							}
						}
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

		token := showTok[1]
		hold := ""
		for v := range token {
			if string(token[v]) == plus || string(token[v]) == minus || string(token[v]) == multiply || string(token[v]) == divide {
				hold += " "
				hold += string(token[v])
				hold += " "

			} else {
				hold += string(token[v])

			}

		}

		vars := strings.Split(hold, " ")
		// fmt.Println(" Count : ", strings.Count(showTok[1], " "))
		// fmt.Println("var", vars)
		for v := range vars {

			_, isPresent := variableDict[vars[v]]
			// fmt.Println(isPresent)
			if isPresent {
				// fmt.Println(variableDict[newVarTok[v]])
				//variable += variableDict[newVarTok[v]]
				vars[v] = variableDict[vars[v]]

			}
		}

		//token = eval(token)
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
