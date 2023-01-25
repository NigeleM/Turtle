package main

import (
	"bufio"
	"fmt"
	"go/token"
	"go/types"
	"os"
	"strconv"
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

// Eval different types of data in order to parser data easier
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

// global variable dictionary
// There will be a template dictionary for functions as well
var variableDict = make(map[string]string)

// Insert variable into the variable dictionary
// may change this functionality but so far it works well
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
	} else if evalType(varToken[1]) == "Bool" {
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

// call when variable is assigned to a function
func insertFunction(function string, state string) {
	// fmt.Println(state, "state is")
	if state == "isMain" {
		funcVar := strings.SplitAfter(function, "=")
		variableDict[getVariable(funcVar[0])] = funcVar[1]
		functionProtocol(function, state)
	} else {
		funcVar := strings.SplitAfter(function, "=")
		functionDict[state].funcVariableDict[getVariable(funcVar[0])] = funcVar[1]
		// fmt.Println(functionDict, "-- FUNCTION DICT \n\n")
		// fmt.Println(function, "function ---------->\n\n", state)
		// fmt.Println(funcVar[1], " ---------->\n\n", funcVar[0])
		// fmt.Println(functionDict[state].funcVariableDict[getVariable(funcVar[0])])
		functionProtocol(function, state)

	}
}

// Used to evaluate expression and different expression cases
// This is used only for non variable expression
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

// Parses out variables from variable expression
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

// check and see if a statement only has one variable
func isOneVariable(str string) bool {
	inString := 0
	Statement := true
	// fmt.Println(len(str), "--STR--->")
	for count := 0; count <= len(str)-1; count++ {
		if string(str[count]) == "\"" {
			inString += 1
			if inString > 1 {
				inString = 0
			}
		} else if inString == 0 && string(str[count]) == plus || string(str[count]) == minus || string(str[count]) == divide || string(str[count]) == multiply {
			Statement = false
			break
		}
	}

	return Statement
}

// Check and see if a concatExpression is there
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

// Evaluate variable expressions by getting variables and changing
// variable expressions into non variable expressions
func getevalVar(str string) string {
	arg := ""
	newExp := ""
	newString := ""
	place := 0

	newPlace := 0
	InnerinString := 0
	anotherPlace := place
	newExp = str
	// fmt.Println(str, "----eval comma----")
	for newPlace = place; newPlace < len(str)-1; newPlace++ {
		if string(str[newPlace]) == "\"" {
			InnerinString += 1
			if InnerinString > 1 {
				InnerinString = 0
			}
			continue
		} else if InnerinString == 0 && string(str[newPlace]) == plus || string(str[newPlace]) == minus || string(str[newPlace]) == divide || string(str[newPlace]) == multiply {
			variable := getVariable(str[anotherPlace:newPlace])
			arg = variableDict[variable]
			newExp = strings.ReplaceAll(newExp, variable, arg)
			anotherPlace = newPlace
		} else {
			continue
		}
	}

	variable := getVariable(newExp)
	arg = variableDict[variable]
	newExp = strings.ReplaceAll(newExp, variable, arg)
	anotherPlace = newPlace
	// fmt.Println("---", newExp, "---", "NEW EXP")
	arg = eval(newExp)
	// fmt.Println("---", arg, "---", "NEW EVAL")
	newString += parseString(arg)
	return newString

}

// Evaluate variable expressions by getting variables and changing
// variable expressions into non variable expressions
// Evaluate variable expression at end of a statement before period
func getevalVarPeriod(str string) string {
	arg := ""
	newExp := ""
	newString := ""
	place := 0

	newPlace := 0
	InnerinString := 0
	anotherPlace := place
	newExp = str
	for newPlace = place; newPlace <= len(str)-1; newPlace++ {
		arg = ""
		if string(str[newPlace]) == "\"" {
			InnerinString += 1
			if InnerinString > 1 {
				InnerinString = 0
			}
			continue
		} else if InnerinString == 0 && string(str[newPlace]) == plus || string(str[newPlace]) == minus || string(str[newPlace]) == divide || string(str[newPlace]) == multiply {
			variable := getVariable(str[anotherPlace:newPlace])
			arg = variableDict[variable]
			// fmt.Println(variable, arg)
			newExp = strings.ReplaceAll(newExp, variable, arg)
			anotherPlace = newPlace
		} else if InnerinString == 0 && newPlace == len(str)-1 {
			variable := getVariable(str[anotherPlace:newPlace])
			arg = variableDict[variable]
			// fmt.Println(variable, arg)
			newExp = strings.ReplaceAll(newExp, variable, arg)
			anotherPlace = newPlace

		} else {
			continue
		}

	}
	// fmt.Println(arg, "--------->>>>>>>>>>>", newExp)

	arg = eval(newExp)
	newString += parseString(arg)
	return newString

}

// parse and evaluate variable functions
func evalVarExpression(str string) string {
	inString := 0
	newShow := str
	newString := ""
	// check if the string has any commas outside string or if it's a concat
	var isConcat bool
	var oneStatement bool
	// fmt.Println(str, " STR : ")

	oneStatement = isOneVariable(str)
	isConcat = isConcatExp(str)
	// fmt.Println(str, oneStatement, isConcat, "----eval something")
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
		return parseString(strings.ReplaceAll(eval(parseToken), "\\n", "\n"))
	} else if oneStatement == true {
		// fmt.Println("here -------->")
		variable := getVariable(str[0:strings.LastIndex(str, ".")])
		if !strings.Contains(str, ",") {
			return variableDict[variable]
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
					arg = ""
					if evalType(newShow[place:count]) == "Var" {
						oneVar := isOneVariable(newShow[place:count])
						if oneVar == true {
							variable := getVariable(newShow[place:count])
							arg = variableDict[variable]
							newString += parseString(arg)
						} else {
							// fmt.Println(newShow[place:count], "((((((((((((((((((((((((((")
							newString += getevalVar(newShow[place:count])
							// fmt.Println(newString, "))))))))))))))))))))))))))))))))))))")
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
					place = count + 1
				} else if count == strings.LastIndex(str, ".") {
					arg = ""
					if evalType(newShow[place:count]) == "Var" {
						// fmt.Println(place, "<------", count, "------>")
						oneVar := isOneVariable(newShow[place:count])
						// fmt.Println(isOneVariable(newShow[place:count]), newShow[place:count], "-----------", isOneVariable(newShow), "here----->")
						if oneVar == true {
							variable := getVariable(newShow[place:count])
							arg = variableDict[variable]
							newString += parseString(arg)
						} else {
							newString += getevalVarPeriod(newShow[place:count])
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
			return strings.ReplaceAll(newString, "\\n", "\n")
		}

	} else {
		// fmt.Println(str, "-------------------->>>>>>>>>>>>>>>>>>>>>>>>>>")
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
				arg = ""
				if evalType(newShow[place:count]) == "Var" {
					oneVar := isOneVariable(newShow[place:count])
					if oneVar == true {
						variable := getVariable(newShow[place:count])
						arg = variableDict[variable]
						newString += parseString(arg)
					} else {
						// fmt.Println(newShow[place:count], "((((((((((((((((((((((((((")
						newString += getevalVar(newShow[place:count])
						// fmt.Println(newString, "))))))))))))))))))))))))))))))))))))")
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
				place = count + 1
			} else if count == strings.LastIndex(str, ".") {
				arg = ""
				if evalType(newShow[place:count]) == "Var" {
					// fmt.Println(place, "<------", count, "------>")
					oneVar := isOneVariable(newShow[place:count])
					// fmt.Println(isOneVariable(newShow[place:count]), newShow[place:count], "-----------", isOneVariable(newShow), "here----->")
					if oneVar == true {
						variable := getVariable(newShow[place:count])
						arg = variableDict[variable]
						newString += parseString(arg)
					} else {
						newString += getevalVarPeriod(newShow[place:count])
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

// Parse string and put it in the proper format
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

// show function
// use evalType to show eval expressions
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
		// fmt.Println(showTok[1])
		fmt.Println(evalVarExpression(showTok[1]))
	} else if evalType(parseToken) == "Exp" {
		fmt.Println(evalExpression(showTok[1]))
	} else if evalType(parseToken) == "Bool" {
		fmt.Println(evalExpression(showTok[1]))
	}
}

func check(err error) {
	if err != nil {
		fmt.Println(err)
		return
	}
}

// ------------------------------------------------------------
// Function section
// This area for the function ability of the interpreter
// ------------------------------------------------------------
// function dictionary

var functionDict = make(map[string]function)

// function Struct
type function struct {
	funcVariableDict map[string]string
	argumentState    bool
	argumentDict     []string
	argumentCount    int
	content          []string
	name             string
	contentLen       int
}

// Evaluate variable expressions by getting variables and changing
// variable expressions into non variable expressions
func getevalVarFunc(str string, name string) string {
	arg := ""
	newExp := ""
	newString := ""
	place := 0

	newPlace := 0
	InnerinString := 0
	anotherPlace := place
	newExp = str
	// fmt.Println(str, "----eval comma----")
	for newPlace = place; newPlace < len(str)-1; newPlace++ {
		if string(str[newPlace]) == "\"" {
			InnerinString += 1
			if InnerinString > 1 {
				InnerinString = 0
			}
			continue
		} else if InnerinString == 0 && string(str[newPlace]) == plus || string(str[newPlace]) == minus || string(str[newPlace]) == divide || string(str[newPlace]) == multiply {
			variable := getVariable(str[anotherPlace:newPlace])
			arg = functionDict[name].funcVariableDict[variable]
			newExp = strings.ReplaceAll(newExp, variable, arg)
			anotherPlace = newPlace
		} else {
			continue
		}
	}
	// fmt.Println("---", newExp, "---", "NEW EXP")
	variable := getVariable(newExp)
	arg = functionDict[name].funcVariableDict[variable]
	newExp = strings.ReplaceAll(newExp, variable, arg)
	anotherPlace = newPlace
	// fmt.Println("---", arg, "---", "NEW EVAL")
	arg = eval(newExp)
	// fmt.Println("---", arg, "---", "NEW EVAL")
	newString += parseString(arg)
	return newString

}

// Evaluate variable expressions by getting variables and changing
// variable expressions into non variable expressions
// Evaluate variable expression at end of a statement before period
func getevalVarPeriodFunc(str string, name string) string {
	arg := ""
	newExp := ""
	newString := ""
	place := 0

	newPlace := 0
	InnerinString := 0
	anotherPlace := place
	newExp = str
	for newPlace = place; newPlace <= len(str)-1; newPlace++ {
		arg = ""
		if string(str[newPlace]) == "\"" {
			InnerinString += 1
			if InnerinString > 1 {
				InnerinString = 0
			}
			continue
		} else if InnerinString == 0 && string(str[newPlace]) == plus || string(str[newPlace]) == minus || string(str[newPlace]) == divide || string(str[newPlace]) == multiply {
			variable := getVariable(str[anotherPlace:newPlace])
			arg = functionDict[name].funcVariableDict[variable]
			// fmt.Println(variable, arg)
			newExp = strings.ReplaceAll(newExp, variable, arg)
			anotherPlace = newPlace
		} else if InnerinString == 0 && newPlace == len(str)-1 {
			variable := getVariable(str[anotherPlace:newPlace])
			arg = functionDict[name].funcVariableDict[variable]
			// fmt.Println(variable, arg)
			newExp = strings.ReplaceAll(newExp, variable, arg)
			anotherPlace = newPlace

		} else {
			continue
		}

	}
	// fmt.Println(arg, "--------->>>>>>>>>>>", newExp)

	arg = eval(newExp)
	newString += parseString(arg)
	return newString

}

// Insert variable into the variable dictionary
// may change this functionality but so far it works well
func insertVariableFunc(variableToken string, name string) {
	// fmt.Println(variableDict, variableToken, "<---->")
	newtoken := variableToken
	varToken := strings.Split(newtoken, "=")
	//fmt.Println("Variable type : ", evalType(varToken[1]), "  : variable value : ", varToken[1])
	varToken[0] = strings.ReplaceAll(varToken[0], " ", "")
	if evalType(varToken[1]) == "String" {
		functionDict[name].funcVariableDict[string(varToken[0])] = eval(varToken[1])
	} else if evalType(varToken[1]) == "Int" {
		functionDict[name].funcVariableDict[string(varToken[0])] = eval(varToken[1])
	} else if evalType(varToken[1]) == "Float" {
		functionDict[name].funcVariableDict[string(varToken[0])] = eval(varToken[1])
	} else if evalType(varToken[1]) == "Bool" {
		functionDict[name].funcVariableDict[string(varToken[0])] = eval(varToken[1])
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
			_, isPresent := functionDict[name].funcVariableDict[newVarTok[v]]
			if isPresent {
				newVarTok[v] = functionDict[name].funcVariableDict[newVarTok[v]]
			}
		}
		VarTok := strings.Join(newVarTok, " ")
		functionDict[name].funcVariableDict[string(varToken[0])] = eval(VarTok)

	}
}

// parse and evaluate variable functions
func evalVarExpressionFunc(str string, name string) string {
	inString := 0
	newShow := str
	newString := ""
	// check if the string has any commas outside string or if it's a concat
	var isConcat bool
	var oneStatement bool
	// fmt.Println(str, " STR : ")

	oneStatement = isOneVariable(str)
	isConcat = isConcatExp(str)
	// fmt.Println(str, oneStatement, isConcat, "----eval something")
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
				parseToken = strings.ReplaceAll(parseToken, variable, functionDict[name].funcVariableDict[variable])
				place = count + 1
			} else if count == strings.LastIndex(str, ".") {
				variable := getVariable(parseToken[place:count])
				parseToken = strings.ReplaceAll(parseToken, variable, functionDict[name].funcVariableDict[variable])
			}
		}
		// fmt.Println(parseToken)
		variable := getVariable(parseToken)
		parseToken = strings.ReplaceAll(parseToken, variable, functionDict[name].funcVariableDict[variable])
		// fmt.Println(parseToken)
		parseToken = parseToken[0:strings.LastIndex(parseToken, ".")]
		return parseString(strings.ReplaceAll(eval(parseToken), "\\n", "\n"))
	} else if oneStatement == true {
		variable := getVariable(str[0:strings.LastIndex(str, ".")])
		if !strings.Contains(str, ",") {
			return functionDict[name].funcVariableDict[variable]
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
					arg = ""
					if evalType(newShow[place:count]) == "Var" {
						oneVar := isOneVariable(newShow[place:count])
						if oneVar == true {
							variable := getVariable(newShow[place:count])
							arg = functionDict[name].funcVariableDict[variable]
							newString += parseString(arg)
						} else {
							// fmt.Println(newShow[place:count], "((((((((((((((((((((((((((")
							newString += getevalVarFunc(newShow[place:count], name)
							// fmt.Println(newString, "))))))))))))))))))))))))))))))))))))")
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
					place = count + 1
				} else if count == strings.LastIndex(str, ".") {
					arg = ""
					if evalType(newShow[place:count]) == "Var" {
						// fmt.Println(place, "<------", count, "------>")
						oneVar := isOneVariable(newShow[place:count])
						// fmt.Println(isOneVariable(newShow[place:count]), newShow[place:count], "-----------", isOneVariable(newShow), "here----->")
						if oneVar == true {
							variable := getVariable(newShow[place:count])
							arg = functionDict[name].funcVariableDict[variable]
							newString += parseString(arg)
						} else {
							newString += getevalVarPeriodFunc(newShow[place:count], name)
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
			return strings.ReplaceAll(newString, "\\n", "\n")
		}

	} else {
		// fmt.Println(str, "-------------------->>>>>>>>>>>>>>>>>>>>>>>>>>")
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
				arg = ""
				if evalType(newShow[place:count]) == "Var" {
					oneVar := isOneVariable(newShow[place:count])
					if oneVar == true {
						variable := getVariable(newShow[place:count])
						arg = functionDict[name].funcVariableDict[variable]
						newString += parseString(arg)
					} else {
						// fmt.Println(newShow[place:count], "((((((((((((((((((((((((((")
						newString += getevalVarFunc(newShow[place:count], name)
						// fmt.Println(newString, "))))))))))))))))))))))))))))))))))))")
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
				place = count + 1
			} else if count == strings.LastIndex(str, ".") {
				arg = ""
				if evalType(newShow[place:count]) == "Var" {
					// fmt.Println(place, "<------", count, "------>")
					oneVar := isOneVariable(newShow[place:count])
					// fmt.Println(isOneVariable(newShow[place:count]), newShow[place:count], "-----------", isOneVariable(newShow), "here----->")
					if oneVar == true {
						variable := getVariable(newShow[place:count])
						arg = functionDict[name].funcVariableDict[variable]
						newString += parseString(arg)
					} else {
						newString += getevalVarPeriodFunc(newShow[place:count], name)
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

// show function
// use evalType to show eval expressions
func showRealFunc(str string, name string) {

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
		fmt.Println(evalVarExpressionFunc(showTok[1], name))
	} else if evalType(parseToken) == "Exp" {
		fmt.Println(evalExpression(showTok[1]))
	} else if evalType(parseToken) == "Bool" {
		fmt.Println(evalExpression(showTok[1]))
	}
}

// Function protocol for when functions are called
func functionProtocol(str string, state string) {
	// fmt.Println(str, "<------>", state)
	conditionState := false
	conditionName := ""
	loopState := false
	loopName := ""
	name := str
	if strings.Contains(name, "=") && strings.Index(name, "[") > strings.Index(name, "=") {
		variable := getVariable(name[0:strings.Index(name, "=")])
		name = strings.ReplaceAll(name, variable, "")
		name = name[0:strings.Index(name, "[")]
		name = strings.ReplaceAll(name, " ", "")
		name = strings.ReplaceAll(name, "=", "")
	} else {
		name = name[0:strings.Index(name, "[")]
		name = strings.ReplaceAll(name, " ", "")
	}
	Calledfunction := functionDict[name]

	if Calledfunction.name != name {
		fmt.Println("function Error")
	}
	// change the variables in scope
	variables := str[strings.Index(str, "[")+1 : strings.Index(str, "]")]
	// fmt.Println(variables)
	variablesSet := strings.Split(variables, ",")
	// fmt.Println(variablesSet)
	// fmt.Println(Calledfunction, "<--- Called Function --->")
	count := 0
	for _, vars := range Calledfunction.argumentDict {

		// Calledfunction.funcVariableDict[vars] = variablesSet[count]
		if isOneVariable(variablesSet[count]) {
			// fmt.Println(getVariable(variablesSet[count]), "----", variableDict[getVariable(variablesSet[count])])
			if state == "isMain" {
				_, isPresent := variableDict[getVariable(variablesSet[count])]

				if isPresent {
					// add getVariable to clean up spaces on vars
					Calledfunction.funcVariableDict[vars] = variableDict[getVariable(variablesSet[count])]

				} else {
					Calledfunction.funcVariableDict[vars] = variablesSet[count]
				}
			} else {
				_, isPresent := functionDict[state].funcVariableDict[getVariable(variablesSet[count])]

				if isPresent {
					// add getVariable to clean up spaces on vars
					Calledfunction.funcVariableDict[vars] = functionDict[state].funcVariableDict[getVariable(variablesSet[count])]
				} else {
					Calledfunction.funcVariableDict[vars] = variablesSet[count]
				}

			}

		}

		count += 1
	}
	// fmt.Println(Calledfunction, "<--- Called Function --->")

	for _, tok := range Calledfunction.content {
		if len(tok) == 0 {
			continue
		} else if strings.Contains(tok, "//") && strings.Contains(tok, "\"") && strings.Index(tok, "//") < strings.Index(tok, "\"") {
			continue
		} else if strings.Contains(tok, "//") {
			comments := strings.SplitAfter(tok, "//")
			if strings.Contains(comments[0], "//") {
				continue
			}
		} else if strings.Contains(tok, "return") {
			returnCode := strings.Split(tok, "return ")
			varState := false
			// fmt.Println("RETURN : ", returnCode[1])
			// fmt.Println(getVariable(returnCode[1]))
			// fmt.Println(str, state)
			// fmt.Println(name, state)
			// fmt.Println(variableDict)
			// Calledfunction.funcVariableDict[getVariable(returnCode[1])]
			if state == "isMain" {
				// variableDict[]
				for key, element := range variableDict {
					// fmt.Println("Key:", key, "=>", "Element:", element)
					if strings.Contains(element, name) {
						variableDict[key] = Calledfunction.funcVariableDict[getVariable(returnCode[1])]
						varState = true
						break
					}
				}
				if varState == false {
					fmt.Println("variable error :")
				}

			} else {
				// fmt.Println(state, Calledfunction.name)
				// fmt.Println(functionDict, "Function Dictionary", "\n", "\n")
				// fmt.Println(Calledfunction, "Called Function ", "\n", "\n")
				for key, element := range functionDict[state].funcVariableDict {
					// fmt.Println("Key:", key, "=>", "Element:", element)

					if strings.Contains(element, Calledfunction.name) {
						// fmt.Println("found element")
						functionDict[state].funcVariableDict[key] = Calledfunction.funcVariableDict[getVariable(returnCode[1])]
						varState = true
						// fmt.Println(functionDict, "Function Dictionary", "\n", "\n")
						break

					}
				}
				if varState == false {
					fmt.Println("Called function variable error :")
				}

			}

			continue

		} else if strings.Contains(tok, "def") && strings.Contains(tok, "[end]") {
			continue

		} else if strings.Contains(tok, "if") && strings.Contains(tok, "]") && strings.Contains(tok, "[") && conditionState == false {
			conditionState = true
			var ifElse ifelseCondition
			ifElse.head = tok
			conditionName = tok
			ifElse.content = append(ifElse.content, tok)
			ifElseDict[ifElse.head] = ifElse
			continue
		} else if conditionState {
			if strings.Contains(tok, "[end]") && strings.Contains(tok, "if") {
				ifelseCopy := ifElseDict[conditionName]
				ifelseCopy.content = append(ifElseDict[conditionName].content, tok)
				ifElseDict[conditionName] = ifelseCopy
				ifelse(ifElseDict[conditionName].content, Calledfunction.name)
				conditionState = false
				// ifElseDict[conditionName].content = append(ifElseDict[conditionName].content, tok)
			} else {
				ifelseCopy := ifElseDict[conditionName]
				ifelseCopy.content = append(ifElseDict[conditionName].content, tok)
				ifElseDict[conditionName] = ifelseCopy

			}
			continue
		} else if strings.Contains(tok, "]") && strings.Contains(tok, "loop") && strings.Contains(tok, "[") && !strings.Contains(tok, "[end]") && loopState == false {
			// fmt.Println("here")
			loopState = true
			loopName = tok
			var Newloop loop
			Newloop.name = loopName
			Newloop.content = append(Newloop.content, tok)
			loopDict[loopName] = Newloop
		} else if loopState == true {
			Newloop := loopDict[loopName]
			Newloop.content = append(Newloop.content, tok)
			loopDict[loopName] = Newloop
			if strings.Contains(tok, "[loop]") && strings.Contains(tok, "[end]") {
				loopState = false
				loopStructure(loopDict[loopName].content, Calledfunction.name)
			}

		} else if strings.Contains(tok, "[") && strings.Contains(tok, "]") && strings.Contains(tok, "=") && strings.Index(tok, "=") < strings.Index(tok, "[") {
			// fmt.Println("here --->", name)
			insertFunction(tok, Calledfunction.name)

		} else if strings.Contains(tok, "[") && strings.Contains(tok, "]") {
			functionProtocol(tok, name)
		} else if strings.Contains(tok, "show") {
			showTok := strings.SplitAfter(tok, "show")
			if strings.Contains(showTok[0], "show") {
				showRealFunc(tok, name)
			}
		} else if strings.Contains(tok, "?") && strings.Contains(tok, "=") {
			if strings.Index(tok, "?") < strings.Index(tok, "\"") {
				input := strings.Split(tok, "=")
				// fmt.Println(input)
				var variable string = ""
				fmt.Print(getPrompt(tok))
				scanIn := bufio.NewScanner(os.Stdin)
				scanIn.Scan()
				variable = scanIn.Text()
				vars := strings.ReplaceAll(input[0], " ", "")
				// fmt.Println(variable)
				Calledfunction.funcVariableDict[vars] = variable
				//variableDict[vars] = variable
				// fmt.Println(variableDict)
			}
		} else if strings.Contains(tok, "=") {
			varTok := strings.SplitAfter(tok, "=")
			if strings.Contains(varTok[0], "=") {
				insertVariableFunc(tok, name)
			}
		}
	}

}

// ------------------------------------------------------------
// IF Else logic
// ------------------------------------------------------------
// take a struct that has a slice to hold contents.
// ifelse function used to loop through structure to execute proper actions.
type ifelseCondition struct {
	head    string
	content []string
}

var ifElseDict = make(map[string]ifelseCondition)

func ifelse(token []string, state string) {
	conditionState := false
	conditionMet := false
	currentCondition := false
	nested := false
	nestedState := false
	outerCondition := false
	//outerState := ""
	// nestedSet := make([]string, 0)
	for _, tok := range token {
		// fmt.Println(currentCondition, tok, "------------------------------------->", nested)
		if strings.Contains(tok, "if") && strings.Contains(tok, "]") && !strings.Contains(tok, "else") && !strings.Contains(tok, "[end]") {
			// if only check to see if nested
			if strings.Index(tok, "[") > strings.Index(tok, "if") {
				// if not nested check expression
				expression := ifelseParser(tok, state)
				if "true" == eval(expression) {
					// if expression is true set true
					conditionMet = true
					conditionState = true
					currentCondition = true
					// set outer condition true if nested
					//outerState = "IF"
					outerCondition = true
				} else {
					// set all false
					conditionMet = false
					conditionState = false
					currentCondition = false
					//outerState = "IF"
					outerCondition = false
				}
			} else {
				// first if of a nested statement
				if nested == false && currentCondition == true && outerCondition == true {
					tok = strings.Replace(tok, "[", "", 1)

					expression := ifelseParser(tok, state)
					if "true" == eval(expression) {
						conditionMet = true
						conditionState = true
						currentCondition = true
						nestedState = true
						nested = true
						//outerState = "IF"
						outerCondition = true
					} else {
						//outerState = "IF"
						outerCondition = true
						conditionMet = false
						conditionState = false
						currentCondition = false
						nested = true
						nestedState = false
					}
				} else {
					//
					nested = true
					nestedState = false
					conditionMet = false
					conditionState = false
					currentCondition = false
					outerCondition = false
				}
			}
		} else if strings.Contains(tok, "if") && strings.Contains(tok, "else") && strings.Contains(tok, "]") {
			// else if check
			if strings.Index(tok, "[") > strings.Index(tok, "else if") && currentCondition == false {
				// ELSE IF THAT is not nested
				//outerState = "ELSE IF"
				nested = false
				nestedState = false
				expression := ifelseParser(tok, state)
				// check ELSE IF
				if "true" == eval(expression) {
					conditionMet = true
					conditionState = true
					outerCondition = true
					// nested = false
					currentCondition = true
				} else {
					conditionMet = false
					conditionState = false
					outerCondition = false
					currentCondition = false
				}

			} else if strings.Index(tok, "[") > strings.Index(tok, "else if") && currentCondition == true {
				break
			} else if strings.Index(tok, "[") < strings.Index(tok, "else if") && currentCondition == true {
				break
			} else if strings.Index(tok, "[") < strings.Index(tok, "else if") && currentCondition == false && nestedState == false && nested == true {
				if outerCondition == false {
					continue
				} else {
					tok = strings.Replace(tok, "[", "", 1)
					expression := ifelseParser(tok, state)
					if "true" == eval(expression) {
						conditionMet = true
						conditionState = true
						currentCondition = true
						nestedState = true
					} else {
						conditionMet = false
						conditionState = false
						currentCondition = false
						nestedState = false
					}

				}
			}

		} else if strings.Contains(tok, "]") && strings.Contains(tok, "else") && !strings.Contains(tok, "if") {
			if strings.Index(tok, "[") > strings.Index(tok, "else") && currentCondition == false {
				conditionMet = true
				conditionState = true
				nested = false
				nestedState = false
			} else if strings.Index(tok, "[") > strings.Index(tok, "else") && currentCondition == true {
				break
			} else if strings.Index(tok, "[") < strings.Index(tok, "else") && currentCondition == true && outerCondition == true {
				break
			} else if strings.Index(tok, "[") < strings.Index(tok, "else") && currentCondition == false && nested == true && nestedState == false {
				if outerCondition == false {
					nested = false
					continue
				} else {
					conditionMet = true
					conditionState = true
					currentCondition = true
					nestedState = false
				}

			} else {
				conditionMet = true
				conditionState = true
			}

		} else if strings.Contains(tok, "if [end]") {
			conditionState = true
			conditionMet = true
		} else if conditionState == true && conditionMet == true {
			callCode(tok, state)
		}
	}
}

func ifelseParser(tok string, state string) string {
	// Add state for functions
	newExpression := ""
	expression := tok[strings.Index(tok, "]")+1 : strings.LastIndex(tok, "[")]
	for _, word := range expression {
		// fmt.Println(string(word))
		if string(word) == ">" || string(word) == "<" || string(word) == "=" || string(word) == "&" || string(word) == "|" || string(word) == "!" {
			newExpression += string(word)
			variable := getVariable(newExpression)
			if state == "isMain" && variable != "" {
				newExpression = strings.ReplaceAll(newExpression, variable, variableDict[variable])
			} else if state != "isMain" && variable != "" {
				newExpression = strings.ReplaceAll(newExpression, variable, functionDict[state].funcVariableDict[variable])
			}
		} else {
			newExpression += string(word)
		}
	}
	variable := getVariable(newExpression)
	if state == "isMain" && variable != "" {
		newExpression = strings.ReplaceAll(newExpression, variable, variableDict[variable])
	} else if state != "isMain" && variable != "" {
		newExpression = strings.ReplaceAll(newExpression, variable, functionDict[state].funcVariableDict[variable])
	}
	// def definition [a]

	return newExpression
}

// independent code execution without use of main or function dependency
func callCode(tok string, state string) {
	definitionState := false
	definitionName := ""
	//conditionState := false
	conditionState := false
	conditionName := ""
	loopState := false
	loopName := ""
	if len(tok) == 0 {

	} else if strings.Contains(tok, "//") && strings.Contains(tok, "\"") && strings.Index(tok, "//") < strings.Index(tok, "\"") {

	} else if strings.Contains(tok, "//") {
		comments := strings.SplitAfter(tok, "//")
		if strings.Contains(comments[0], "//") {

		}
	} else if strings.Contains(tok, "def ") && strings.Contains(tok, "[") && strings.Contains(tok, "[") && definitionState == false {
		definitionState = true
		var Newfunction function
		nameSet := strings.SplitAfter(tok, "def ")
		name := nameSet[1]
		name = name[0:strings.Index(name, "[")]
		name = strings.ReplaceAll(name, " ", "")
		Newfunction.name = name
		// fmt.Println(name, "--name of function")
		variables := nameSet[1][strings.Index(nameSet[1], "[")+1 : strings.Index(nameSet[1], "]")]
		variablesSet := strings.Split(variables, ",")
		// fmt.Println(variables, "-- variables of function", variablesSet)
		Newfunction.argumentCount = len(variablesSet)
		Newfunction.argumentDict = variablesSet
		Newfunction.funcVariableDict = make(map[string]string)
		if Newfunction.argumentCount > 0 {
			Newfunction.argumentState = true
		}
		for v := range variablesSet {
			// fmt.Println(variablesSet[v], "v-set----")
			// Newfunction.argumentDict[variablesSet[v]] = variablesSet[v]
			Newfunction.funcVariableDict[variablesSet[v]] = variablesSet[v]
		}
		// fmt.Println(Newfunction, "-- data code")
		functionDict[Newfunction.name] = Newfunction
		definitionName = Newfunction.name
		Newfunction.content = make([]string, 0)

	} else if definitionState == true {
		// fmt.Println(tok, "--def state---")
		if strings.Contains(tok, "def ") && strings.Contains(tok, "[end]") {
			definitionState = false
			contentDef := functionDict[definitionName]
			contentDef.content = append(functionDict[definitionName].content, tok)
			contentDef.contentLen = len(contentDef.content)
			functionDict[definitionName] = contentDef

		} else {
			contentDef := functionDict[definitionName]
			contentDef.content = append(functionDict[definitionName].content, tok)
			functionDict[definitionName] = contentDef
		}
		// fmt.Println(functionDict[definitionName], "--------- FUNC CODE")

	} else if strings.Contains(tok, "if") && strings.Contains(tok, "]") && strings.Contains(tok, "[") && conditionState == false {
		conditionState = true
		var ifElse ifelseCondition
		ifElse.head = tok
		conditionName = tok
		ifElse.content = append(ifElse.content, tok)
		ifElseDict[ifElse.head] = ifElse
		// continue
	} else if conditionState {
		if strings.Contains(tok, "[end]") && strings.Contains(tok, "if") {
			ifelseCopy := ifElseDict[conditionName]
			ifelseCopy.content = append(ifElseDict[conditionName].content, tok)
			ifElseDict[conditionName] = ifelseCopy
			ifelse(ifElseDict[conditionName].content, state)
			conditionState = false
			// ifElseDict[conditionName].content = append(ifElseDict[conditionName].content, tok)
		} else {
			ifelseCopy := ifElseDict[conditionName]
			ifelseCopy.content = append(ifElseDict[conditionName].content, tok)
			ifElseDict[conditionName] = ifelseCopy

		}
		// continue
	} else if strings.Contains(tok, "]") && strings.Contains(tok, "loop") && strings.Contains(tok, "[") && !strings.Contains(tok, "[end]") && loopState == false {
		// fmt.Println("here")
		loopState = true
		loopName = tok
		var Newloop loop
		Newloop.name = loopName
		Newloop.content = append(Newloop.content, tok)
		loopDict[loopName] = Newloop
	} else if loopState == true {
		Newloop := loopDict[loopName]
		Newloop.content = append(Newloop.content, tok)
		loopDict[loopName] = Newloop
		if strings.Contains(tok, "[loop]") && strings.Contains(tok, "[end]") {
			loopState = false
			loopStructure(loopDict[loopName].content, state)
		}

	} else if strings.Contains(tok, "[") && strings.Contains(tok, "]") && strings.Contains(tok, "=") && strings.Index(tok, "=") < strings.Index(tok, "[") {
		// fmt.Println("here --->")
		insertFunction(tok, state)
	} else if strings.Contains(tok, "[") && strings.Contains(tok, "]") {
		functionProtocol(tok, state)

	} else if strings.Contains(tok, "show") {
		showTok := strings.SplitAfter(tok, "show")
		if strings.Contains(showTok[0], "show") && state == "isMain" {
			showReal(tok)
		} else {
			showRealFunc(tok, state)
		}
	} else if strings.Contains(tok, "?") && strings.Contains(tok, "=") {
		if strings.Index(tok, "?") < strings.Index(tok, "\"") {
			input := strings.Split(tok, "=")
			// fmt.Println(input)
			var variable string = ""
			fmt.Print(getPrompt(tok))
			scanIn := bufio.NewScanner(os.Stdin)
			scanIn.Scan()
			variable = scanIn.Text()
			vars := strings.ReplaceAll(input[0], " ", "")
			// fmt.Println(variable)
			variableDict[vars] = variable
			// fmt.Println(variableDict)
		}
	} else if strings.Contains(tok, "=") {
		varTok := strings.SplitAfter(tok, "=")
		if strings.Contains(varTok[0], "=") && state == "isMain" {
			insertVariable(tok)
		} else {
			insertVariableFunc(tok, state)
		}
	}

}

// Get phrase from prompt for input for variables
func getPrompt(prompt string) string {
	start := strings.Index(prompt, "\"")
	end := strings.LastIndex(prompt, "\"")
	phrase := ""
	if start < 0 {

	} else if start > 0 && end > 0 {
		start += 1
		for ; start < end; start++ {
			phrase += string(prompt[start])
		}
	}

	return phrase
}

// ------------------------------------------------------------
// loop logic
// ------------------------------------------------------------
// [loop][7]
// [loop][counter = 0; counter < a; counter++]
// [loop][counter < 5]
type loop struct {
	counter     int
	content     []string
	nestedState bool
	nested      map[string][]string
	name        string
}

var loopDict = make(map[string]loop)

func loopStructure(loop []string, state string) {
	// add logic to parse out the conditions of the loop
	// fix the outer loop logic
	// if state == "isMain" {
	// 	state = loop[0]
	// }
	// fmt.Println(loop)
	loopConstruct := loop[0]
	loopConstruct = strings.ReplaceAll(loopConstruct, "[loop]", "")
	loopConstruct = strings.ReplaceAll(loopConstruct, "[", "")
	loopConstruct = strings.ReplaceAll(loopConstruct, "]", "")
	count := 0
	expressionV := 0
	incrementer := ""
	operator := ""

	if strings.Contains(loopConstruct, ";") {
		loopParsed := strings.Split(loopConstruct, ";")
		for _, looptoken := range loopParsed {
			if strings.Contains(looptoken, "=") && !strings.Contains(looptoken, "<") && !strings.Contains(looptoken, ">") && !strings.Contains(looptoken, "!") {
				value := strings.Split(looptoken, "=")
				if getVariable(value[1]) != "" {
					if state == "isMain" {
						counter, _ := strconv.Atoi(variableDict[getVariable(value[1])])
						count = counter
					} else {
						counter, _ := strconv.Atoi(functionDict[state].funcVariableDict[getVariable(value[1])])
						count = counter
					}

				} else {
					counter, _ := strconv.Atoi(value[1])
					count = counter
				}

			} else if strings.Contains(looptoken, "<") && !strings.Contains(looptoken, "=") {
				operator = "<"
				value := strings.Split(looptoken, operator)
				if getVariable(value[1]) != "" {
					if state == "isMain" {
						expressionValue, _ := strconv.Atoi(variableDict[getVariable(value[1])])
						expressionV = expressionValue
					} else {
						expressionValue, _ := strconv.Atoi(functionDict[state].funcVariableDict[getVariable(value[1])])
						expressionV = expressionValue
					}

				} else {
					expressionValue, _ := strconv.Atoi(strings.ReplaceAll(value[1], " ", ""))
					expressionV = expressionValue
				}
			} else if strings.Contains(looptoken, "<") && strings.Contains(looptoken, "=") {
				operator = "<="
				value := strings.Split(looptoken, operator)
				if getVariable(value[1]) != "" {
					if state == "isMain" {
						expressionValue, _ := strconv.Atoi(variableDict[getVariable(value[1])])
						expressionV = expressionValue
					} else {
						expressionValue, _ := strconv.Atoi(functionDict[state].funcVariableDict[getVariable(value[1])])
						expressionV = expressionValue
					}

				} else {
					expressionValue, _ := strconv.Atoi(strings.ReplaceAll(value[1], " ", ""))
					expressionV = expressionValue
				}
			} else if strings.Contains(looptoken, ">") && !strings.Contains(looptoken, "=") {
				operator = ">"
				value := strings.Split(looptoken, operator)
				if getVariable(value[1]) != "" {
					if state == "isMain" {
						expressionValue, _ := strconv.Atoi(variableDict[getVariable(value[1])])
						expressionV = expressionValue
					} else {
						expressionValue, _ := strconv.Atoi(functionDict[state].funcVariableDict[getVariable(value[1])])
						expressionV = expressionValue
					}

				} else {
					expressionValue, _ := strconv.Atoi(strings.ReplaceAll(value[1], " ", ""))
					expressionV = expressionValue
				}
			} else if strings.Contains(looptoken, ">") && strings.Contains(looptoken, "=") {
				operator = ">="
				value := strings.Split(looptoken, operator)
				if getVariable(value[1]) != "" {
					if state == "isMain" {
						expressionValue, _ := strconv.Atoi(variableDict[getVariable(value[1])])
						expressionV = expressionValue
					} else {
						expressionValue, _ := strconv.Atoi(functionDict[state].funcVariableDict[getVariable(value[1])])
						expressionV = expressionValue
					}

				} else {
					expressionValue, _ := strconv.Atoi(strings.ReplaceAll(value[1], " ", ""))
					expressionV = expressionValue
				}
			} else if strings.Contains(looptoken, "!") && strings.Contains(looptoken, "=") {
				operator = "!="
				value := strings.Split(looptoken, operator)
				if getVariable(value[1]) != "" {
					if state == "isMain" {
						expressionValue, _ := strconv.Atoi(variableDict[getVariable(value[1])])
						expressionV = expressionValue
					} else {
						expressionValue, _ := strconv.Atoi(functionDict[state].funcVariableDict[getVariable(value[1])])
						expressionV = expressionValue
					}

				} else {
					expressionValue, _ := strconv.Atoi(strings.ReplaceAll(value[1], " ", ""))
					expressionV = expressionValue
				}
			} else if strings.Contains(looptoken, "++") {
				incrementer = "++"
			} else if strings.Contains(looptoken, "--") {
				incrementer = "--"
			}

		}

	} else {

	}
	// fmt.Println("i :=", count, "i", operator, expressionV, "; i", incrementer)
	if operator == "<" && incrementer == "++" {
		for i := count; i < expressionV; i++ {
			// perfect logic below
			for _, tok := range loop {
				if strings.Contains(tok, "]") && strings.Contains(tok, "loop") && strings.Contains(tok, "[") && !strings.Contains(tok, "[end]") {
					continue
				} else if strings.Contains(tok, "]") && strings.Contains(tok, "loop") && strings.Contains(tok, "[") && strings.Contains(tok, "[end]") {
					continue
				} else if strings.Contains(tok, "break") {
					// add further logic for making sure break is not in a string
					break

				} else if strings.Contains(tok, "continue") {
					// add further logic for making sure continue is not in a string
					continue

				} else {
					// fmt.Println(tok)
					callCode(tok, state)
				}
			}
		}
	} else if operator == "<=" && incrementer == "++" {
		for i := count; i <= expressionV; i++ {
			// perfect logic below
			nested := false
			loopheaderCount := 0
			nestedLoop := make([]string, 0)
			for _, tok := range loop {
				if strings.Contains(tok, "]") && strings.Contains(tok, "loop") && strings.Contains(tok, "[") && !strings.Contains(tok, "[end]") {
					loopheaderCount += 1
					if loopheaderCount > 1 {
						nested = true
						nestedLoop = append(nestedLoop, tok)
					}
					continue
				} else if strings.Contains(tok, "]") && strings.Contains(tok, "loop") && strings.Contains(tok, "[") && strings.Contains(tok, "[end]") {
					if nested {
						nestedLoop = append(nestedLoop, tok)
						loopStructure(nestedLoop, state)

					} else {
						continue
					}

				} else if strings.Contains(tok, "break") {
					// add further logic for making sure break is not in a string
					break

				} else if strings.Contains(tok, "continue") {
					// add further logic for making sure continue is not in a string
					continue

				} else {
					// fmt.Println(tok)
					if nested {
						nestedLoop = append(nestedLoop, tok)
					} else {
						callCode(tok, state)
					}
				}
			}
		}

	} else if operator == "<=" && incrementer == "--" {
		for i := count; i <= expressionV; i-- {
			// perfect logic below
			nested := false
			loopheaderCount := 0
			nestedLoop := make([]string, 0)
			for _, tok := range loop {
				if strings.Contains(tok, "]") && strings.Contains(tok, "loop") && strings.Contains(tok, "[") && !strings.Contains(tok, "[end]") {
					loopheaderCount += 1
					if loopheaderCount > 1 {
						nested = true
						nestedLoop = append(nestedLoop, tok)
					}
					continue
				} else if strings.Contains(tok, "]") && strings.Contains(tok, "loop") && strings.Contains(tok, "[") && strings.Contains(tok, "[end]") {
					if nested {
						nestedLoop = append(nestedLoop, tok)
						loopStructure(nestedLoop, state)

					} else {
						continue
					}

				} else if strings.Contains(tok, "break") {
					// add further logic for making sure break is not in a string
					break

				} else if strings.Contains(tok, "continue") {
					// add further logic for making sure continue is not in a string
					continue

				} else {
					// fmt.Println(tok)
					if nested {
						nestedLoop = append(nestedLoop, tok)
					} else {
						callCode(tok, state)
					}
				}
			}
		}

	} else if operator == "<" && incrementer == "--" {
		for i := count; i < expressionV; i-- {
			// perfect logic below
			nested := false
			loopheaderCount := 0
			nestedLoop := make([]string, 0)
			for _, tok := range loop {
				if strings.Contains(tok, "]") && strings.Contains(tok, "loop") && strings.Contains(tok, "[") && !strings.Contains(tok, "[end]") {
					loopheaderCount += 1
					if loopheaderCount > 1 {
						nested = true
						nestedLoop = append(nestedLoop, tok)
					}
					continue
				} else if strings.Contains(tok, "]") && strings.Contains(tok, "loop") && strings.Contains(tok, "[") && strings.Contains(tok, "[end]") {
					if nested {
						nestedLoop = append(nestedLoop, tok)
						loopStructure(nestedLoop, state)

					} else {
						continue
					}

				} else if strings.Contains(tok, "break") {
					// add further logic for making sure break is not in a string
					break

				} else if strings.Contains(tok, "continue") {
					// add further logic for making sure continue is not in a string
					continue

				} else {
					// fmt.Println(tok)
					if nested {
						nestedLoop = append(nestedLoop, tok)
					} else {
						callCode(tok, state)
					}
				}
			}
		}

	} else if operator == ">" && incrementer == "++" {
		for i := count; i < expressionV; i++ {
			// perfect logic below
			nested := false
			loopheaderCount := 0
			nestedLoop := make([]string, 0)
			for _, tok := range loop {
				if strings.Contains(tok, "]") && strings.Contains(tok, "loop") && strings.Contains(tok, "[") && !strings.Contains(tok, "[end]") {
					loopheaderCount += 1
					if loopheaderCount > 1 {
						nested = true
						nestedLoop = append(nestedLoop, tok)
					}
					continue
				} else if strings.Contains(tok, "]") && strings.Contains(tok, "loop") && strings.Contains(tok, "[") && strings.Contains(tok, "[end]") {
					if nested {
						nestedLoop = append(nestedLoop, tok)
						loopStructure(nestedLoop, state)

					} else {
						continue
					}

				} else if strings.Contains(tok, "break") {
					// add further logic for making sure break is not in a string
					break

				} else if strings.Contains(tok, "continue") {
					// add further logic for making sure continue is not in a string
					continue

				} else {
					// fmt.Println(tok)
					if nested {
						nestedLoop = append(nestedLoop, tok)
					} else {
						callCode(tok, state)
					}
				}
			}
		}
	} else if operator == ">=" && incrementer == "++" {
		for i := count; i >= expressionV; i++ {
			// perfect logic below
			nested := false
			loopheaderCount := 0
			nestedLoop := make([]string, 0)
			for _, tok := range loop {
				if strings.Contains(tok, "]") && strings.Contains(tok, "loop") && strings.Contains(tok, "[") && !strings.Contains(tok, "[end]") {
					loopheaderCount += 1
					if loopheaderCount > 1 {
						nested = true
						nestedLoop = append(nestedLoop, tok)
					}
					continue
				} else if strings.Contains(tok, "]") && strings.Contains(tok, "loop") && strings.Contains(tok, "[") && strings.Contains(tok, "[end]") {
					if nested {
						nestedLoop = append(nestedLoop, tok)
						loopStructure(nestedLoop, state)

					} else {
						continue
					}

				} else if strings.Contains(tok, "break") {
					// add further logic for making sure break is not in a string
					break

				} else if strings.Contains(tok, "continue") {
					// add further logic for making sure continue is not in a string
					continue

				} else {
					// fmt.Println(tok)
					if nested {
						nestedLoop = append(nestedLoop, tok)
					} else {
						callCode(tok, state)
					}
				}
			}
		}

	} else if operator == ">=" && incrementer == "--" {
		for i := count; i >= expressionV; i-- {
			// perfect logic below
			nested := false
			loopheaderCount := 0
			nestedLoop := make([]string, 0)
			for _, tok := range loop {
				if strings.Contains(tok, "]") && strings.Contains(tok, "loop") && strings.Contains(tok, "[") && !strings.Contains(tok, "[end]") {
					loopheaderCount += 1
					if loopheaderCount > 1 {
						nested = true
						nestedLoop = append(nestedLoop, tok)
					}
					continue
				} else if strings.Contains(tok, "]") && strings.Contains(tok, "loop") && strings.Contains(tok, "[") && strings.Contains(tok, "[end]") {
					if nested {
						nestedLoop = append(nestedLoop, tok)
						loopStructure(nestedLoop, state)

					} else {
						continue
					}

				} else if strings.Contains(tok, "break") {
					// add further logic for making sure break is not in a string
					break

				} else if strings.Contains(tok, "continue") {
					// add further logic for making sure continue is not in a string
					continue

				} else {
					// fmt.Println(tok)
					if nested {
						nestedLoop = append(nestedLoop, tok)
					} else {
						callCode(tok, state)
					}
				}
			}
		}

	} else if operator == ">" && incrementer == "--" {
		for i := count; i > expressionV; i-- {
			// perfect logic below
			nested := false
			loopheaderCount := 0
			nestedLoop := make([]string, 0)
			for _, tok := range loop {
				if strings.Contains(tok, "]") && strings.Contains(tok, "loop") && strings.Contains(tok, "[") && !strings.Contains(tok, "[end]") {
					loopheaderCount += 1
					if loopheaderCount > 1 {
						nested = true
						nestedLoop = append(nestedLoop, tok)
					}
					continue
				} else if strings.Contains(tok, "]") && strings.Contains(tok, "loop") && strings.Contains(tok, "[") && strings.Contains(tok, "[end]") {
					if nested {
						nestedLoop = append(nestedLoop, tok)
						loopStructure(nestedLoop, state)

					} else {
						continue
					}

				} else if strings.Contains(tok, "break") {
					// add further logic for making sure break is not in a string
					break

				} else if strings.Contains(tok, "continue") {
					// add further logic for making sure continue is not in a string
					continue

				} else {
					// fmt.Println(tok)
					if nested {
						nestedLoop = append(nestedLoop, tok)
					} else {
						callCode(tok, state)
					}
				}
			}
		}

	} else if operator == "!=" && incrementer == "--" {
		for i := count; i != expressionV; i-- {
			// perfect logic below
			nested := false
			loopheaderCount := 0
			nestedLoop := make([]string, 0)
			for _, tok := range loop {
				if strings.Contains(tok, "]") && strings.Contains(tok, "loop") && strings.Contains(tok, "[") && !strings.Contains(tok, "[end]") {
					loopheaderCount += 1
					if loopheaderCount > 1 {
						nested = true
						nestedLoop = append(nestedLoop, tok)
					}
					continue
				} else if strings.Contains(tok, "]") && strings.Contains(tok, "loop") && strings.Contains(tok, "[") && strings.Contains(tok, "[end]") {
					if nested {
						nestedLoop = append(nestedLoop, tok)
						loopStructure(nestedLoop, state)

					} else {
						continue
					}

				} else if strings.Contains(tok, "break") {
					// add further logic for making sure break is not in a string
					break

				} else if strings.Contains(tok, "continue") {
					// add further logic for making sure continue is not in a string
					continue

				} else {
					// fmt.Println(tok)
					if nested {
						nestedLoop = append(nestedLoop, tok)
					} else {
						callCode(tok, state)
					}
				}
			}
		}

	} else if operator == "!=" && incrementer == "++" {
		for i := count; i != expressionV; i-- {
			// perfect logic below
			nested := false
			loopheaderCount := 0
			nestedLoop := make([]string, 0)
			for _, tok := range loop {
				if strings.Contains(tok, "]") && strings.Contains(tok, "loop") && strings.Contains(tok, "[") && !strings.Contains(tok, "[end]") {
					loopheaderCount += 1
					if loopheaderCount > 1 {
						nested = true
						nestedLoop = append(nestedLoop, tok)
					}
					continue
				} else if strings.Contains(tok, "]") && strings.Contains(tok, "loop") && strings.Contains(tok, "[") && strings.Contains(tok, "[end]") {
					if nested {
						nestedLoop = append(nestedLoop, tok)
						loopStructure(nestedLoop, state)

					} else {
						continue
					}

				} else if strings.Contains(tok, "break") {
					// add further logic for making sure break is not in a string
					break

				} else if strings.Contains(tok, "continue") {
					// add further logic for making sure continue is not in a string
					continue

				} else {
					// fmt.Println(tok)
					if nested {
						nestedLoop = append(nestedLoop, tok)
					} else {
						callCode(tok, state)
					}
				}
			}
		}

	}
}

// Main function
func main() {

	file, err := os.Open(os.Args[1])
	check(err)
	definitionState := false
	definitionName := ""
	conditionState := false
	conditionName := ""
	loopState := false
	loopName := ""

	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {

		tok := scanner.Text()
		// fmt.Println(tok)
		if len(tok) == 0 {
			continue
		} else if strings.Contains(tok, "//") && strings.Contains(tok, "\"") && strings.Index(tok, "//") < strings.Index(tok, "\"") {
			continue
		} else if strings.Contains(tok, "//") {
			comments := strings.SplitAfter(tok, "//")
			if strings.Contains(comments[0], "//") {
				continue
			}
		} else if strings.Contains(tok, "def ") && strings.Contains(tok, "[") && strings.Contains(tok, "[") && definitionState == false {
			definitionState = true
			var Newfunction function
			nameSet := strings.SplitAfter(tok, "def ")
			name := nameSet[1]
			name = name[0:strings.Index(name, "[")]
			name = strings.ReplaceAll(name, " ", "")
			Newfunction.name = name
			// fmt.Println(name, "--name of function")
			variables := nameSet[1][strings.Index(nameSet[1], "[")+1 : strings.Index(nameSet[1], "]")]
			variablesSet := strings.Split(variables, ",")
			// fmt.Println(variables, "-- variables of function", variablesSet)
			Newfunction.argumentCount = len(variablesSet)
			Newfunction.argumentDict = variablesSet
			Newfunction.funcVariableDict = make(map[string]string)
			if Newfunction.argumentCount > 0 {
				Newfunction.argumentState = true
			}
			for v := range variablesSet {
				// fmt.Println(variablesSet[v], "v-set----")
				// Newfunction.argumentDict[variablesSet[v]] = variablesSet[v]
				Newfunction.funcVariableDict[variablesSet[v]] = variablesSet[v]
			}
			// fmt.Println(Newfunction, "-- data code")
			functionDict[Newfunction.name] = Newfunction
			definitionName = Newfunction.name
			Newfunction.content = make([]string, 0)
			continue

		} else if definitionState == true {
			// fmt.Println(tok, "--def state---")
			if strings.Contains(tok, "def ") && strings.Contains(tok, "[end]") {
				definitionState = false
				contentDef := functionDict[definitionName]
				contentDef.content = append(functionDict[definitionName].content, tok)
				contentDef.contentLen = len(contentDef.content)
				functionDict[definitionName] = contentDef

			} else {
				contentDef := functionDict[definitionName]
				contentDef.content = append(functionDict[definitionName].content, tok)
				functionDict[definitionName] = contentDef
			}
			// fmt.Println(functionDict[definitionName], "--------- FUNC CODE")
			continue

		} else if strings.Contains(tok, "if") && strings.Contains(tok, "]") && strings.Contains(tok, "[") && conditionState == false {
			conditionState = true
			var ifElse ifelseCondition
			ifElse.head = tok
			conditionName = tok
			ifElse.content = append(ifElse.content, tok)
			ifElseDict[ifElse.head] = ifElse
			continue
		} else if conditionState {
			if strings.Contains(tok, "[end]") && strings.Contains(tok, "if") {
				ifelseCopy := ifElseDict[conditionName]
				ifelseCopy.content = append(ifElseDict[conditionName].content, tok)
				ifElseDict[conditionName] = ifelseCopy
				ifelse(ifElseDict[conditionName].content, "isMain")
				conditionState = false
				// ifElseDict[conditionName].content = append(ifElseDict[conditionName].content, tok)
			} else {
				ifelseCopy := ifElseDict[conditionName]
				ifelseCopy.content = append(ifElseDict[conditionName].content, tok)
				ifElseDict[conditionName] = ifelseCopy

			}
			continue
		} else if strings.Contains(tok, "]") && strings.Contains(tok, "loop") && strings.Contains(tok, "[") && !strings.Contains(tok, "[end]") && loopState == false {
			// fmt.Println("here")
			loopState = true
			loopName = tok
			var Newloop loop
			Newloop.name = loopName
			Newloop.content = append(Newloop.content, tok)
			loopDict[loopName] = Newloop
		} else if loopState == true {
			Newloop := loopDict[loopName]
			Newloop.content = append(Newloop.content, tok)
			loopDict[loopName] = Newloop
			if strings.Contains(tok, "[loop]") && strings.Contains(tok, "[end]") {
				loopState = false
				loopStructure(loopDict[loopName].content, "isMain")
			}

		} else if strings.Contains(tok, "[") && strings.Contains(tok, "]") && strings.Contains(tok, "=") && strings.Index(tok, "=") < strings.Index(tok, "[") {
			// fmt.Println("here --->")
			insertFunction(tok, "isMain")
		} else if strings.Contains(tok, "[") && strings.Contains(tok, "]") {
			functionProtocol(tok, "isMain")

		} else if strings.Contains(tok, "show") {
			showTok := strings.SplitAfter(tok, "show")
			if strings.Contains(showTok[0], "show") {
				showReal(tok)
			}
		} else if strings.Contains(tok, "?") && strings.Contains(tok, "=") {
			if strings.Index(tok, "?") < strings.Index(tok, "\"") {
				input := strings.Split(tok, "=")
				// fmt.Println(input)
				var variable string = ""
				fmt.Print(getPrompt(tok))
				scanIn := bufio.NewScanner(os.Stdin)
				scanIn.Scan()
				variable = scanIn.Text()
				vars := strings.ReplaceAll(input[0], " ", "")
				// fmt.Println(variable)
				variableDict[vars] = variable
				// fmt.Println(variableDict)
			}
		} else if strings.Contains(tok, "=") {
			varTok := strings.SplitAfter(tok, "=")
			if strings.Contains(varTok[0], "=") {
				insertVariable(tok)
			}
		}

	}
}
