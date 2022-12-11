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
		variable := getVariable(str[0:strings.LastIndex(str, ".")])
		return variableDict[variable]
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
		parseToken = parseToken[0:strings.LastIndex(parseToken, ".")]
		return parseString(strings.ReplaceAll(eval(parseToken), "\\n", "\n"))
	} else if oneStatement == true {
		variable := getVariable(str[0:strings.LastIndex(str, ".")])
		return functionDict[name].funcVariableDict[variable]
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
	}
}

// Function protocol for when functions are called

func functionProtocol(str string) {

	name := str
	name = name[0:strings.Index(name, "[")]
	name = strings.ReplaceAll(name, " ", "")
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
		Calledfunction.funcVariableDict[vars] = variablesSet[count]
		count += 1
	}
	// fmt.Println(Calledfunction, "<--- Called Function --->")

	for _, tok := range Calledfunction.content {
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

		if strings.Contains(tok, "def") && strings.Contains(tok, "[end]") {
			continue

		}

		if strings.Contains(tok, "[") && strings.Contains(tok, "]") {
			functionProtocol(tok)

		}

		if strings.Contains(tok, "show") {
			showTok := strings.SplitAfter(tok, "show")
			if strings.Contains(showTok[0], "show") {
				showRealFunc(tok, name)
			}
		}

		if strings.Contains(tok, "=") {
			varTok := strings.SplitAfter(tok, "=")
			if strings.Contains(varTok[0], "=") {
				insertVariableFunc(tok, name)
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
	//conditionState := false
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {

		tok := scanner.Text()
		// fmt.Println(tok)
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

		if strings.Contains(tok, "def") && strings.Contains(tok, "[") && strings.Contains(tok, "[") && definitionState == false {
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

		}

		if definitionState == true {
			// fmt.Println(tok, "--def state---")
			if strings.Contains(tok, "def") && strings.Contains(tok, "[end]") {
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

		}

		if strings.Contains(tok, "[") && strings.Contains(tok, "]") {
			functionProtocol(tok)

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
