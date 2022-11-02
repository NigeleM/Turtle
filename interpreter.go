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

func evalExpression(str string) string {

	inString := 0
	newShow := str
	newString := ""
	// check if the string has any commas outside string or if it's a concat

	isConcat := isConcatExp(str)
	// fmt.Println("str coming into function : ", str)
	if isConcat == true {
		parseToken := str[0:strings.LastIndex(str, ".")]
		// parseToken = strings.ReplaceAll(parseToken, "\\n", "\n")
		return strings.ReplaceAll(eval(parseToken), "\\n", "\n")
		// return eval(parseToken)
	} else {
		place := 0
		arg := ""
		for count := 0; count <= strings.LastIndex(str, "."); count++ {
			arg = ""
			if string(newShow[count]) == "\"" {
				inString += 1
				if inString > 1 {
					inString = 0
				}
				continue
			} else if string(newShow[count]) == "," && inString == 0 {
				arg += string(eval(newShow[place:count]))
				for i := range arg {
					if string(arg[i]) == "\"" {
						continue
					} else {
						newString += string(arg[i])
					}
				}
				place = count + 1
			} else if count == strings.LastIndex(str, ".") {
				if newShow[place:count] == " " {
					continue
				} else {
					arg = string(eval(newShow[place:count]))
					for i := range arg {
						if string(arg[i]) == "\"" {
							continue
						} else {
							newString += string(arg[i])
						}
					}
				}
			}
		}
	}

	return strings.ReplaceAll(newString, "\\n", "\n")
}

func getVariable(str string) string {
	s := str
	instring := 0
	varStat := ""
	varTok := ""
	for i := range s {
		if string(s[i]) == "\"" {
			instring += 1
			if instring > 1 {
				instring = 0
			}
		} else if s[i] >= 48 && s[i] <= 57 && instring == 0 {
			varStat += string(s[i])
		} else if string(s[i]) == "." || string(s[i]) == minus || string(s[i]) == plus || string(s[i]) == divide ||
			string(s[i]) == multiply || string(s[i]) == "," || string(s[i]) == " " || string(s[i]) == ")" || string(s[i]) == "(" && instring == 0 {
			continue
		} else if s[i] >= 65 && s[i] <= 90 && instring == 0 || s[i] == 95 && instring == 0 {
			varTok += string(s[i])
		} else if s[i] >= 97 && s[i] <= 122 && instring == 0 || s[i] == 95 && instring == 0 {
			varTok += string(s[i])
		} else {
			continue
		}
	}
	return varTok
}

func isOneVariable(str string) bool {
	inString := 0
	newShow := str
	oneStatement := true
	for count := 0; count < strings.LastIndex(str, "."); count++ {
		if string(newShow[count]) == "\"" {
			inString += 1
			if inString > 1 {
				inString = 0
			}
			continue
		} else if inString == 0 && string(newShow[count]) == "," || string(str[count]) == plus || string(str[count]) == minus || string(str[count]) == divide || string(str[count]) == multiply {
			oneStatement = false
			break
		} else {
			continue
		}
	}
	return oneStatement
}

func isConcatExp(str string) bool {
	inString := 0
	newShow := str
	isConcat := false
	for count := 0; count < strings.LastIndex(str, "."); count++ {
		if string(newShow[count]) == "\"" {
			inString += 1
			if inString > 1 {
				inString = 0
			}
			continue
		} else if inString == 0 && string(newShow[count]) == "," || string(str[count]) == minus || string(str[count]) == divide || string(str[count]) == multiply {
			isConcat = false
			break
		} else if inString == 0 && string(str[count]) == plus {
			isConcat = true
		} else {
			continue
		}
	}
	return isConcat
}

