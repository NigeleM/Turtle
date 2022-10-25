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

	instring := 0
	variable := -1
	varStat := ""
	varTok := ""
	// fmt.Println(s)
	fs := token.NewFileSet()
	s = strings.ReplaceAll(s, "\\n", "\n")
	tv, err := types.Eval(fs, nil, token.NoPos, s)
	if err != nil {
		for i := range s {
			if string(s[i]) == "\"" {
				instring += 1
				if instring > 1 {
					instring = 0
				}
			} else if s[i] >= 48 && s[i] <= 57 && instring == 0 {
				varStat += string(s[i])
				// continue
			} else if string(s[i]) == "." || string(s[i]) == minus || string(s[i]) == plus || string(s[i]) == divide ||
				string(s[i]) == multiply || string(s[i]) == "," || string(s[i]) == " " || string(s[i]) == ")" || string(s[i]) == "(" && instring == 0 {
				continue
				// if string(s[i]) != " " {
				// 	varStat += string(s[i])
				// }

			} else if s[i] >= 65 && s[i] <= 90 && instring == 0 || s[i] == 95 && instring == 0 {
				variable = 1
				varTok += string(s[i])
			} else if s[i] >= 97 && s[i] <= 122 && instring == 0 || s[i] == 95 && instring == 0 {
				variable = 1
				varTok += string(s[i])
			} else {
				continue
			}
		}

		if variable == 1 {
			// fmt.Println(varStat, "-------------->")
			// tv.Value.Kind().String()
			// fmt.Println("Variable varTok : ", varTok)
			return "Var"
		}
		// fmt.Println("Exp varTok : ", varTok)
		return "Exp"

	}

	return tv.Value.Kind().String()
}

var variableDict = make(map[string]string)

func insertVariable(variableToken string) {
	// fmt.Println(variableDict, variableToken, "<---->")
	newtoken := variableToken
	varToken := strings.Split(newtoken, "=")
	//fmt.Println("Variable type : ", evalType(varToken[1]), "  : variable value : ", varToken[1])
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

// func indexList(s []int, str string) []int {
// 	for i := range str {
// 		if string(str[i]) == "," {
// 			s = append(s, i)
// 		}
// 	}
// 	return s
// }

// func contains(s []string, str string) bool {
// 	for _, v := range s {
// 		if v == str {
// 			return true
// 		}
// 	}
// 	return false
// }

// func containsStr(s string, str string) bool {
// 	for i := range s {
// 		if string(s[i]) == str {
// 			return true
// 		}
// 	}
// 	return false
// }

func evalExpression(str string) string {
	expression := ""
	newtok := ""
	inString := 0
	newShow := str
	newString := ""

	// check if the string has any commas outside string or if it's a concat

	isConcat := true
	for count := 0; count < strings.LastIndex(str, "."); count++ {
		if string(newShow[count]) == "\"" {
			inString += 1
			if inString > 1 {
				inString = 0
			}
			continue
		} else if inString == 0 && string(newShow[count]) == "," {
			isConcat = false
		} else {
			continue
		}
	}

	if isConcat == true {
		parseToken := str[0:strings.LastIndex(str, ".")]
		// parseToken = strings.ReplaceAll(parseToken, "\\n", "\n")
		return strings.ReplaceAll(eval(parseToken), "\\n", "\n")
		// return eval(parseToken)

	} else {
		fmt.Println(str, "series check")
		series := make([]string, 0)
		evalSeries := make([]string, 0)
		for count := 0; count < strings.LastIndex(str, "."); count++ {
			if string(newShow[count]) == "\"" {
				// newtok += string(newShow[count])
				inString += 1
				if inString > 1 {
					// newtok += string(newShow[count])
					series = append(series, newtok)
					evalSeries = append(evalSeries, "String")
					newtok = ""
					inString = 0
				}
				continue
			} else if string(newShow[count]) == " " && inString == 0 {
				continue
			} else if string(newShow[count]) == "," && inString == 0 {
				if len(expression) > 0 {
					series = append(series, expression)
					evalSeries = append(evalSeries, "EXP")
					expression = ""
				} else {
					continue
				}
			} else if inString == 0 {
				expression += string(newShow[count])
			} else if inString == 1 {
				newtok += string(newShow[count])
			} else {
				newtok += string(newShow[count])
			}
		}
		fmt.Println("SERIES : ", series)
		fmt.Println("EVAL SERIES : ", evalSeries)
		for value := range series {
			fmt.Println(series[value])
			if evalSeries[value] == "EXP" {
				newString += string(eval(series[value]))
			} else {
				newString += series[value]
			}
		}
	}

	return strings.ReplaceAll(newString, "\\n", "\n")
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

func parseString(str string) string {
	newString := ""
	for v := range str {
		if string(str[v]) == "\"" {
			continue
		} else {
			newString += string(str[v])
		}
	}
	return newString
}

func showReal(str string) {

	showTok := strings.SplitAfterN(str, "show", 2)
	parseToken := showTok[1][0:strings.LastIndex(showTok[1], ".")]
	// fmt.Println(parseToken, evalType(parseToken), " ")
	parseToken = strings.ReplaceAll(parseToken, "\\n", "\n")
	if evalType(parseToken) == "Int" {
		fmt.Println(eval(parseToken))
	} else if evalType(parseToken) == "Float" {
		fmt.Println(eval(parseToken))
	} else if evalType(parseToken) == "String" {
		fmt.Println(parseString(eval(parseToken)))
	} else if evalType(parseToken) == "Var" {
		//fmt.Println(eval(parseToken))
		fmt.Println(parseToken, "VAR")
	} else if evalType(parseToken) == "Exp" {
		fmt.Println(evalExpression(showTok[1]), "EXP")
		// fmt.Println(parseToken, "EXP")
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
				// fmt.Println("show : ", showTok)
				showReal(tok)
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