func evalVarExpression(str string) string {
	inString := 0
	newShow := str
	newString := ""
	// check if the string has any commas outside string or if it's a concat
	isConcat := false
	oneStatement := true
	// fmt.Println(str, " STR : ")

	oneStatement = isOneVariable(str)
	isConcat = isConcatExp(str)

	if isConcat == true {
		parseToken := str
		place := 0
		for count := 0; count <= strings.LastIndex(str, "."); count++ {
			if string(str[count]) == "\"" {
				inString += 1
				if inString > 1 {
					inString = 0
				}
				continue
			} else if string(str[count]) == plus && inString == 0 {
				variable := getVariable(parseToken[place:count])
				parseToken = strings.ReplaceAll(parseToken, variable, variableDict[variable])
				place = count + 1
			} else if count == strings.LastIndex(str, ".") {
				variable := getVariable(parseToken[place:count])
				parseToken = strings.ReplaceAll(parseToken, variable, variableDict[variable])
			}
		}
		parseToken = parseToken[0:strings.LastIndex(parseToken, ".")]
		return strings.ReplaceAll(eval(parseToken), "\\n", "\n")
	} else if oneStatement == true {
		variable := getVariable(str[0:strings.LastIndex(str, ".")])
		return variableDict[variable]
	} else {

		place := 0
		arg := ""
		newExp := ""
		for count := 0; count <= strings.LastIndex(str, "."); count++ {
			arg = ""
			newExp = ""
			if string(newShow[count]) == "\"" {
				inString += 1
				if inString > 1 {
					inString = 0
				}
				continue
			} else if string(newShow[count]) == "," && inString == 0 {
				arg = ""
				newExp = ""
				if evalType(newShow[place:count]) == "Var" {
					oneVar := isOneVariable(newShow[place:count])
					if oneVar == true {
						variable := getVariable(newShow[place:count])
						arg = variableDict[variable]
						newString += parseString(arg)
					} else {
						newPlace := 0
						InnerinString := 0
						anotherPlace := place
						newExp = newShow[place:count]
						for newPlace = place; newPlace < count; newPlace++ {
							if string(newShow[count]) == "\"" {
								InnerinString += 1
								if InnerinString > 1 {
									InnerinString = 0
								}
								continue
							} else if InnerinString == 0 && string(str[newPlace]) == plus || string(str[newPlace]) == minus || string(str[newPlace]) == divide || string(str[newPlace]) == multiply {
								variable := getVariable(newShow[anotherPlace:newPlace])
								arg = variableDict[variable]
								newExp = strings.ReplaceAll(newExp, variable, arg)
								anotherPlace = newPlace
							} else {
								continue
							}
						}
						arg = eval(newExp)
						newString += parseString(arg)
						place = count + 1
					}

				} else {
					if evalType(newShow[place:count]) == "Exp" {
						arg = evalExpression(newShow[place:count])
						newString += parseString(arg)
					} else {
						arg = eval(newShow[place:count])
						newString += parseString(arg)
					}
				}

			} else if count == strings.LastIndex(str, ".") {
				arg = ""
				newExp = ""
				if evalType(newShow[place:count]) == "Var" {
					oneVar := isOneVariable(newShow[place:count])
					if oneVar == true {
						variable := getVariable(newShow[place:count])
						arg = variableDict[variable]
						newString += parseString(arg)
					} else {
						newPlace := 0
						InnerinString := 0
						anotherPlace := place
						newExp = newShow[place:count]
						for newPlace = place; newPlace <= count; newPlace++ {
							if string(newShow[count]) == "\"" {
								InnerinString += 1
								if InnerinString > 1 {
									InnerinString = 0
								}
								continue
							} else if InnerinString == 0 && string(str[newPlace]) == plus || string(str[newPlace]) == minus || string(str[newPlace]) == divide || string(str[newPlace]) == multiply {
								variable := getVariable(newShow[anotherPlace:newPlace])
								arg = variableDict[variable]
								newExp = strings.ReplaceAll(newExp, variable, arg)
								anotherPlace = newPlace
							} else if InnerinString == 0 && count == strings.LastIndex(str, ".") {
								variable := getVariable(newShow[anotherPlace:newPlace])
								arg = variableDict[variable]
								newExp = strings.ReplaceAll(newExp, variable, arg)
								anotherPlace = newPlace

							} else {
								continue
							}

						}
						arg = eval(newExp)
						newString += parseString(arg)
					}

				} else {
					if evalType(newShow[place:count]) == "Exp" {
						arg = evalExpression(newShow[place:count])
						newString += parseString(arg)
					} else {
						arg = eval(newShow[place:count])
						newString += parseString(arg)
					}
				}

			}
		}
	}
	return strings.ReplaceAll(newString, "\\n", "\n")
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
	parseToken = strings.ReplaceAll(parseToken, "\\n", "\n")
	if evalType(parseToken) == "Int" {
		fmt.Println(eval(parseToken))
	} else if evalType(parseToken) == "Float" {
		fmt.Println(eval(parseToken))
	} else if evalType(parseToken) == "String" {
		fmt.Println(parseString(eval(parseToken)))
	} else if evalType(parseToken) == "Var" {
		fmt.Println(evalVarExpression(showTok[1]))
	} else if evalType(parseToken) == "Exp" {
		fmt.Println(evalExpression(showTok[1]))
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

		tok := scanner.Text()
		if len(tok) == 0 {
			continue
		}

		if strings.Contains(tok, "//") && strings.Contains(tok, "\"") && strings.Index(tok, "//") < strings.Index(tok, "\"") {
			continue
		}

		if strings.Contains(tok, "//") {
			comments := strings.SplitAfter(tok, "//")
			if strings.Contains(comments[0], "//") {
				continue
			}
		}

		if strings.Contains(tok, "show") {
			showTok := strings.SplitAfter(tok, "show")
			if strings.Contains(showTok[0], "show") {
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
}
