package main

import (
	"bufio"
	"fmt"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// eval function
// used for evaluation expression

const (
	plus     = "+"
	minus    = "-"
	divide   = "/"
	multiply = "*"
)

// Evaluate operations and expressions
func eval(s string) string {
	fs := token.NewFileSet()
	tv, err := types.Eval(fs, nil, token.NoPos, s)
	if err != nil {
		panic(err)
	}
	return tv.Value.String()
}

// Eval different types of data in order to parser data easier
// determines the type of data that needs to be evaluated
// types are int, string, bool, var , expression
func evalType(s string, state string) string {

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
			} else if string(s[i]) == "." || string(s[i]) == minus || string(s[i]) == plus || string(s[i]) == divide ||
				string(s[i]) == multiply || string(s[i]) == "," || string(s[i]) == " " || string(s[i]) == ")" || string(s[i]) == "(" && instring == 0 {
				continue

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
			if state == "isMain" {
				// fmt.Println(varTok)
				// fmt.Println(reflect.TypeOf(variableDict[varTok]).Name())
				return "Var"
			} else {
				return "Var"
			}

		}
		return "Exp"

	}

	return tv.Value.Kind().String()
}

// variable dictionary
// used for variables that will be used for isMain scope
// depending on the scope of the variable variableDict will be used
var variableDict = make(map[string]interface{})

// Insert variable into the variable dictionary
// may change this functionality but so far it works well
func insertVariable(variableToken string, state string) {
	newtoken := variableToken
	varToken := strings.Split(newtoken, "=")

	varToken[0] = strings.ReplaceAll(varToken[0], " ", "")
	if evalType(varToken[1], state) == "String" {
		variableDict[string(varToken[0])] = eval(varToken[1])
	} else if evalType(varToken[1], state) == "Int" {
		variableDict[string(varToken[0])] = eval(varToken[1])
	} else if evalType(varToken[1], state) == "Float" {
		variableDict[string(varToken[0])] = eval(varToken[1])
	} else if evalType(varToken[1], state) == "Bool" {
		variableDict[string(varToken[0])] = eval(varToken[1])
	} else if evalType(varToken[1], state) == "Var" {
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
				newVarTok[v] = variableDict[newVarTok[v]].(string)
			}
		}
		VarTok := strings.Join(newVarTok, " ")
		variableDict[string(varToken[0])] = eval(VarTok)

	}
}

// called when variable is assigned to a function(s)
// special use case for if else and loop structures as well
func insertFunction(function string, state string) {
	if state == "isMain" {
		funcVar := strings.SplitAfter(function, "=")
		variableDict[getVariable(funcVar[0])] = funcVar[1]
		functionProtocol(function, state)
	} else {
		funcVar := strings.SplitAfter(function, "=")
		functionDict[state].funcVariableDict[getVariable(funcVar[0])] = funcVar[1]
		functionProtocol(function, state)

	}
}

// Used to evaluate expressions and different expression cases
// This is used only for non variable expressions
func evalExpression(str string) string {

	inString := 0
	newShow := str
	newString := ""
	// check if the string has any commas outside string or if it's a concat

	isConcat := isConcatExp(str)
	if isConcat == true {
		parseToken := str[0:strings.LastIndex(str, ".")]
		return strings.ReplaceAll(eval(parseToken), "\\n", "\n")
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

// Parses out variables from variable expressions
// expressions such as a+b , or a + bubble[10]
// variable expressions require a little more parsing than
// regular expressions such as 10 + 12 ...
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

		} else if s[i] == 91 && instring == 0 {
			return varTok
		} else {
			continue
		}
	}
	return varTok
}

// check and see if a statement only has one variable
// example show a . returns true
func isOneVariable(str string) bool {
	inString := 0
	Statement := true

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
// example show a + b . returns true , show a + b - c , returns false .
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

// Evaluate variable expressions by making them ready for evaluation
func getevalVar(str string) string {
	arg := ""
	newExp := ""
	newString := ""
	place := 0
	newPlace := 0
	InnerinString := 0
	anotherPlace := place
	newExp = str
	for newPlace = place; newPlace < len(str)-1; newPlace++ {
		if string(str[newPlace]) == "\"" {
			InnerinString += 1
			if InnerinString > 1 {
				InnerinString = 0
			}
			continue
		} else if InnerinString == 0 && string(str[newPlace]) == plus || string(str[newPlace]) == minus || string(str[newPlace]) == divide || string(str[newPlace]) == multiply {

			if strings.Contains(newExp, "[") && strings.Contains(newExp, "]") && strings.Contains(str[anotherPlace:newPlace], "[") && strings.Contains(str[anotherPlace:newPlace], "]") {
				funcshowState = true
				funcParsed := getVariable(str[anotherPlace:newPlace])
				functionProtocol(str[strings.Index(str, string(funcParsed[0])):newPlace], "isMain")
				newExp = strings.ReplaceAll(newExp, str[strings.Index(str, string(funcParsed[0])):newPlace], funcShowReturn)
				funcshowState = false
				funcShowReturn = ""
				anotherPlace = newPlace
			} else {
				variable := getVariable(str[anotherPlace:newPlace])
				if variable == "" {
					anotherPlace = newPlace
				} else {
					arg = variableDict[variable].(string)
					newExp = strings.ReplaceAll(newExp, variable, arg)
					anotherPlace = newPlace
				}

			}

		} else {
			continue
		}
	}

	if strings.Contains(newExp, "[") && strings.Contains(newExp, "]") && strings.Contains(str[anotherPlace:newPlace], "[") && strings.Contains(str[anotherPlace:newPlace], "]") {
		funcshowState = true
		funcParsed := getVariable(str[anotherPlace:newPlace])
		functionProtocol(str[strings.Index(str, string(funcParsed[0])):newPlace], "isMain")
		newExp = strings.ReplaceAll(newExp, str[strings.Index(str, string(funcParsed[0])):newPlace], funcShowReturn)
		funcshowState = false
		funcShowReturn = ""
		anotherPlace = newPlace
	} else {
		variable := getVariable(str[anotherPlace:newPlace])
		if variable == "" {
			anotherPlace = newPlace
		} else {
			arg = variableDict[variable].(string)
			newExp = strings.ReplaceAll(newExp, variable, arg)
			anotherPlace = newPlace
		}

	}
	arg = eval(newExp)
	newString += parseString(arg)
	return newString

}

// Evaluate variable expressions by making them ready for evaluation
// difference is that it parses expressions that end with a period
// logic is similar to the previous expression
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
			if strings.Contains(newExp, "[") && strings.Contains(newExp, "]") && strings.Contains(str[anotherPlace:newPlace], "[") && strings.Contains(str[anotherPlace:newPlace], "]") {
				funcshowState = true
				functionProtocol(str[anotherPlace:newPlace], "isMain")
				newExp = strings.ReplaceAll(newExp, string(str[anotherPlace:newPlace]), funcShowReturn)
				funcshowState = false
				funcShowReturn = ""
				anotherPlace = newPlace
			} else {
				variable := getVariable(str[anotherPlace:newPlace])
				if variable == "" {
					anotherPlace = newPlace
				} else {
					arg = variableDict[variable].(string)
					newExp = strings.ReplaceAll(newExp, variable, arg)
					anotherPlace = newPlace
				}

			}
		} else if InnerinString == 0 && newPlace == len(str)-1 {
			if strings.Contains(newExp, "[") && strings.Contains(newExp, "]") && strings.Contains(str[anotherPlace:newPlace], "[") && strings.Contains(str[anotherPlace:newPlace], "]") {
				funcshowState = true
				functionProtocol(str[anotherPlace:newPlace], "isMain")
				newExp = strings.ReplaceAll(newExp, string(str[anotherPlace:newPlace]), funcShowReturn)
				funcshowState = false
				funcShowReturn = ""
				anotherPlace = newPlace
			} else {
				variable := getVariable(str[anotherPlace:newPlace])
				if variable == "" {
					anotherPlace = newPlace
				} else {
					arg = variableDict[variable].(string)
					newExp = strings.ReplaceAll(newExp, variable, arg)
					anotherPlace = newPlace
				}

			}

		} else {
			continue
		}

	}

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
	oneStatement = isOneVariable(str)
	isConcat = isConcatExp(str)
	if isConcat == true {
		parseToken := str
		place := 0
		placeAfter := 0
		for count := 0; count <= strings.LastIndex(str, "."); count++ {
			if string(str[count]) == "\"" {
				inString += 1
				if inString > 1 {
					inString = 0
				}
				continue
			} else if string(str[count]) == plus && inString == 0 {
				if placeAfter == 0 {
					if strings.Contains(str, "[") && strings.Contains(str, "]") && strings.Contains(str[placeAfter:count-1], "[") && strings.Contains(str[placeAfter:count-1], "]") {
						funcshowState = true
						functionProtocol(str[0:count-1], "isMain")
						parseToken = strings.ReplaceAll(parseToken, string(str[0:count-1]), funcShowReturn)
						funcshowState = false
						funcShowReturn = ""
					} else {
						variable := getVariable(parseToken[place:count])
						if variable == "" {
							place = count + 1
						} else {
							parseToken = strings.ReplaceAll(parseToken, variable, variableDict[variable].(string))
							place = count + 1
						}
					}
					placeAfter = count + 1

				} else {
					if strings.Contains(parseToken, "[") && strings.Contains(parseToken, "]") && strings.Contains(str[placeAfter:count-1], "[") && strings.Contains(str[placeAfter:count-1], "]") {
						funcshowState = true
						functionProtocol(str[placeAfter:count-1], "isMain")
						parseToken = strings.ReplaceAll(parseToken, string(str[placeAfter:count-1]), funcShowReturn)
						funcshowState = false
						funcShowReturn = ""
					} else {
						variable := getVariable(str[placeAfter : count-1])
						if variable == "" {
							place = count + 1
						} else {

							parseToken = strings.ReplaceAll(parseToken, variable, variableDict[variable].(string))
							place = count + 1
						}
					}
					placeAfter = count + 1

				}

			} else if count == strings.LastIndex(str, ".") {
				if strings.Contains(parseToken, "[") && strings.Contains(parseToken, "]") && strings.Contains(str[placeAfter:count-1], "[") && strings.Contains(str[placeAfter:count-1], "]") {
					funcshowState = true
					functionProtocol(str[placeAfter:count-1], "isMain")
					parseToken = strings.ReplaceAll(parseToken, string(str[placeAfter:count-1]), funcShowReturn)
					funcshowState = false
					funcShowReturn = ""

				} else {
					variable := getVariable(str[placeAfter:count])
					if variable == "" {
						place = count + 1
					} else {
						parseToken = strings.ReplaceAll(parseToken, variable, variableDict[variable].(string))
						place = count + 1
					}

				}
			}
		}
		parseToken = parseToken[0:strings.LastIndex(parseToken, ".")]
		return parseString(strings.ReplaceAll(eval(parseToken), "\\n", "\n"))
	} else if oneStatement == true {
		variable := getVariable(str[0:strings.LastIndex(str, ".")])
		if !strings.Contains(str, ",") {
			if strings.Contains(str, "[") && strings.Contains(str, "]") {
				funcshowState = true
				functionProtocol(str, "isMain")
				funcshowState = false
				return funcShowReturn
			} else {

				return variableDict[variable].(string)
			}
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
					if evalType(newShow[place:count], "isMain") == "Var" {
						oneVar := isOneVariable(newShow[place:count])
						if oneVar == true {
							if strings.Contains(newShow[place:count], "[") && strings.Contains(newShow[place:count], "]") {
								funcshowState = true
								functionProtocol(newShow[place:count], "isMain")
								newString += funcShowReturn

							} else {
								variable := getVariable(newShow[place:count])
								arg = variableDict[variable].(string)
								newString += parseString(arg)

							}

						} else {
							newString += getevalVar(newShow[place:count])
						}

					} else {
						if evalType(newShow[place:count], "isMain") == "Exp" {
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
					if evalType(newShow[place:count], "isMain") == "Var" {
						oneVar := isOneVariable(newShow[place:count])
						if oneVar == true {
							if strings.Contains(newShow[place:count], "[") && strings.Contains(newShow[place:count], "]") {
								funcshowState = true
								functionProtocol(newShow[place:count], "isMain")
								newString += funcShowReturn
							} else {
								variable := getVariable(newShow[place:count])
								arg = variableDict[variable].(string)
								newString += parseString(arg)
							}
						} else {
							newString += getevalVarPeriod(newShow[place:count])
						}
					} else {
						if evalType(newShow[place:count], "isMain") == "Exp" {
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
				if evalType(newShow[place:count], "isMain") == "Var" {
					oneVar := isOneVariable(newShow[place:count])
					if oneVar == true {
						variable := getVariable(newShow[place:count])
						arg = variableDict[variable].(string)
						newString += parseString(arg)
					} else {
						newString += getevalVar(newShow[place:count])
					}

				} else {
					if evalType(newShow[place:count], "isMain") == "Exp" {
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
				if evalType(newShow[place:count], "isMain") == "Var" {
					oneVar := isOneVariable(newShow[place:count])
					if oneVar == true {
						variable := getVariable(newShow[place:count])
						arg = variableDict[variable].(string)
						newString += parseString(arg)
					} else {
						newString += getevalVarPeriod(newShow[place:count])
					}
				} else {
					if evalType(newShow[place:count], "isMain") == "Exp" {
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
// to avoid strings that appear with "quotes and not with quotes"
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
// uses evalType func to show eval expression types
// then for each case sends to proper showing mechanism
func showReal(str string, state string) {

	showTok := strings.SplitAfterN(str, "show", 2)
	parseToken := showTok[1][0:strings.LastIndex(showTok[1], ".")]
	parseToken = strings.ReplaceAll(parseToken, "\\n", "\n")
	if evalType(parseToken, state) == "Int" {
		fmt.Println(eval(parseToken))
	} else if evalType(parseToken, state) == "Float" {
		fmt.Println(eval(parseToken))
	} else if evalType(parseToken, state) == "String" {
		fmt.Println(parseString(eval(parseToken)))
	} else if evalType(parseToken, state) == "Var" {
		fmt.Println(evalDataExpressions(showTok[1], state))
	} else if evalType(parseToken, state) == "Exp" {
		fmt.Println(evalExpression(showTok[1]))
	} else if evalType(parseToken, state) == "Bool" {
		fmt.Println(evalExpression(showTok[1]))
	} else if evalType(parseToken, state) == "List" {
		fmt.Println(evalDataExpressions(showTok[1], state))
	} else if evalType(parseToken, state) == "Set" {
		fmt.Println(evalDataExpressions(showTok[1], state))
	} else if evalType(parseToken, state) == "Map" {
		fmt.Println(evalDataExpressions(showTok[1], state))

	}
}

// check errors
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
	funcVariableDict map[string]interface{}
	argumentState    bool
	argumentDict     []string
	argumentCount    int
	content          []string
	name             string
	contentLen       int
}

// Evaluate variable expressions by making them ready for evaluation
func getevalVarFunc(str string, name string) string {
	arg := ""
	newExp := ""
	newString := ""
	place := 0

	newPlace := 0
	InnerinString := 0
	anotherPlace := place
	newExp = str
	for newPlace = place; newPlace < len(str)-1; newPlace++ {
		if string(str[newPlace]) == "\"" {
			InnerinString += 1
			if InnerinString > 1 {
				InnerinString = 0
			}
			continue
		} else if InnerinString == 0 && string(str[newPlace]) == plus || string(str[newPlace]) == minus || string(str[newPlace]) == divide || string(str[newPlace]) == multiply {

			if strings.Contains(newExp, "[") && strings.Contains(newExp, "]") && strings.Contains(str[anotherPlace:newPlace], "[") && strings.Contains(str[anotherPlace:newPlace], "]") {
				funcshowState = true
				functionProtocol(str[anotherPlace:newPlace], name)
				newExp = strings.ReplaceAll(newExp, string(str[anotherPlace:newPlace]), funcShowReturn)
				funcshowState = false
				funcShowReturn = ""
				anotherPlace = newPlace
			} else {
				variable := getVariable(str[anotherPlace:newPlace])
				if variable == "" {
					anotherPlace = newPlace
				} else {
					arg = functionDict[name].funcVariableDict[variable].(string)
					newExp = strings.ReplaceAll(newExp, variable, arg)
					anotherPlace = newPlace
				}

			}
		} else {
			continue
		}
	}
	if strings.Contains(newExp, "[") && strings.Contains(newExp, "]") && strings.Contains(str[anotherPlace:newPlace], "[") && strings.Contains(str[anotherPlace:newPlace], "]") {
		funcshowState = true
		funcParsed := getVariable(str[anotherPlace:newPlace])
		functionProtocol(str[strings.Index(str, string(funcParsed[0])):newPlace], name)
		newExp = strings.ReplaceAll(newExp, str[strings.Index(str, string(funcParsed[0])):newPlace], funcShowReturn)
		funcshowState = false
		funcShowReturn = ""
		anotherPlace = newPlace
	} else {
		variable := getVariable(str[anotherPlace:newPlace])
		if variable == "" {
			anotherPlace = newPlace
		} else {
			arg = functionDict[name].funcVariableDict[variable].(string)
			newExp = strings.ReplaceAll(newExp, variable, arg)
			anotherPlace = newPlace
		}

	}
	arg = eval(newExp)
	newString += parseString(arg)
	return newString

}

// Evaluate variable expressions by making them ready for evaluation
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
			if strings.Contains(newExp, "[") && strings.Contains(newExp, "]") && strings.Contains(str[anotherPlace:newPlace], "[") && strings.Contains(str[anotherPlace:newPlace], "]") {
				funcshowState = true
				functionProtocol(str[anotherPlace:newPlace], name)
				newExp = strings.ReplaceAll(newExp, string(str[anotherPlace:newPlace]), funcShowReturn)
				funcshowState = false
				funcShowReturn = ""
				anotherPlace = newPlace
			} else {
				variable := getVariable(str[anotherPlace:newPlace])
				if variable == "" {
					anotherPlace = newPlace
				} else {
					arg = functionDict[name].funcVariableDict[variable].(string)
					newExp = strings.ReplaceAll(newExp, variable, arg)
					anotherPlace = newPlace
				}

			}
		} else if InnerinString == 0 && newPlace == len(str)-1 {
			if strings.Contains(newExp, "[") && strings.Contains(newExp, "]") && strings.Contains(str[anotherPlace:newPlace], "[") && strings.Contains(str[anotherPlace:newPlace], "]") {
				funcshowState = true
				functionProtocol(str[anotherPlace:newPlace], name)
				newExp = strings.ReplaceAll(newExp, string(str[anotherPlace:newPlace]), funcShowReturn)
				funcshowState = false
				funcShowReturn = ""
				anotherPlace = newPlace
			} else {
				variable := getVariable(str[anotherPlace:newPlace])
				if variable == "" {
					anotherPlace = newPlace
				} else {
					arg = functionDict[name].funcVariableDict[variable].(string)
					newExp = strings.ReplaceAll(newExp, variable, arg)
					anotherPlace = newPlace
				}

			}

		} else {
			continue
		}

	}

	arg = eval(newExp)
	newString += parseString(arg)
	return newString

}

// function version of adding a variable to the variable dictionary of a function
// function version of insertVariable
func insertVariableFunc(variableToken string, name string) {
	newtoken := variableToken
	varToken := strings.Split(newtoken, "=")
	varToken[0] = strings.ReplaceAll(varToken[0], " ", "")
	if evalType(varToken[1], name) == "String" {
		functionDict[name].funcVariableDict[string(varToken[0])] = eval(varToken[1])
	} else if evalType(varToken[1], name) == "Int" {
		functionDict[name].funcVariableDict[string(varToken[0])] = eval(varToken[1])
	} else if evalType(varToken[1], name) == "Float" {
		functionDict[name].funcVariableDict[string(varToken[0])] = eval(varToken[1])
	} else if evalType(varToken[1], name) == "Bool" {
		functionDict[name].funcVariableDict[string(varToken[0])] = eval(varToken[1])
	} else if evalType(varToken[1], name) == "Var" {
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
				newVarTok[v] = functionDict[name].funcVariableDict[newVarTok[v]].(string)
			}
		}
		VarTok := strings.Join(newVarTok, " ")
		functionDict[name].funcVariableDict[string(varToken[0])] = eval(VarTok)

	}
}

// parse and evaluate variables of functions
// function version of evalVarExpression
func evalVarExpressionFunc(str string, name string) string {
	inString := 0
	newShow := str
	newString := ""
	// check if the string has any commas outside string or if it's a concat
	var isConcat bool
	var oneStatement bool
	oneStatement = isOneVariable(str)
	isConcat = isConcatExp(str)
	if isConcat == true {
		parseToken := str
		place := 0
		placeAfter := 0
		for count := 0; count <= strings.LastIndex(str, "."); count++ {
			if string(str[count]) == "\"" {
				inString += 1
				if inString > 1 {
					inString = 0
				}
				continue
			} else if string(str[count]) == plus && inString == 0 {
				if placeAfter == 0 {
					if strings.Contains(parseToken, "[") && strings.Contains(parseToken, "]") && strings.Contains(str[placeAfter:count-1], "[") && strings.Contains(str[placeAfter:count-1], "]") {
						funcshowState = true
						functionProtocol(str[0:count-1], name)
						parseToken = strings.ReplaceAll(parseToken, string(str[0:count-1]), funcShowReturn)
						funcshowState = false
						funcShowReturn = ""
					} else {
						variable := getVariable(parseToken[place:count])
						if variable == "" {
							place = count + 1
						} else {
							parseToken = strings.ReplaceAll(parseToken, variable, functionDict[name].funcVariableDict[variable].(string))
							place = count + 1
						}
					}
					placeAfter = count + 1

				} else {
					if strings.Contains(parseToken, "[") && strings.Contains(parseToken, "]") && strings.Contains(str[placeAfter:count-1], "[") && strings.Contains(str[placeAfter:count-1], "]") {
						funcshowState = true
						functionProtocol(str[placeAfter:count-1], name)
						parseToken = strings.ReplaceAll(parseToken, string(str[placeAfter:count-1]), funcShowReturn)
						funcshowState = false
						funcShowReturn = ""
					} else {
						variable := getVariable(str[placeAfter : count-1])
						if variable == "" {
							place = count + 1
						} else {

							parseToken = strings.ReplaceAll(parseToken, variable, functionDict[name].funcVariableDict[variable].(string))
							place = count + 1
						}
					}
					placeAfter = count + 1

				}
			} else if count == strings.LastIndex(str, ".") {
				if placeAfter == 0 {
					if strings.Contains(str, "[") && strings.Contains(str, "]") && strings.Contains(str[placeAfter:count-1], "[") && strings.Contains(str[placeAfter:count-1], "]") {
						funcshowState = true
						functionProtocol(str[0:count-1], name)
						parseToken = strings.ReplaceAll(parseToken, string(str[0:count-1]), funcShowReturn)
						funcshowState = false
						funcShowReturn = ""
					} else {
						variable := getVariable(parseToken[place:count])
						if variable == "" {
							place = count + 1
						} else {
							parseToken = strings.ReplaceAll(parseToken, variable, functionDict[name].funcVariableDict[variable].(string))
							place = count + 1
						}
					}
					placeAfter = count + 1

				} else {
					if strings.Contains(parseToken, "[") && strings.Contains(parseToken, "]") && strings.Contains(str[placeAfter:count-1], "[") && strings.Contains(str[placeAfter:count-1], "]") {
						funcshowState = true
						functionProtocol(str[placeAfter:count-1], name)
						parseToken = strings.ReplaceAll(parseToken, string(str[placeAfter:count-1]), funcShowReturn)
						funcshowState = false
						funcShowReturn = ""
					} else {
						variable := getVariable(str[placeAfter : count-1])
						if variable == "" {
							place = count + 1
						} else {
							parseToken = strings.ReplaceAll(parseToken, variable, functionDict[name].funcVariableDict[variable].(string))
							place = count + 1
						}
					}
					placeAfter = count + 1

				}

			}
		}

		parseToken = parseToken[0:strings.LastIndex(parseToken, ".")]

		return parseString(strings.ReplaceAll(eval(parseToken), "\\n", "\n"))
	} else if oneStatement == true {
		variable := getVariable(str[0:strings.LastIndex(str, ".")])
		if !strings.Contains(str, ",") {
			if strings.Contains(str, "[") && strings.Contains(str, "]") {
				funcshowState = true
				functionProtocol(str, name)
				funcshowState = false
				return funcShowReturn
			} else {
				return functionDict[name].funcVariableDict[variable].(string)
			}

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
					if evalType(newShow[place:count], name) == "Var" {
						oneVar := isOneVariable(newShow[place:count])
						if oneVar == true {
							if strings.Contains(newShow[place:count], "[") && strings.Contains(newShow[place:count], "]") {
								funcshowState = true
								functionProtocol(newShow[place:count], name)
								newString += funcShowReturn

							} else {
								variable := getVariable(newShow[place:count])
								arg = functionDict[name].funcVariableDict[variable].(string)
								newString += parseString(arg)
							}

						} else {
							newString += getevalVarFunc(newShow[place:count], name)
						}

					} else {
						if evalType(newShow[place:count], name) == "Exp" {
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
					if evalType(newShow[place:count], name) == "Var" {
						oneVar := isOneVariable(newShow[place:count])
						if oneVar == true {
							if strings.Contains(newShow[place:count], "[") && strings.Contains(newShow[place:count], "]") {
								funcshowState = true
								functionProtocol(newShow[place:count], name)
								newString += funcShowReturn

							} else {
								variable := getVariable(newShow[place:count])
								arg = functionDict[name].funcVariableDict[variable].(string)
								newString += parseString(arg)
							}

						} else {
							newString += getevalVarPeriodFunc(newShow[place:count], name)
						}
					} else {
						if evalType(newShow[place:count], name) == "Exp" {
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
				if evalType(newShow[place:count], name) == "Var" {
					oneVar := isOneVariable(newShow[place:count])
					if oneVar == true {
						if strings.Contains(newShow[place:count], "[") && strings.Contains(newShow[place:count], "]") {
							funcshowState = true
							functionProtocol(newShow[place:count], name)
							newString += funcShowReturn

						} else {
							variable := getVariable(newShow[place:count])
							arg = functionDict[name].funcVariableDict[variable].(string)
							newString += parseString(arg)
						}
					} else {
						newString += getevalVarFunc(newShow[place:count], name)
					}

				} else {
					if evalType(newShow[place:count], name) == "Exp" {
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
				if evalType(newShow[place:count], name) == "Var" {
					oneVar := isOneVariable(newShow[place:count])
					if oneVar == true {
						if strings.Contains(newShow[place:count], "[") && strings.Contains(newShow[place:count], "]") {
							funcshowState = true
							functionProtocol(newShow[place:count], name)
							newString += funcShowReturn

						} else {
							variable := getVariable(newShow[place:count])
							arg = functionDict[name].funcVariableDict[variable].(string)
							newString += parseString(arg)
						}
					} else {
						newString += getevalVarPeriodFunc(newShow[place:count], name)
					}
				} else {
					if evalType(newShow[place:count], name) == "Exp" {
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

// function version of showReal
func showRealFunc(str string, name string) {

	showTok := strings.SplitAfterN(str, "show", 2)
	parseToken := showTok[1][0:strings.LastIndex(showTok[1], ".")]
	parseToken = strings.ReplaceAll(parseToken, "\\n", "\n")
	if evalType(parseToken, name) == "Int" {
		fmt.Println(eval(parseToken))
	} else if evalType(parseToken, name) == "Float" {
		fmt.Println(eval(parseToken))
	} else if evalType(parseToken, name) == "String" {
		fmt.Println(parseString(eval(parseToken)))
	} else if evalType(parseToken, name) == "Var" {
		fmt.Println(evalDataExpressionFunc(showTok[1], name))
	} else if evalType(parseToken, name) == "Exp" {
		fmt.Println(evalExpression(showTok[1]))
	} else if evalType(parseToken, name) == "Bool" {
		fmt.Println(evalExpression(showTok[1]))
	} else if evalType(parseToken, name) == "Var" {
		fmt.Println(evalDataExpressionFunc(showTok[1], name))
	} else if evalType(parseToken, name) == "Var" {
		fmt.Println(evalDataExpressionFunc(showTok[1], name))
	} else if evalType(parseToken, name) == "Var" {
		fmt.Println(evalDataExpressionFunc(showTok[1], name))

	}
}

// show function return
// used for function expressions and functions that return a value
var funcShowReturn string

// boolean for state
// used for function expressions and that return a value
var funcshowState bool

// Function protocol for when functions are called
// this is used to set up the function definition call order,
// variables , arguments , return function ability and etc
func functionProtocol(str string, state string) {
	// organizes function state and etc
	conditionState := false
	conditionName := ""
	loopState := false
	loopName := ""
	name := str
	// parse name of function
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
	// sets called function to function definition found in function dictionary
	Calledfunction := functionDict[name]

	// checks to make sure name found is the same of function
	if Calledfunction.name != name {
		fmt.Println(name)
		fmt.Println("function Error")
		// potentially add function check for other functions
	}

	// change the variables in arguments to add variable to function scope
	variables := str[strings.Index(str, "[")+1 : strings.Index(str, "]")]
	variablesSet := strings.Split(variables, ",")
	count := 0
	for _, vars := range Calledfunction.argumentDict {

		if isOneVariable(variablesSet[count]) {
			if state == "isMain" {
				_, isPresent := variableDict[getVariable(variablesSet[count])]

				if isPresent {
					Calledfunction.funcVariableDict[vars] = variableDict[getVariable(variablesSet[count])].(string)

				} else {
					Calledfunction.funcVariableDict[vars] = variablesSet[count]
				}
			} else {
				_, isPresent := functionDict[state].funcVariableDict[getVariable(variablesSet[count])]
				if isPresent {

					Calledfunction.funcVariableDict[vars] = functionDict[state].funcVariableDict[getVariable(variablesSet[count])]
				} else {

					Calledfunction.funcVariableDict[vars] = variablesSet[count]
				}

			}

		}

		count += 1
	}
	// loop through the function statements
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
			// return function actions
			returnCode := strings.Split(tok, "return ")
			varState := false
			if state == "isMain" {
				// variableDict[]
				for key, element := range variableDict {
					if strings.Contains(element.(string), name) {
						variableDict[key] = Calledfunction.funcVariableDict[getVariable(returnCode[1])]
						varState = true
						break
					}
				}
				if varState == false {
					if funcshowState {
						funcShowReturn = Calledfunction.funcVariableDict[getVariable(returnCode[1])].(string)
						break
					} else {
						fmt.Println("variable error, or return error :")
					}
				}

			} else {
				// functVariableDict
				for key, element := range functionDict[state].funcVariableDict {

					if strings.Contains(element.(string), Calledfunction.name) {
						functionDict[state].funcVariableDict[key] = Calledfunction.funcVariableDict[getVariable(returnCode[1])]
						varState = true
						break

					}
				}
				if varState == false {
					if funcshowState {
						funcShowReturn = Calledfunction.funcVariableDict[getVariable(returnCode[1])].(string)
						break
					} else {
						fmt.Println("variable error, or return error :")
					}
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
			} else {
				ifelseCopy := ifElseDict[conditionName]
				ifelseCopy.content = append(ifElseDict[conditionName].content, tok)
				ifElseDict[conditionName] = ifelseCopy

			}
			continue
		} else if strings.Contains(tok, "]") && strings.Contains(tok, "loop") && strings.Contains(tok, "[") && !strings.Contains(tok, "[end]") && loopState == false {
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

			if strings.Contains(tok, "list") && getVariable(strings.Split(tok, "=")[1]) == "list" {
				dataStructureProtocol("list", name, tok)
			} else if strings.Contains(tok, "map") && getVariable(strings.Split(tok, "=")[1]) == "map" {
				dataStructureProtocol("map", name, tok)
			} else if strings.Contains(tok, "set") && getVariable(strings.Split(tok, "=")[1]) == "set" {
				dataStructureProtocol("set", name, tok)
			} else {
				insertFunction(tok, name)
			}
		} else if strings.Contains(tok, "[") && strings.Contains(tok, "]") && strings.LastIndex(tok, "]") > strings.LastIndex(tok, ".") {
			functionProtocol(tok, name)
		} else if strings.Contains(tok, "show") {
			showTok := strings.SplitAfter(tok, "show")
			if strings.Contains(showTok[0], "show") {
				showRealFunc(tok, name)
			}
		} else if strings.Contains(tok, "?") && strings.Contains(tok, "=") {
			if strings.Index(tok, "?") < strings.Index(tok, "\"") {
				input := strings.Split(tok, "=")
				var variable string = ""
				fmt.Print(getPrompt(tok))
				scanIn := bufio.NewScanner(os.Stdin)
				scanIn.Scan()
				variable = scanIn.Text()
				vars := strings.ReplaceAll(input[0], " ", "")
				Calledfunction.funcVariableDict[vars] = variable
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

// if else dictionary used to hold if else structure
var ifElseDict = make(map[string]ifelseCondition)

// if else statements parsed and executed
// ability to get one nested level
// working on a better algorithm
func ifelse(token []string, state string) {
	conditionState := false
	conditionMet := false
	currentCondition := false
	nested := false
	nestedState := false
	outerCondition := false
	for _, tok := range token {
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

// If else parser function is able to parse variables , functions , and literals
// makes if else statements ready for execution...
func ifelseParser(tok string, state string) string {
	newExpression := ""
	expression := tok[strings.Index(tok, "]")+1 : strings.LastIndex(tok, "[")]
	for _, word := range expression {
		if string(word) == ">" || string(word) == "<" || string(word) == "=" || string(word) == "&" || string(word) == "|" || string(word) == "!" {
			newExpression += string(word)
			variable := getVariable(newExpression)
			if state == "isMain" && variable != "" {
				// add logic in for functions here
				if strings.Contains(newExpression, "[") && strings.Contains(newExpression, "]") {
					funcshowState = true
					funcParsed := getVariable(newExpression)
					// with getVariable it will only get function name and not arguments have to parse out the rest of the function .
					// code below to get the other part of the function
					functionProtocol(newExpression[strings.Index(newExpression, funcParsed):strings.Index(newExpression, "]")+1], "isMain")
					newExpression = strings.ReplaceAll(newExpression, newExpression[strings.Index(newExpression, funcParsed):strings.Index(newExpression, "]")+1], funcShowReturn)
					funcshowState = false
					funcShowReturn = ""
				} else {
					newExpression = strings.ReplaceAll(newExpression, variable, variableDict[variable].(string))
				}
			} else if state != "isMain" && variable != "" {
				// add logic in for functions here
				if strings.Contains(newExpression, "[") && strings.Contains(newExpression, "]") {
					funcshowState = true
					funcParsed := getVariable(newExpression)
					// with getVariable it will only get function name and not arguments have to parse out the rest of the function .
					// code below to get the other part of the function
					if "" != getVariable(newExpression[strings.Index(newExpression, "[")+1:strings.Index(newExpression, "]")]) {
						functionProtocol(newExpression[strings.Index(newExpression, funcParsed):strings.Index(newExpression, "]")+1], state)
						newExpression = strings.ReplaceAll(newExpression, newExpression[strings.Index(newExpression, funcParsed):strings.Index(newExpression, "]")+1], funcShowReturn)
						funcshowState = false
						funcShowReturn = ""

					} else {
						functionProtocol(newExpression[strings.Index(newExpression, funcParsed):strings.Index(newExpression, "]")+1], "isMain")
						newExpression = strings.ReplaceAll(newExpression, newExpression[strings.Index(newExpression, funcParsed):strings.Index(newExpression, "]")+1], funcShowReturn)
						funcshowState = false
						funcShowReturn = ""

					}

				} else {
					newExpression = strings.ReplaceAll(newExpression, variable, functionDict[state].funcVariableDict[variable].(string))
				}
			}
		} else {
			newExpression += string(word)
		}
	}
	variable := getVariable(newExpression)
	if state == "isMain" && variable != "" {
		if strings.Contains(newExpression, "[") && strings.Contains(newExpression, "]") {
			funcshowState = true
			funcParsed := getVariable(newExpression)
			// with getVariable it will only get function name and not arguments have to parse out the rest of the function .
			// code below to get the other part of the function
			functionProtocol(newExpression[strings.Index(newExpression, funcParsed):strings.Index(newExpression, "]")+1], "isMain")
			newExpression = strings.ReplaceAll(newExpression, newExpression[strings.Index(newExpression, funcParsed):strings.Index(newExpression, "]")+1], funcShowReturn)
			funcshowState = false
			funcShowReturn = ""
		} else {
			newExpression = strings.ReplaceAll(newExpression, variable, variableDict[variable].(string))
		}
	} else if state != "isMain" && variable != "" {
		if strings.Contains(newExpression, "[") && strings.Contains(newExpression, "]") {
			funcshowState = true
			funcParsed := getVariable(newExpression)
			// with getVariable it will only get function name and not arguments have to parse out the rest of the function .
			// code below to get the other part of the function
			// Learned also that the scope doesn't have to reference another functions scope if it has a literal value.
			if "" != getVariable(newExpression[strings.Index(newExpression, "[")+1:strings.Index(newExpression, "]")]) {
				functionProtocol(newExpression[strings.Index(newExpression, funcParsed):strings.Index(newExpression, "]")+1], state)
				newExpression = strings.ReplaceAll(newExpression, newExpression[strings.Index(newExpression, funcParsed):strings.Index(newExpression, "]")+1], funcShowReturn)
				funcshowState = false
				funcShowReturn = ""

			} else {
				functionProtocol(newExpression[strings.Index(newExpression, funcParsed):strings.Index(newExpression, "]")+1], "isMain")
				newExpression = strings.ReplaceAll(newExpression, newExpression[strings.Index(newExpression, funcParsed):strings.Index(newExpression, "]")+1], funcShowReturn)
				funcshowState = false
				funcShowReturn = ""

			}
		} else {
			newExpression = strings.ReplaceAll(newExpression, variable, functionDict[state].funcVariableDict[variable].(string))
		}
	}

	return newExpression
}

// callCode used to execute code independent of isMain and function Protocol
// uses need for when you don't want to recursively call certain functions
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
		variables := nameSet[1][strings.Index(nameSet[1], "[")+1 : strings.Index(nameSet[1], "]")]
		variablesSet := strings.Split(variables, ",")
		Newfunction.argumentCount = len(variablesSet)
		Newfunction.argumentDict = variablesSet
		Newfunction.funcVariableDict = make(map[string]interface{})
		if Newfunction.argumentCount > 0 {
			Newfunction.argumentState = true
		}
		for v := range variablesSet {
			Newfunction.funcVariableDict[variablesSet[v]] = variablesSet[v]
		}
		functionDict[Newfunction.name] = Newfunction
		definitionName = Newfunction.name
		Newfunction.content = make([]string, 0)

	} else if definitionState == true {
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

	} else if strings.Contains(tok, "if") && strings.Contains(tok, "]") && strings.Contains(tok, "[") && conditionState == false {
		conditionState = true
		var ifElse ifelseCondition
		ifElse.head = tok
		conditionName = tok
		ifElse.content = append(ifElse.content, tok)
		ifElseDict[ifElse.head] = ifElse
	} else if conditionState {
		if strings.Contains(tok, "[end]") && strings.Contains(tok, "if") {
			ifelseCopy := ifElseDict[conditionName]
			ifelseCopy.content = append(ifElseDict[conditionName].content, tok)
			ifElseDict[conditionName] = ifelseCopy
			ifelse(ifElseDict[conditionName].content, state)
			conditionState = false
		} else {
			ifelseCopy := ifElseDict[conditionName]
			ifelseCopy.content = append(ifElseDict[conditionName].content, tok)
			ifElseDict[conditionName] = ifelseCopy

		}
	} else if strings.Contains(tok, "]") && strings.Contains(tok, "loop") && strings.Contains(tok, "[") && !strings.Contains(tok, "[end]") && loopState == false {
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

		if strings.Contains(tok, "list") && getVariable(strings.Split(tok, "=")[1]) == "list" {
			dataStructureProtocol("list", state, tok)
		} else if strings.Contains(tok, "map") && getVariable(strings.Split(tok, "=")[1]) == "map" {
			dataStructureProtocol("map", state, tok)
		} else if strings.Contains(tok, "set") && getVariable(strings.Split(tok, "=")[1]) == "set" {
			dataStructureProtocol("set", state, tok)
		} else {
			insertFunction(tok, state)
		}
	} else if strings.Contains(tok, "[") && strings.Contains(tok, "]") && strings.LastIndex(tok, "]") > strings.LastIndex(tok, ".") {
		functionProtocol(tok, state)
	} else if strings.Contains(tok, "show") {
		showTok := strings.SplitAfter(tok, "show")
		if strings.Contains(showTok[0], "show") && state == "isMain" {
			showReal(tok, state)
		} else {
			showRealFunc(tok, state)
		}
	} else if strings.Contains(tok, "?") && strings.Contains(tok, "=") {
		if strings.Index(tok, "?") < strings.Index(tok, "\"") {
			input := strings.Split(tok, "=")
			var variable string = ""
			fmt.Print(getPrompt(tok))
			scanIn := bufio.NewScanner(os.Stdin)
			scanIn.Scan()
			variable = scanIn.Text()
			vars := strings.ReplaceAll(input[0], " ", "")
			variableDict[vars] = variable
		}
	} else if strings.Contains(tok, "=") {
		varTok := strings.SplitAfter(tok, "=")
		if strings.Contains(varTok[0], "=") && state == "isMain" {
			insertVariable(tok, state)
		} else {
			insertVariableFunc(tok, state)
		}
	}

}

// Get phrase from prompt for input for variables
// add a prompt for inputing variables rather than using show
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

// works the same as the if else dictionary
// used to hold the loop structure
var loopDict = make(map[string]loop)

// develop loop structure and run it. works like function protocol
// it manages the loop variables, parses everything for execution
func loopStructure(loop []string, state string) {
	// formats the loop to a proper format then executes it
	loopConstruct := loop[0]
	loopConstruct = strings.ReplaceAll(loopConstruct, "[loop]", "")
	loopConstruct = strings.ReplaceAll(loopConstruct, "[", "")
	loopConstruct = strings.ReplaceAll(loopConstruct, "]", "")
	count := 0
	var expressionV int
	incrementer := ""
	operator := ""
	// get the loop structure first
	if strings.Contains(loopConstruct, ";") {
		loopParsed := strings.Split(loopConstruct, ";")
		for _, looptoken := range loopParsed {
			if strings.Contains(looptoken, "=") && !strings.Contains(looptoken, "<") && !strings.Contains(looptoken, ">") && !strings.Contains(looptoken, "!") {
				value := strings.Split(looptoken, "=")
				// logic need for functions
				if getVariable(value[1]) != "" {
					if state == "isMain" {
						counter, _ := strconv.Atoi(variableDict[getVariable(value[1])].(string))
						variableDict[getVariable(value[0])] = variableDict[getVariable(value[1])]
						count = counter
					} else {
						counter, _ := strconv.Atoi(functionDict[state].funcVariableDict[getVariable(value[1])].(string))
						functionDict[state].funcVariableDict[getVariable(value[0])] = functionDict[state].funcVariableDict[getVariable(value[1])]
						count = counter
					}

				} else {
					if state == "isMain" {
						counter, _ := strconv.Atoi(strings.ReplaceAll(value[1], " ", ""))
						variableDict[getVariable(value[0])] = eval(value[1])
						count = counter
					} else {
						counter, _ := strconv.Atoi(functionDict[state].funcVariableDict[getVariable(value[1])].(string))
						functionDict[state].funcVariableDict[getVariable(value[0])] = eval(value[1])
						count = counter
					}
				}

			} else if strings.Contains(looptoken, "<") && !strings.Contains(looptoken, "=") {
				operator = "<"
				value := strings.Split(looptoken, operator)
				if getVariable(value[1]) != "" {
					if state == "isMain" {
						expressionValue, _ := strconv.Atoi(variableDict[getVariable(value[1])].(string))
						variableDict[getVariable(value[0])] = variableDict[getVariable(value[1])]
						expressionV = expressionValue
					} else {
						expressionValue, _ := strconv.Atoi(functionDict[state].funcVariableDict[getVariable(value[1])].(string))
						functionDict[state].funcVariableDict[getVariable(value[0])] = functionDict[state].funcVariableDict[getVariable(value[1])]
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
						expressionValue, _ := strconv.Atoi(variableDict[getVariable(value[1])].(string))
						variableDict[getVariable(value[0])] = variableDict[getVariable(value[1])]
						expressionV = expressionValue
					} else {
						expressionValue, _ := strconv.Atoi(functionDict[state].funcVariableDict[getVariable(value[1])].(string))
						functionDict[state].funcVariableDict[getVariable(value[0])] = functionDict[state].funcVariableDict[getVariable(value[1])]
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
						expressionValue, _ := strconv.Atoi(variableDict[getVariable(value[1])].(string))
						variableDict[getVariable(value[0])] = variableDict[getVariable(value[1])]
						expressionV = expressionValue
					} else {
						expressionValue, _ := strconv.Atoi(functionDict[state].funcVariableDict[getVariable(value[1])].(string))
						functionDict[state].funcVariableDict[getVariable(value[0])] = functionDict[state].funcVariableDict[getVariable(value[1])]
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
						expressionValue, _ := strconv.Atoi(variableDict[getVariable(value[1])].(string))
						variableDict[getVariable(value[0])] = variableDict[getVariable(value[1])]
						expressionV = expressionValue
					} else {
						expressionValue, _ := strconv.Atoi(functionDict[state].funcVariableDict[getVariable(value[1])].(string))
						functionDict[state].funcVariableDict[getVariable(value[0])] = functionDict[state].funcVariableDict[getVariable(value[1])]
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
						expressionValue, _ := strconv.Atoi(variableDict[getVariable(value[1])].(string))
						variableDict[getVariable(value[0])] = variableDict[getVariable(value[1])]
						expressionV = expressionValue
					} else {
						expressionValue, _ := strconv.Atoi(functionDict[state].funcVariableDict[getVariable(value[1])].(string))
						functionDict[state].funcVariableDict[getVariable(value[0])] = functionDict[state].funcVariableDict[getVariable(value[1])]
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
		// while loop logic add here
		// add variable state for while loops
		// if either of the expression is a variable check the increment or deincrement option before calling code
		counterState := false
		expressionVState := false
		var counterValue string
		var expressionVarValue string
		if strings.Contains(loopConstruct, "<") {
			value := strings.Split(loopConstruct, "<")
			if getVariable(value[0]) != "" {
				if state == "isMain" {
					counter, _ := strconv.Atoi(variableDict[getVariable(value[0])].(string))
					count = counter
					counterState = true
					counterValue = value[0]
				} else {
					counter, _ := strconv.Atoi(functionDict[state].funcVariableDict[getVariable(value[0])].(string))
					count = counter
					counterState = true
					counterValue = value[0]
				}

			} else {
				// fuzzy logic here will have to fix later as i think on it
				counter, _ := strconv.Atoi(strings.ReplaceAll(value[1], " ", ""))
				count = counter
			}
			// fmt.Println(value[1], "===", getVariable(value[1]))
			if getVariable(value[1]) != "" {

				if state == "isMain" {
					expressionValue, _ := strconv.Atoi(variableDict[getVariable(value[1])].(string))
					expressionV = expressionValue
					expressionVState = true
					expressionVarValue = value[1]
				} else {

					expressionValue, _ := strconv.Atoi(functionDict[state].funcVariableDict[getVariable(value[1])].(string))
					expressionV = expressionValue
					expressionVState = true
					expressionVarValue = value[1]
				}

			} else {
				expressionValue, _ := strconv.Atoi(strings.ReplaceAll(value[1], " ", ""))
				expressionV = expressionValue
				// fmt.Println(expressionValue)
			}
			for count < expressionV {
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
						if counterState == true && expressionVState == true {
							if state == "isMain" {
								if strings.Contains(tok, "=") && strings.Contains(tok, "+") || strings.Contains(tok, "-") {
									countContext := getVariable(strings.Split(tok, "=")[0])
									if getVariable(countContext) == getVariable(counterValue) {
										callCode(tok, state)
										count, _ = strconv.Atoi(variableDict[getVariable(counterValue)].(string))
									}
									if getVariable(countContext) == getVariable(expressionVarValue) {
										callCode(tok, state)
										expressionV, _ = strconv.Atoi(variableDict[getVariable(expressionVarValue)].(string))
									}
								} else {

									callCode(tok, state)
								}

							} else {
								if strings.Contains(tok, "=") && strings.Contains(tok, "+") || strings.Contains(tok, "-") {
									countContext := getVariable(strings.Split(tok, "=")[0])
									if getVariable(countContext) == getVariable(counterValue) {
										callCode(tok, state)
										count, _ = strconv.Atoi(functionDict[state].funcVariableDict[getVariable(counterValue)].(string))
									}
									if getVariable(tok) == getVariable(expressionVarValue) {
										callCode(tok, state)
										expressionV, _ = strconv.Atoi(functionDict[state].funcVariableDict[getVariable(expressionVarValue)].(string))
									}
								} else {

									callCode(tok, state)
								}
							}

						} else if counterState == true && expressionVState == false {
							// fmt.Println(counterValue, expressionVarValue)
							if state == "isMain" {
								if strings.Contains(tok, "=") && strings.Contains(tok, "+") || strings.Contains(tok, "-") {
									countContext := getVariable(strings.Split(tok, "=")[0])
									if getVariable(countContext) == getVariable(counterValue) {
										callCode(tok, state)
										count, _ = strconv.Atoi(variableDict[getVariable(counterValue)].(string))
									}
								} else {

									callCode(tok, state)
								}

							} else {
								if strings.Contains(tok, "=") && strings.Contains(tok, "+") || strings.Contains(tok, "-") {
									countContext := getVariable(strings.Split(tok, "=")[0])
									if getVariable(countContext) == getVariable(counterValue) {
										callCode(tok, state)
										count, _ = strconv.Atoi(functionDict[state].funcVariableDict[getVariable(counterValue)].(string))
									}
								} else {
									callCode(tok, state)
								}
							}

						} else if counterState == false && expressionVState == true {
							if state == "isMain" {
								if strings.Contains(tok, "=") && strings.Contains(tok, "+") || strings.Contains(tok, "-") {
									countContext := getVariable(strings.Split(tok, "=")[0])

									if getVariable(countContext) == getVariable(expressionVarValue) {
										callCode(tok, state)
										expressionV, _ = strconv.Atoi(variableDict[getVariable(expressionVarValue)].(string))
									}
								} else {

									callCode(tok, state)
								}

							} else {
								if strings.Contains(tok, "=") && strings.Contains(tok, "+") || strings.Contains(tok, "-") {
									countContext := getVariable(strings.Split(tok, "=")[0])
									if getVariable(countContext) == getVariable(counterValue) {
										callCode(tok, state)
										count, _ = strconv.Atoi(functionDict[state].funcVariableDict[getVariable(counterValue)].(string))
									}
									if getVariable(tok) == getVariable(expressionVarValue) {
										callCode(tok, state)
										expressionV, _ = strconv.Atoi(functionDict[state].funcVariableDict[getVariable(expressionVarValue)].(string))
									}
								} else {

									callCode(tok, state)
								}
							}

						}
					}
				}
			}

		} else if strings.Contains(loopConstruct, ">") {
			value := strings.Split(loopConstruct, ">")
			if getVariable(value[0]) != "" {
				if state == "isMain" {
					counter, _ := strconv.Atoi(variableDict[getVariable(value[0])].(string))
					count = counter
					counterState = true
					counterValue = value[0]
				} else {
					counter, _ := strconv.Atoi(functionDict[state].funcVariableDict[getVariable(value[0])].(string))
					count = counter
					counterState = true
					counterValue = value[0]
				}

			} else {
				// fuzzy logic here will have to fix later as i think on it
				counter, _ := strconv.Atoi(strings.ReplaceAll(value[1], " ", ""))
				count = counter
			}
			// fmt.Println(value[1], "===", getVariable(value[1]))
			if getVariable(value[1]) != "" {

				if state == "isMain" {
					expressionValue, _ := strconv.Atoi(variableDict[getVariable(value[1])].(string))
					expressionV = expressionValue
					expressionVState = true
					expressionVarValue = value[1]
				} else {

					expressionValue, _ := strconv.Atoi(functionDict[state].funcVariableDict[getVariable(value[1])].(string))
					expressionV = expressionValue
					expressionVState = true
					expressionVarValue = value[1]
				}

			} else {
				expressionValue, _ := strconv.Atoi(strings.ReplaceAll(value[1], " ", ""))
				expressionV = expressionValue
				// fmt.Println(expressionValue)
			}
			// fmt.Println(count, expressionV, counterState, expressionVState)
			for count > expressionV {
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
						if counterState == true && expressionVState == true {
							if state == "isMain" {
								if strings.Contains(tok, "=") && strings.Contains(tok, "+") || strings.Contains(tok, "-") {
									countContext := getVariable(strings.Split(tok, "=")[0])
									if getVariable(countContext) == getVariable(counterValue) {
										callCode(tok, state)
										count, _ = strconv.Atoi(variableDict[getVariable(counterValue)].(string))
									}
									if getVariable(countContext) == getVariable(expressionVarValue) {
										callCode(tok, state)
										expressionV, _ = strconv.Atoi(variableDict[getVariable(expressionVarValue)].(string))
										// fmt.Println(expressionV)
									}
								} else {

									callCode(tok, state)
								}

							} else {
								if strings.Contains(tok, "=") && strings.Contains(tok, "+") || strings.Contains(tok, "-") {
									countContext := getVariable(strings.Split(tok, "=")[0])
									if getVariable(countContext) == getVariable(counterValue) {
										callCode(tok, state)
										count, _ = strconv.Atoi(functionDict[state].funcVariableDict[getVariable(counterValue)].(string))
									}
									if getVariable(countContext) == getVariable(expressionVarValue) {
										callCode(tok, state)
										expressionV, _ = strconv.Atoi(functionDict[state].funcVariableDict[getVariable(expressionVarValue)].(string))
									}
								} else {

									callCode(tok, state)
								}
							}

						} else if counterState == true && expressionVState == false {
							// fmt.Println(counterValue, expressionVarValue)
							if state == "isMain" {
								if strings.Contains(tok, "=") && strings.Contains(tok, "+") || strings.Contains(tok, "-") {
									countContext := getVariable(strings.Split(tok, "=")[0])
									if getVariable(countContext) == getVariable(counterValue) {
										callCode(tok, state)
										count, _ = strconv.Atoi(variableDict[getVariable(counterValue)].(string))
									}
								} else {

									callCode(tok, state)
								}

							} else {
								if strings.Contains(tok, "=") && strings.Contains(tok, "+") || strings.Contains(tok, "-") {
									countContext := getVariable(strings.Split(tok, "=")[0])
									if getVariable(countContext) == getVariable(counterValue) {
										callCode(tok, state)
										count, _ = strconv.Atoi(functionDict[state].funcVariableDict[getVariable(counterValue)].(string))
									}
								} else {
									callCode(tok, state)
								}
							}

						} else if counterState == false && expressionVState == true {
							if state == "isMain" {
								if strings.Contains(tok, "=") && strings.Contains(tok, "+") || strings.Contains(tok, "-") {
									countContext := getVariable(strings.Split(tok, "=")[0])

									if getVariable(countContext) == getVariable(expressionVarValue) {
										callCode(tok, state)
										expressionV, _ = strconv.Atoi(variableDict[getVariable(expressionVarValue)].(string))
									}
								} else {

									callCode(tok, state)
								}

							} else {
								if strings.Contains(tok, "=") && strings.Contains(tok, "+") || strings.Contains(tok, "-") {
									countContext := getVariable(strings.Split(tok, "=")[0])
									if getVariable(countContext) == getVariable(counterValue) {
										callCode(tok, state)
										count, _ = strconv.Atoi(functionDict[state].funcVariableDict[getVariable(counterValue)].(string))
									}
									if getVariable(countContext) == getVariable(expressionVarValue) {
										callCode(tok, state)
										expressionV, _ = strconv.Atoi(functionDict[state].funcVariableDict[getVariable(expressionVarValue)].(string))
									}
								} else {

									callCode(tok, state)
								}
							}

						}
					}
				}
			}

		} else if strings.Contains(loopConstruct, "<=") {
			value := strings.Split(loopConstruct, "<=")
			if getVariable(value[0]) != "" {
				if state == "isMain" {
					counter, _ := strconv.Atoi(variableDict[getVariable(value[0])].(string))
					count = counter
					counterState = true
					counterValue = value[0]
				} else {
					counter, _ := strconv.Atoi(functionDict[state].funcVariableDict[getVariable(value[0])].(string))
					count = counter
					counterState = true
					counterValue = value[0]
				}

			} else {
				// fuzzy logic here will have to fix later as i think on it
				counter, _ := strconv.Atoi(strings.ReplaceAll(value[1], " ", ""))
				count = counter
			}
			// fmt.Println(value[1], "===", getVariable(value[1]))
			if getVariable(value[1]) != "" {

				if state == "isMain" {
					expressionValue, _ := strconv.Atoi(variableDict[getVariable(value[1])].(string))
					expressionV = expressionValue
					expressionVState = true
					expressionVarValue = value[1]
				} else {

					expressionValue, _ := strconv.Atoi(functionDict[state].funcVariableDict[getVariable(value[1])].(string))
					expressionV = expressionValue
					expressionVState = true
					expressionVarValue = value[1]
				}

			} else {
				expressionValue, _ := strconv.Atoi(strings.ReplaceAll(value[1], " ", ""))
				expressionV = expressionValue
				// fmt.Println(expressionValue)
			}

			for count <= expressionV {
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
						if counterState == true && expressionVState == true {
							if state == "isMain" {
								if strings.Contains(tok, "=") && strings.Contains(tok, "+") || strings.Contains(tok, "-") {
									countContext := getVariable(strings.Split(tok, "=")[0])
									if getVariable(countContext) == getVariable(counterValue) {
										callCode(tok, state)
										count, _ = strconv.Atoi(variableDict[getVariable(counterValue)].(string))
									}
									if getVariable(countContext) == getVariable(expressionVarValue) {
										callCode(tok, state)
										expressionV, _ = strconv.Atoi(variableDict[getVariable(expressionVarValue)].(string))
									}
								} else {

									callCode(tok, state)
								}

							} else {
								if strings.Contains(tok, "=") && strings.Contains(tok, "+") || strings.Contains(tok, "-") {
									countContext := getVariable(strings.Split(tok, "=")[0])
									if getVariable(countContext) == getVariable(counterValue) {
										callCode(tok, state)
										count, _ = strconv.Atoi(functionDict[state].funcVariableDict[getVariable(counterValue)].(string))
									}
									if getVariable(tok) == getVariable(expressionVarValue) {
										callCode(tok, state)
										expressionV, _ = strconv.Atoi(functionDict[state].funcVariableDict[getVariable(expressionVarValue)].(string))
									}
								} else {

									callCode(tok, state)
								}
							}

						} else if counterState == true && expressionVState == false {
							// fmt.Println(counterValue, expressionVarValue)
							if state == "isMain" {
								if strings.Contains(tok, "=") && strings.Contains(tok, "+") || strings.Contains(tok, "-") {
									countContext := getVariable(strings.Split(tok, "=")[0])
									if getVariable(countContext) == getVariable(counterValue) {
										callCode(tok, state)
										count, _ = strconv.Atoi(variableDict[getVariable(counterValue)].(string))
									}
								} else {

									callCode(tok, state)
								}

							} else {
								if strings.Contains(tok, "=") && strings.Contains(tok, "+") || strings.Contains(tok, "-") {
									countContext := getVariable(strings.Split(tok, "=")[0])
									if getVariable(countContext) == getVariable(counterValue) {
										callCode(tok, state)
										count, _ = strconv.Atoi(functionDict[state].funcVariableDict[getVariable(counterValue)].(string))
									}
								} else {
									callCode(tok, state)
								}
							}

						} else if counterState == false && expressionVState == true {
							if state == "isMain" {
								if strings.Contains(tok, "=") && strings.Contains(tok, "+") || strings.Contains(tok, "-") {
									countContext := getVariable(strings.Split(tok, "=")[0])

									if getVariable(countContext) == getVariable(expressionVarValue) {
										callCode(tok, state)
										expressionV, _ = strconv.Atoi(variableDict[getVariable(expressionVarValue)].(string))
									}
								} else {

									callCode(tok, state)
								}

							} else {
								if strings.Contains(tok, "=") && strings.Contains(tok, "+") || strings.Contains(tok, "-") {
									countContext := getVariable(strings.Split(tok, "=")[0])
									if getVariable(countContext) == getVariable(counterValue) {
										callCode(tok, state)
										count, _ = strconv.Atoi(functionDict[state].funcVariableDict[getVariable(counterValue)].(string))
									}
									if getVariable(tok) == getVariable(expressionVarValue) {
										callCode(tok, state)
										expressionV, _ = strconv.Atoi(functionDict[state].funcVariableDict[getVariable(expressionVarValue)].(string))
									}
								} else {

									callCode(tok, state)
								}
							}

						}
					}
				}
			}

		} else if strings.Contains(loopConstruct, ">=") {
			value := strings.Split(loopConstruct, ">=")
			if getVariable(value[0]) != "" {
				if state == "isMain" {
					counter, _ := strconv.Atoi(variableDict[getVariable(value[0])].(string))
					count = counter
					counterState = true
					counterValue = value[0]
				} else {
					counter, _ := strconv.Atoi(functionDict[state].funcVariableDict[getVariable(value[0])].(string))
					count = counter
					counterState = true
					counterValue = value[0]
				}

			} else {
				// fuzzy logic here will have to fix later as i think on it
				counter, _ := strconv.Atoi(strings.ReplaceAll(value[1], " ", ""))
				count = counter
			}
			// fmt.Println(value[1], "===", getVariable(value[1]))
			if getVariable(value[1]) != "" {

				if state == "isMain" {
					expressionValue, _ := strconv.Atoi(variableDict[getVariable(value[1])].(string))
					expressionV = expressionValue
					expressionVState = true
					expressionVarValue = value[1]
				} else {

					expressionValue, _ := strconv.Atoi(functionDict[state].funcVariableDict[getVariable(value[1])].(string))
					expressionV = expressionValue
					expressionVState = true
					expressionVarValue = value[1]
				}

			} else {
				expressionValue, _ := strconv.Atoi(strings.ReplaceAll(value[1], " ", ""))
				expressionV = expressionValue
				// fmt.Println(expressionValue)
			}

			for count >= expressionV {
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
						if counterState == true && expressionVState == true {
							if state == "isMain" {
								if strings.Contains(tok, "=") && strings.Contains(tok, "+") {
									countContext := getVariable(strings.Split(tok, "=")[0])
									if getVariable(countContext) == getVariable(counterValue) {
										callCode(tok, state)
										count, _ = strconv.Atoi(variableDict[getVariable(counterValue)].(string))
									}
									if getVariable(countContext) == getVariable(expressionVarValue) {
										callCode(tok, state)
										expressionV, _ = strconv.Atoi(variableDict[getVariable(expressionVarValue)].(string))
									}
								} else {

									callCode(tok, state)
								}

							} else {
								if strings.Contains(tok, "=") && strings.Contains(tok, "+") {
									countContext := getVariable(strings.Split(tok, "=")[0])
									if getVariable(countContext) == getVariable(counterValue) {
										callCode(tok, state)
										count, _ = strconv.Atoi(functionDict[state].funcVariableDict[getVariable(counterValue)].(string))
									}
									if getVariable(tok) == getVariable(expressionVarValue) {
										callCode(tok, state)
										expressionV, _ = strconv.Atoi(functionDict[state].funcVariableDict[getVariable(expressionVarValue)].(string))
									}
								} else {

									callCode(tok, state)
								}
							}

						} else if counterState == true && expressionVState == false {
							// fmt.Println(counterValue, expressionVarValue)
							if state == "isMain" {
								if strings.Contains(tok, "=") && strings.Contains(tok, "+") {
									countContext := getVariable(strings.Split(tok, "=")[0])
									if getVariable(countContext) == getVariable(counterValue) {
										callCode(tok, state)
										count, _ = strconv.Atoi(variableDict[getVariable(counterValue)].(string))
									}
								} else {

									callCode(tok, state)
								}

							} else {
								if strings.Contains(tok, "=") && strings.Contains(tok, "+") {
									countContext := getVariable(strings.Split(tok, "=")[0])
									if getVariable(countContext) == getVariable(counterValue) {
										callCode(tok, state)
										count, _ = strconv.Atoi(functionDict[state].funcVariableDict[getVariable(counterValue)].(string))
									}
								} else {
									callCode(tok, state)
								}
							}

						} else if counterState == false && expressionVState == true {
							if state == "isMain" {
								if strings.Contains(tok, "=") && strings.Contains(tok, "+") {
									countContext := getVariable(strings.Split(tok, "=")[0])

									if getVariable(countContext) == getVariable(expressionVarValue) {
										callCode(tok, state)
										expressionV, _ = strconv.Atoi(variableDict[getVariable(expressionVarValue)].(string))
									}
								} else {

									callCode(tok, state)
								}

							} else {
								if strings.Contains(tok, "=") && strings.Contains(tok, "+") {
									countContext := getVariable(strings.Split(tok, "=")[0])
									if getVariable(countContext) == getVariable(counterValue) {
										callCode(tok, state)
										count, _ = strconv.Atoi(functionDict[state].funcVariableDict[getVariable(counterValue)].(string))
									}
									if getVariable(tok) == getVariable(expressionVarValue) {
										callCode(tok, state)
										expressionV, _ = strconv.Atoi(functionDict[state].funcVariableDict[getVariable(expressionVarValue)].(string))
									}
								} else {

									callCode(tok, state)
								}
							}

						}
					}
				}
			}

		} else if strings.Contains(loopConstruct, "!=") {
			value := strings.Split(loopConstruct, "!=")
			if getVariable(value[0]) != "" {
				if state == "isMain" {
					counter, _ := strconv.Atoi(variableDict[getVariable(value[0])].(string))
					count = counter
					counterState = true
					counterValue = value[0]
				} else {
					counter, _ := strconv.Atoi(functionDict[state].funcVariableDict[getVariable(value[0])].(string))
					count = counter
					counterState = true
					counterValue = value[0]
				}

			} else {
				// fuzzy logic here will have to fix later as i think on it
				counter, _ := strconv.Atoi(strings.ReplaceAll(value[1], " ", ""))
				count = counter
			}
			// fmt.Println(value[1], "===", getVariable(value[1]))
			if getVariable(value[1]) != "" {

				if state == "isMain" {
					expressionValue, _ := strconv.Atoi(variableDict[getVariable(value[1])].(string))
					expressionV = expressionValue
					expressionVState = true
					expressionVarValue = value[1]
				} else {

					expressionValue, _ := strconv.Atoi(functionDict[state].funcVariableDict[getVariable(value[1])].(string))
					expressionV = expressionValue
					expressionVState = true
					expressionVarValue = value[1]
				}

			} else {
				expressionValue, _ := strconv.Atoi(strings.ReplaceAll(value[1], " ", ""))
				expressionV = expressionValue
				// fmt.Println(expressionValue)
			}

			for count != expressionV {
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
						if counterState == true && expressionVState == true {
							if state == "isMain" {
								if strings.Contains(tok, "=") && strings.Contains(tok, "+") || strings.Contains(tok, "-") {
									countContext := getVariable(strings.Split(tok, "=")[0])
									if getVariable(countContext) == getVariable(counterValue) {
										callCode(tok, state)
										count, _ = strconv.Atoi(variableDict[getVariable(counterValue)].(string))
									}
									if getVariable(countContext) == getVariable(expressionVarValue) {
										callCode(tok, state)
										expressionV, _ = strconv.Atoi(variableDict[getVariable(expressionVarValue)].(string))
									}
								} else {

									callCode(tok, state)
								}

							} else {
								if strings.Contains(tok, "=") && strings.Contains(tok, "+") || strings.Contains(tok, "-") {
									countContext := getVariable(strings.Split(tok, "=")[0])
									if getVariable(countContext) == getVariable(counterValue) {
										callCode(tok, state)
										count, _ = strconv.Atoi(functionDict[state].funcVariableDict[getVariable(counterValue)].(string))
									}
									if getVariable(tok) == getVariable(expressionVarValue) {
										callCode(tok, state)
										expressionV, _ = strconv.Atoi(functionDict[state].funcVariableDict[getVariable(expressionVarValue)].(string))
									}
								} else {

									callCode(tok, state)
								}
							}

						} else if counterState == true && expressionVState == false {
							// fmt.Println(counterValue, expressionVarValue)
							if state == "isMain" {
								if strings.Contains(tok, "=") && strings.Contains(tok, "+") || strings.Contains(tok, "-") {
									countContext := getVariable(strings.Split(tok, "=")[0])
									if getVariable(countContext) == getVariable(counterValue) {
										callCode(tok, state)
										count, _ = strconv.Atoi(variableDict[getVariable(counterValue)].(string))
									}
								} else {

									callCode(tok, state)
								}

							} else {
								if strings.Contains(tok, "=") && strings.Contains(tok, "+") || strings.Contains(tok, "-") {
									countContext := getVariable(strings.Split(tok, "=")[0])
									if getVariable(countContext) == getVariable(counterValue) {
										callCode(tok, state)
										count, _ = strconv.Atoi(functionDict[state].funcVariableDict[getVariable(counterValue)].(string))
									}
								} else {
									callCode(tok, state)
								}
							}

						} else if counterState == false && expressionVState == true {
							if state == "isMain" {
								if strings.Contains(tok, "=") && strings.Contains(tok, "+") || strings.Contains(tok, "-") {
									countContext := getVariable(strings.Split(tok, "=")[0])

									if getVariable(countContext) == getVariable(expressionVarValue) {
										callCode(tok, state)
										expressionV, _ = strconv.Atoi(variableDict[getVariable(expressionVarValue)].(string))
									}
								} else {

									callCode(tok, state)
								}

							} else {
								if strings.Contains(tok, "=") && strings.Contains(tok, "+") || strings.Contains(tok, "-") {
									countContext := getVariable(strings.Split(tok, "=")[0])
									if getVariable(countContext) == getVariable(counterValue) {
										callCode(tok, state)
										count, _ = strconv.Atoi(functionDict[state].funcVariableDict[getVariable(counterValue)].(string))
									}
									if getVariable(tok) == getVariable(expressionVarValue) {
										callCode(tok, state)
										expressionV, _ = strconv.Atoi(functionDict[state].funcVariableDict[getVariable(expressionVarValue)].(string))
									}
								} else {

									callCode(tok, state)
								}
							}

						}
					}
				}
			}

		}
	}

	// cases for loop to run
	// further logic needed for loop variables i , and expression v
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

// Adds data to a list data struture , takes string data and does the proper mapping
func dataStructureListParser(strlist string) list {
	var newStructure list
	newlist := strlist[strings.Index(strlist, "[")+1 : strings.LastIndex(strlist, "]")]
	stringStatus := 0
	// fmt.Println(newlist)
	var element string
	// fmt.Println(newlist)
	for _, value := range newlist {
		// fmt.Println(string(value))
		if string(value) == "\"" {
			if stringStatus == 0 {
				stringStatus += 1
				element += string(value)
			} else {
				stringStatus = 0
				element += string(value)
				// newStructure.add(element)
				// element = ""
			}
		} else if string(value) == "," && stringStatus == 0 {
			newStructure.add(element)
			element = ""
		} else {
			element += string(value)
		}

	}

	if element != "" {
		newStructure.add(element)
		element = ""
	}

	return newStructure
}

// Adds data to a set data struture , takes string data and does the proper mapping
func dataStructureSetParser(list string) set {
	var newStructure set
	newlist := list[strings.Index(list, "[")+1 : strings.LastIndex(list, "]")]
	stringStatus := 0
	var element string
	// fmt.Println(newlist)
	for _, value := range newlist {
		// fmt.Println(string(value))
		if string(value) == "\"" {

			if stringStatus == 0 {
				stringStatus += 1
				element += string(value)
			} else {
				stringStatus = 0
				element += string(value)
				newStructure.add(element)
				element = ""
			}
		} else if string(value) == "," && stringStatus == 0 {
			newStructure.add(element)
			element = ""

		} else {
			element += string(value)
		}

	}

	if element != "" {
		newStructure.add(element)
		element = ""
	}

	return newStructure
}

func dataStructureMapsParser(list string) maps {
	var newStructure maps
	newStructure.maps = make(map[string]interface{})
	newlist := list[strings.Index(list, "[")+1 : strings.LastIndex(list, "]")]
	stringStatus := 0
	var element string
	// fmt.Println(newlist)
	for _, value := range newlist {
		// fmt.Println(string(value))
		if string(value) == "\"" {

			if stringStatus == 0 {
				stringStatus += 1
				element += string(value)
			} else {
				stringStatus = 0
				element += string(value)
			}
		} else if string(value) == "," && stringStatus == 0 {
			stringStat := 0
			for Index, value := range element {
				if string(value) == "\"" {
					if stringStat == 0 {
						stringStat += 1
						element += string(value)
					} else {
						stringStat = 0
						element += string(value)
					}
				} else if string(value) == ":" && stringStatus == 0 {
					// fmt.Println(element)
					key := element[0 : Index-1]
					value := element[Index+1 : len(element)-1]
					newStructure.add(parseString(key), parseString(value))
					element = ""
				}

			}

		} else {
			element += string(value)
		}

	}

	if element != "" {
		stringStat := 0
		for Index, value := range element {
			if string(value) == "\"" {
				if stringStat == 0 {
					stringStat += 1
					element += string(value)
				} else {
					stringStat = 0
					element += string(value)
				}
			} else if string(value) == ":" && stringStatus == 0 {
				key := element[0 : Index-1]
				value := element[Index+1 : len(element)-1]
				newStructure.add(parseString(key), parseString(value))
				element = ""
			}

		}
	}

	return newStructure

}

type maps struct {
	maps   map[string]interface{}
	length int
}

func (Amap *maps) delete(data string) {
	delete(Amap.maps, data)
}

func (maper *maps) add(key string, value string) {
	// fmt.Println(key, value, "in map")
	maper.maps[key] = value
	// fmt.Println(maper)
}

type set struct {
	set    []interface{}
	length int
}

type list struct {
	list   []interface{}
	length int
}

// Add value to list
func (Alist *list) add(data string) {
	Alist.list = append(Alist.list, data)
}

// get length of list
func (Alist *list) len() {
	Alist.length = len(Alist.list)
}

// Add value to set
func (Aset *set) add(data string) {
	var isThere bool
	for _, v := range Aset.set {
		if data == v {
			isThere = true
			break
		}
	}
	if isThere == false {
		Aset.set = append(Aset.set, data)
	}

}

// get length of set
func (Aset *set) len() {
	Aset.length = len(Aset.set)
}

// convert data structures to strings
// convert list of data into a string
func (Alist *list) toString() string {
	list := "[ "
	for _, v := range Alist.list {
		if data, errors := v.([]string); errors {
			list += "[ "
			for _, d := range data {
				list = list + d + " , "
			}
			list += " ]"
		} else {
			list = list + v.(string) + " , "
		}

	}
	list += " ]"
	return list
}

// convert set of data into a string
func (Aset *set) toString() string {
	set := "[ "
	for _, v := range Aset.set {
		if data, errors := v.([]string); errors {
			set += "[ "
			for _, d := range data {
				set = set + d + " , "
			}
			set += " ]"
		} else {
			set = set + v.(string) + " , "
		}

	}
	set += " ]"
	return set
}

// convert map data into a string
func (Amap maps) toString() string {
	maper := "[ "
	for key, value := range Amap.maps {
		maper = maper + key + ":" + value.(string) + ","
	}
	maper += " ]"
	return maper
}

// list functions
func (Alist *list) clear() {
	var newlist list
	Alist.list = newlist.list
	Alist.length = 0

}

// vars is the variablee copying to and lists is list copying from
func (Alist *list) copy(vars string, state string) {
	if state == "isMain" {
		variable := getVariable(vars)
		sorc := variableDict[variable].(list)
		copy(Alist.list, sorc.list)

	} else {
		variable := getVariable(vars)
		sorc := functionDict[state].funcVariableDict[variable].(list)
		copy(Alist.list, sorc.list)
	}

}

func (Alist *list) count(value interface{}) int {
	count := 0
	for _, v := range Alist.list {
		if v == value {
			count += 1
		}
	}
	return count
}

func (Alist *list) index(value interface{}) int {
	count := 0
	for _, v := range Alist.list {
		if v == value {
			count += 1
			break
		}
	}
	return count

}

// For sorting values
func (a *list) Len() int      { return len(a.list) }
func (a *list) Swap(i, j int) { a.list[i], a.list[j] = a.list[j], a.list[i] }

// Less implements sort.Interface
func (Alist *list) Less(i int, j int) bool {
	switch v1 := Alist.list[i].(type) {
	case string:
		return strings.Compare(v1, Alist.list[j].(string)) < 0 // import "strings"
		// return v1 < t2.(string)
	case int:
		return v1 < Alist.list[j].(int)
	case float32:
		return v1 < Alist.list[j].(float32)
	case float64:
		return v1 < Alist.list[j].(float64)

	}

	return false
}

// Not sure if this will even work
func (Alist *list) sort(value interface{}) {

	sort.Sort(Alist)

}

func (Alist *list) remove(value interface{}) {
	index := Alist.index(value)
	copy(Alist.list[index:], Alist.list[index+1:])
	Alist.list = Alist.list[:len(Alist.list)-1]

}

// Reverse function for reversing data
func (Alist *list) reverse() {
	for i, j := 0, len(Alist.list); i < j; i, j = i+1, j-1 {
		Alist.list[i], Alist.list[j] = Alist.list[j], Alist.list[i]
	}

}

// Pop the last index out of list
func (Alist *list) pop() {
	if len(Alist.list) > 0 {
		Alist.list = Alist.list[:len(Alist.list)-1]
	}
}

// find value in list
func (Alist *list) find(value interface{}) bool {
	found := false
	for _, v := range Alist.list {
		if v == value {
			found = true
		}
	}
	return found
}

// Set functions
func (Aset *set) clear() {
	var newSet set
	Aset.set = newSet.set
	Aset.length = 0

}

func (Aset *set) copy(vars string, state string) {
	if state == "isMain" {
		variable := getVariable(vars)
		sorc := variableDict[variable].(set)
		copy(Aset.set, sorc.set)

	} else {
		variable := getVariable(vars)
		sorc := functionDict[state].funcVariableDict[variable].(set)
		copy(Aset.set, sorc.set)
	}

}

func (Aset *set) remove() {

}

// For sorting values
func (a *set) Len() int      { return len(a.set) }
func (a *set) Swap(i, j int) { a.set[i], a.set[j] = a.set[j], a.set[i] }

// Less implements sort.Interface
func (Aset *set) Less(i int, j int) bool {
	switch v1 := Aset.set[i].(type) {
	case string:
		return strings.Compare(v1, Aset.set[j].(string)) < 0 // import "strings"
		// return v1 < t2.(string)
	case int:
		return v1 < Aset.set[j].(int)
	case float32:
		return v1 < Aset.set[j].(float32)
	case float64:
		return v1 < Aset.set[j].(float64)

	}

	return false
}

func (Aset *set) sort() {
	sort.Sort(Aset)

}

func (Aset *set) reverse() {

}

func (Aset *set) pop() {
	if len(Aset.set) > 0 {
		Aset.set = Aset.set[:len(Aset.set)-1]
	}

}

func (Aset *set) union() set {
	return *Aset
}

func (Aset *set) intersection() set {
	return *Aset
}

func (Aset *set) difference() set {
	return *Aset
}

func (Aset *set) subset() bool {
	return true
}

func (Aset *set) superset() bool {
	return true
}

// max function gets the max value out of a list, set, or max key out of a map
func max(data string, state string) interface{} {

	var max interface{}
	if state == "isMain" {
		variable := getVariable(data)
		if data, errors := variableDict[variable].(set); errors {
			max = data.set[0]
			for i := 1; i < len(data.set); i++ {
				if max.(string) < data.set[i].(string) {
					max = data.set[i]
				} else if max.(int) < data.set[i].(int) {
					max = data.set[i]
				} else if max.(float64) < data.set[i].(float64) {
					max = data.set[i]
				} else if max.(float32) < data.set[i].(float32) {
					max = data.set[i]
				} else if len(max.(set).set) < len(data.set[i].(set).set) {
					max = data.set[i]
				} else if len(max.(list).list) < len(data.set[i].(list).list) {
					max = data.set[i]
				}
			}

		} else if data, errors := variableDict[variable].(list); errors {
			max = data.list[0]
			for i := 1; i < len(data.list); i++ {
				if max.(string) < data.list[i].(string) {
					max = data.list[i]
				} else if max.(int) < data.list[i].(int) {
					max = data.list[i]
				} else if max.(float64) < data.list[i].(float64) {
					max = data.list[i]
				} else if max.(float32) < data.list[i].(float32) {
					max = data.list[i]
				} else if len(max.(list).list) < len(data.list[i].(list).list) {
					max = data.list[i]
				} else if len(max.(set).set) < len(data.list[i].(set).set) {
					max = data.list[i]
				}
			}

		} else if data, errors := variableDict[variable].(maps); errors {
			keys := make([]string, 0, len(data.maps))
			for k := range data.maps {
				keys = append(keys, k)
			}

			max = keys[0]
			for i := 1; i < len(keys); i++ {
				if max.(string) < keys[i] {
					max = keys[i]
				}
			}

		}
	} else {
		variable := getVariable(data)
		if data, errors := functionDict[state].funcVariableDict[variable].(set); errors {
			max = data.set[0]
			for i := 1; i < len(data.set); i++ {
				if max.(string) < data.set[i].(string) {
					max = data.set[i]
				} else if max.(int) < data.set[i].(int) {
					max = data.set[i]
				} else if max.(float64) < data.set[i].(float64) {
					max = data.set[i]
				} else if max.(float32) < data.set[i].(float32) {
					max = data.set[i]
				} else if len(max.(set).set) < len(data.set[i].(set).set) {
					max = data.set[i]
				} else if len(max.(list).list) < len(data.set[i].(list).list) {
					max = data.set[i]
				}
			}

		} else if data, errors := functionDict[state].funcVariableDict[variable].(list); errors {
			max = data.list[0]
			for i := 1; i < len(data.list); i++ {
				if max.(string) < data.list[i].(string) {
					max = data.list[i]
				} else if max.(int) < data.list[i].(int) {
					max = data.list[i]
				} else if max.(float64) < data.list[i].(float64) {
					max = data.list[i]
				} else if max.(float32) < data.list[i].(float32) {
					max = data.list[i]
				} else if len(max.(list).list) < len(data.list[i].(list).list) {
					max = data.list[i]
				} else if len(max.(set).set) < len(data.list[i].(set).set) {
					max = data.list[i]
				}
			}

		} else if data, errors := functionDict[state].funcVariableDict[variable].(maps); errors {
			keys := make([]string, 0, len(data.maps))
			for k := range data.maps {
				keys = append(keys, k)
			}

			max = keys[0]
			for i := 1; i < len(keys); i++ {
				if max.(string) < keys[i] {
					max = keys[i]
				}
			}
		}

	}

	return max
}

// get max Value out of a map , max will only get the max key out of a map
func maxValue(data string, state string) interface{} {
	var max interface{}
	if state == "isMain" {
		variable := getVariable(data)
		if data, errors := variableDict[variable].(maps); errors {
			keys := make([]string, 0, len(data.maps))
			for k := range data.maps {
				keys = append(keys, k)
			}

			max = keys[0]
			for i := 1; i < len(keys); i++ {
				if data.maps[max.(string)].(string) < data.maps[keys[i]].(string) {
					max = data.maps[keys[i]]
				} else if data.maps[max.(string)].(int) < data.maps[keys[i]].(int) {
					max = data.maps[keys[i]]
				} else if data.maps[max.(string)].(float32) < data.maps[keys[i]].(float32) {
					max = data.maps[keys[i]]
				} else if data.maps[max.(string)].(float64) < data.maps[keys[i]].(float64) {
					max = data.maps[keys[i]]
				} else if len(data.maps[max.(string)].(set).set) < len(data.maps[keys[i]].(set).set) {
					max = data.maps[keys[i]]
				} else if len(data.maps[max.(string)].(list).list) < len(data.maps[keys[i]].(list).list) {
					max = data.maps[keys[i]]
				}
			}

		}
	} else {
		variable := getVariable(data)
		if data, errors := functionDict[state].funcVariableDict[variable].(maps); errors {
			keys := make([]string, 0, len(data.maps))
			for k := range data.maps {
				keys = append(keys, k)
			}

			max = keys[0]
			for i := 1; i < len(keys); i++ {
				if data.maps[max.(string)].(string) < data.maps[keys[i]].(string) {
					max = data.maps[keys[i]]
				} else if data.maps[max.(string)].(int) < data.maps[keys[i]].(int) {
					max = data.maps[keys[i]]
				} else if data.maps[max.(string)].(float32) < data.maps[keys[i]].(float32) {
					max = data.maps[keys[i]]
				} else if data.maps[max.(string)].(float64) < data.maps[keys[i]].(float64) {
					max = data.maps[keys[i]]
				} else if len(data.maps[max.(string)].(set).set) < len(data.maps[keys[i]].(set).set) {
					max = data.maps[keys[i]]
				} else if len(data.maps[max.(string)].(list).list) < len(data.maps[keys[i]].(list).list) {
					max = data.maps[keys[i]]
				}
			}

		}

	}
	// add map application for max value
	return max
}

// data Structure Protocol used to intialize and setup data structures
func dataStructureProtocol(isType string, state string, tok string) {

	if isType == "list" {
		if state == "isMain" {
			// Parse list Structure
			list := strings.Split(tok, "=")[1]
			list = strings.Replace(list, "list", "", 1)
			variableDict[getVariable(strings.Split(tok, "=")[0])] = dataStructureListParser(list)
		} else {
			// parese list structure
			list := strings.Split(tok, "=")[1]
			list = strings.Replace(list, "list", "", 1)
			functionDict[state].funcVariableDict[getVariable(strings.Split(tok, "=")[0])] = dataStructureListParser(list)

		}
	} else if isType == "set" {
		if state == "isMain" {
			// Parse Set Structure
			set := strings.Split(tok, "=")[1]
			set = strings.Replace(set, "set", "", 1)
			variableDict[getVariable(strings.Split(tok, "=")[0])] = dataStructureSetParser(set)
			//fmt.Println(variableDict[getVariable(strings.Split(tok, "=")[0])]., "here")
		} else {
			// Parse Set Structure
			set := strings.Split(tok, "=")[1]
			set = strings.Replace(set, "set", "", 1)
			functionDict[state].funcVariableDict[getVariable(strings.Split(tok, "=")[0])] = dataStructureSetParser(set)
		}

	} else if isType == "map" {
		if state == "isMain" {
			mapss := strings.Split(tok, "=")[1]
			mapss = strings.Replace(mapss, "map", "", 1)
			variableDict[getVariable(strings.Split(tok, "=")[0])] = dataStructureMapsParser(mapss)

		} else {
			mapss := strings.Split(tok, "=")[1]
			mapss = strings.Replace(mapss, "map", "", 1)
			functionDict[state].funcVariableDict[getVariable(strings.Split(tok, "=")[0])] = dataStructureMapsParser(mapss)
		}

	}

}

func FindKeyword(tok string, keyword []string) bool {
	status := false
	for _, v := range keyword {
		if tok == v {
			status = true
			break
		}
	}
	return status
}

// data Structure operations function used to handle the various data structure
// operations , manipulation and etc.
func dataStructureOperations(state string, tok string) {
	//
	keywords := []string{"if", "sort", "in", "remove", "to", "is", "from", "add", "equal", "not", "of", "max",
		"min", "length", "delete", "sort", "reverse", "insert", "at"}

	if state == "isMain" {
		token := strings.Split(tok, " ")
		if FindKeyword(token[0], keywords) == true {
			fmt.Println(token, tok)

		} else {

		}

	} else {
		token := strings.Split(tok, " ")
		if FindKeyword(token[0], keywords) == true {

		} else {

		}

	}

}

// Evaluating data structures with other data types
// --------------------------------------------------------------------------------
func evalDataExpressions(str string, state string) string {

	inString := 0
	newShow := str
	newString := ""
	// check if the string has any commas outside string or if it's a concat
	var isConcat bool
	var oneStatement bool
	oneStatement = isOneVariable(str)
	isConcat = isConcatExp(str)
	if isConcat == true {
		parseToken := str
		place := 0
		placeAfter := 0
		for count := 0; count <= strings.LastIndex(str, "."); count++ {
			if string(str[count]) == "\"" {
				inString += 1
				if inString > 1 {
					inString = 0
				}
				continue
			} else if string(str[count]) == plus && inString == 0 {
				if placeAfter == 0 {
					if strings.Contains(str, "[") && strings.Contains(str, "]") && strings.Contains(str[placeAfter:count-1], "[") && strings.Contains(str[placeAfter:count-1], "]") {
						funcshowState = true
						functionProtocol(str[0:count-1], "isMain")
						parseToken = strings.ReplaceAll(parseToken, string(str[0:count-1]), funcShowReturn)
						funcshowState = false
						funcShowReturn = ""
					} else {
						variable := getVariable(parseToken[place:count])
						if variable == "" {
							place = count + 1
						} else {
							if Aset, errors := variableDict[variable].(set); errors {
								parseToken = strings.ReplaceAll(parseToken, variable, Aset.toString())
								place = count + 1
							} else if Alist, errors := variableDict[variable].(list); errors {
								parseToken = strings.ReplaceAll(parseToken, variable, Alist.toString())
								place = count + 1
							} else if Amap, errors := variableDict[variable].(maps); errors {
								parseToken = strings.ReplaceAll(parseToken, variable, Amap.toString())
								place = count + 1
							} else {
								parseToken = strings.ReplaceAll(parseToken, variable, variableDict[variable].(string))
								place = count + 1
							}

						}
					}
					placeAfter = count + 1

				} else {
					if strings.Contains(parseToken, "[") && strings.Contains(parseToken, "]") && strings.Contains(str[placeAfter:count-1], "[") && strings.Contains(str[placeAfter:count-1], "]") {
						funcshowState = true
						functionProtocol(str[placeAfter:count-1], "isMain")
						parseToken = strings.ReplaceAll(parseToken, string(str[placeAfter:count-1]), funcShowReturn)
						funcshowState = false
						funcShowReturn = ""
					} else {
						variable := getVariable(str[placeAfter : count-1])
						if variable == "" {
							place = count + 1
						} else {

							if Aset, errors := variableDict[variable].(set); errors {
								parseToken = strings.ReplaceAll(parseToken, variable, Aset.toString())
								place = count + 1
							} else if Alist, errors := variableDict[variable].(list); errors {
								parseToken = strings.ReplaceAll(parseToken, variable, Alist.toString())
								place = count + 1
							} else if Amap, errors := variableDict[variable].(maps); errors {
								parseToken = strings.ReplaceAll(parseToken, variable, Amap.toString())
								place = count + 1
							} else {
								parseToken = strings.ReplaceAll(parseToken, variable, variableDict[variable].(string))
								place = count + 1
							}
						}
					}
					placeAfter = count + 1

				}

			} else if count == strings.LastIndex(str, ".") {
				if strings.Contains(parseToken, "[") && strings.Contains(parseToken, "]") && strings.Contains(str[placeAfter:count-1], "[") && strings.Contains(str[placeAfter:count-1], "]") {
					funcshowState = true
					functionProtocol(str[placeAfter:count-1], "isMain")
					parseToken = strings.ReplaceAll(parseToken, string(str[placeAfter:count-1]), funcShowReturn)
					funcshowState = false
					funcShowReturn = ""

				} else {
					variable := getVariable(str[placeAfter:count])
					if variable == "" {
						place = count + 1
					} else {
						if Aset, errors := variableDict[variable].(set); errors {
							parseToken = strings.ReplaceAll(parseToken, variable, Aset.toString())
							place = count + 1
						} else if Alist, errors := variableDict[variable].(list); errors {
							parseToken = strings.ReplaceAll(parseToken, variable, Alist.toString())
							place = count + 1
						} else if Amap, errors := variableDict[variable].(maps); errors {
							parseToken = strings.ReplaceAll(parseToken, variable, Amap.toString())
							place = count + 1
						} else {
							parseToken = strings.ReplaceAll(parseToken, variable, variableDict[variable].(string))
							place = count + 1
						}
					}

				}
			}
		}
		parseToken = parseToken[0:strings.LastIndex(parseToken, ".")]
		return parseString(strings.ReplaceAll(eval(parseToken), "\\n", "\n"))
	} else if oneStatement == true {
		// fmt.Println(oneStatement)
		// fmt.Println(str)
		// fmt.Println(str[0:strings.LastIndex(str, ".")])
		// fmt.Println(getVariable(str[0:strings.LastIndex(str, ".")]))
		variable := getVariable(str[0:strings.LastIndex(str, ".")])
		if !strings.Contains(str, ",") {
			if strings.Contains(str, "[") && strings.Contains(str, "]") {
				funcshowState = true
				functionProtocol(str, "isMain")
				funcshowState = false
				return funcShowReturn
			} else {
				if Aset, errors := variableDict[variable].(set); errors {
					return Aset.toString()
				} else if Alist, errors := variableDict[variable].(list); errors {
					return Alist.toString()
				} else if Amap, errors := variableDict[variable].(maps); errors {
					return Amap.toString()
				} else {
					return variableDict[variable].(string)
				}

			}
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
					if evalType(newShow[place:count], "isMain") == "Var" {
						oneVar := isOneVariable(newShow[place:count])
						if oneVar == true {
							if strings.Contains(newShow[place:count], "[") && strings.Contains(newShow[place:count], "]") {
								funcshowState = true
								functionProtocol(newShow[place:count], "isMain")
								newString += funcShowReturn

							} else {
								variable := getVariable(newShow[place:count])
								if Aset, errors := variableDict[variable].(set); errors {
									arg = Aset.toString()
									newString += parseString(arg)
								} else if Alist, errors := variableDict[variable].(list); errors {
									arg = Alist.toString()
									newString += parseString(arg)
								} else if Amap, errors := variableDict[variable].(maps); errors {
									arg = Amap.toString()
									newString += parseString(arg)
								} else {
									arg = variableDict[variable].(string)
									newString += parseString(arg)
								}
							}

						} else {
							newString += getevalVar(newShow[place:count])
						}

					} else {
						if evalType(newShow[place:count], "isMain") == "Exp" {
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
					if evalType(newShow[place:count], "isMain") == "Var" {
						oneVar := isOneVariable(newShow[place:count])
						if oneVar == true {
							if strings.Contains(newShow[place:count], "[") && strings.Contains(newShow[place:count], "]") {
								funcshowState = true
								functionProtocol(newShow[place:count], "isMain")
								newString += funcShowReturn
							} else {
								variable := getVariable(newShow[place:count])
								if Aset, errors := variableDict[variable].(set); errors {
									arg = Aset.toString()
									newString += parseString(arg)
								} else if Alist, errors := variableDict[variable].(list); errors {
									arg = Alist.toString()
									newString += parseString(arg)
								} else if Amap, errors := variableDict[variable].(maps); errors {
									arg = Amap.toString()
									newString += parseString(arg)
								} else {
									arg = variableDict[variable].(string)
									newString += parseString(arg)
								}

							}
						} else {
							newString += getevalVarPeriod(newShow[place:count])
						}
					} else {
						if evalType(newShow[place:count], "isMain") == "Exp" {
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
				if evalType(newShow[place:count], "isMain") == "Var" {
					oneVar := isOneVariable(newShow[place:count])
					if oneVar == true {
						variable := getVariable(newShow[place:count])
						if Aset, errors := variableDict[variable].(set); errors {
							arg = Aset.toString()
							newString += parseString(arg)
						} else if Alist, errors := variableDict[variable].(list); errors {
							arg = Alist.toString()
							newString += parseString(arg)
						} else if Amap, errors := variableDict[variable].(maps); errors {
							arg = Amap.toString()
							newString += parseString(arg)
						} else {
							arg = variableDict[variable].(string)
							newString += parseString(arg)
						}

					} else {
						newString += getevalVar(newShow[place:count])
					}

				} else {
					if evalType(newShow[place:count], "isMain") == "Exp" {
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
				if evalType(newShow[place:count], "isMain") == "Var" {
					oneVar := isOneVariable(newShow[place:count])
					if oneVar == true {
						variable := getVariable(newShow[place:count])
						if Aset, errors := variableDict[variable].(set); errors {
							arg = Aset.toString()
							newString += parseString(arg)
						} else if Alist, errors := variableDict[variable].(list); errors {
							arg = Alist.toString()
							newString += parseString(arg)
						} else if Amap, errors := variableDict[variable].(maps); errors {
							arg = Amap.toString()
							newString += parseString(arg)
						} else {
							arg = variableDict[variable].(string)
							newString += parseString(arg)
						}

					} else {
						newString += getevalVarPeriod(newShow[place:count])
					}
				} else {
					if evalType(newShow[place:count], "isMain") == "Exp" {
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

func evalDataExpressionFunc(str string, name string) string {
	inString := 0
	newShow := str
	newString := ""
	// check if the string has any commas outside string or if it's a concat
	var isConcat bool
	var oneStatement bool
	oneStatement = isOneVariable(str)
	isConcat = isConcatExp(str)
	if isConcat == true {
		parseToken := str
		place := 0
		placeAfter := 0
		for count := 0; count <= strings.LastIndex(str, "."); count++ {
			if string(str[count]) == "\"" {
				inString += 1
				if inString > 1 {
					inString = 0
				}
				continue
			} else if string(str[count]) == plus && inString == 0 {
				if placeAfter == 0 {
					if strings.Contains(parseToken, "[") && strings.Contains(parseToken, "]") && strings.Contains(str[placeAfter:count-1], "[") && strings.Contains(str[placeAfter:count-1], "]") {
						funcshowState = true
						functionProtocol(str[0:count-1], name)
						parseToken = strings.ReplaceAll(parseToken, string(str[0:count-1]), funcShowReturn)
						funcshowState = false
						funcShowReturn = ""
					} else {
						variable := getVariable(parseToken[place:count])
						if variable == "" {
							place = count + 1
						} else {
							if Aset, errors := functionDict[name].funcVariableDict[variable].(set); errors {
								parseToken = strings.ReplaceAll(parseToken, variable, Aset.toString())
								place = count + 1
							} else if Alist, errors := functionDict[name].funcVariableDict[variable].(list); errors {
								parseToken = strings.ReplaceAll(parseToken, variable, Alist.toString())
								place = count + 1
							} else if Amap, errors := functionDict[name].funcVariableDict[variable].(maps); errors {
								parseToken = strings.ReplaceAll(parseToken, variable, Amap.toString())
								place = count + 1
							} else {
								parseToken = strings.ReplaceAll(parseToken, variable, functionDict[name].funcVariableDict[variable].(string))
								place = count + 1
							}
						}

					}
					placeAfter = count + 1

				} else {
					if strings.Contains(parseToken, "[") && strings.Contains(parseToken, "]") && strings.Contains(str[placeAfter:count-1], "[") && strings.Contains(str[placeAfter:count-1], "]") {
						funcshowState = true
						functionProtocol(str[placeAfter:count-1], name)
						parseToken = strings.ReplaceAll(parseToken, string(str[placeAfter:count-1]), funcShowReturn)
						funcshowState = false
						funcShowReturn = ""
					} else {
						variable := getVariable(str[placeAfter : count-1])
						if variable == "" {
							place = count + 1
						} else {

							if Aset, errors := functionDict[name].funcVariableDict[variable].(set); errors {
								parseToken = strings.ReplaceAll(parseToken, variable, Aset.toString())
								place = count + 1
							} else if Alist, errors := functionDict[name].funcVariableDict[variable].(list); errors {
								parseToken = strings.ReplaceAll(parseToken, variable, Alist.toString())
								place = count + 1
							} else if Amap, errors := functionDict[name].funcVariableDict[variable].(maps); errors {
								parseToken = strings.ReplaceAll(parseToken, variable, Amap.toString())
								place = count + 1
							} else {
								parseToken = strings.ReplaceAll(parseToken, variable, functionDict[name].funcVariableDict[variable].(string))
								place = count + 1
							}
						}
					}
					placeAfter = count + 1

				}
			} else if count == strings.LastIndex(str, ".") {
				if placeAfter == 0 {
					if strings.Contains(str, "[") && strings.Contains(str, "]") && strings.Contains(str[placeAfter:count-1], "[") && strings.Contains(str[placeAfter:count-1], "]") {
						funcshowState = true
						functionProtocol(str[0:count-1], name)
						parseToken = strings.ReplaceAll(parseToken, string(str[0:count-1]), funcShowReturn)
						funcshowState = false
						funcShowReturn = ""
					} else {
						variable := getVariable(parseToken[place:count])
						if variable == "" {
							place = count + 1
						} else {
							if Aset, errors := functionDict[name].funcVariableDict[variable].(set); errors {
								parseToken = strings.ReplaceAll(parseToken, variable, Aset.toString())
								place = count + 1
							} else if Alist, errors := functionDict[name].funcVariableDict[variable].(list); errors {
								parseToken = strings.ReplaceAll(parseToken, variable, Alist.toString())
								place = count + 1
							} else if Amap, errors := functionDict[name].funcVariableDict[variable].(maps); errors {
								parseToken = strings.ReplaceAll(parseToken, variable, Amap.toString())
								place = count + 1
							} else {
								parseToken = strings.ReplaceAll(parseToken, variable, functionDict[name].funcVariableDict[variable].(string))
								place = count + 1
							}
						}
					}
					placeAfter = count + 1

				} else {
					if strings.Contains(parseToken, "[") && strings.Contains(parseToken, "]") && strings.Contains(str[placeAfter:count-1], "[") && strings.Contains(str[placeAfter:count-1], "]") {
						funcshowState = true
						functionProtocol(str[placeAfter:count-1], name)
						parseToken = strings.ReplaceAll(parseToken, string(str[placeAfter:count-1]), funcShowReturn)
						funcshowState = false
						funcShowReturn = ""
					} else {
						variable := getVariable(str[placeAfter : count-1])
						if variable == "" {
							place = count + 1
						} else {
							if Aset, errors := functionDict[name].funcVariableDict[variable].(set); errors {
								parseToken = strings.ReplaceAll(parseToken, variable, Aset.toString())
								place = count + 1
							} else if Alist, errors := functionDict[name].funcVariableDict[variable].(list); errors {
								parseToken = strings.ReplaceAll(parseToken, variable, Alist.toString())
								place = count + 1
							} else if Amap, errors := functionDict[name].funcVariableDict[variable].(maps); errors {
								parseToken = strings.ReplaceAll(parseToken, variable, Amap.toString())
								place = count + 1
							} else {
								parseToken = strings.ReplaceAll(parseToken, variable, functionDict[name].funcVariableDict[variable].(string))
								place = count + 1
							}
						}
					}
					placeAfter = count + 1

				}

			}
		}

		parseToken = parseToken[0:strings.LastIndex(parseToken, ".")]

		return parseString(strings.ReplaceAll(eval(parseToken), "\\n", "\n"))
	} else if oneStatement == true {
		variable := getVariable(str[0:strings.LastIndex(str, ".")])
		if !strings.Contains(str, ",") {
			if strings.Contains(str, "[") && strings.Contains(str, "]") {
				funcshowState = true
				functionProtocol(str, name)
				funcshowState = false
				return funcShowReturn
			} else {
				if Aset, errors := functionDict[name].funcVariableDict[variable].(set); errors {
					return Aset.toString()
				} else if Alist, errors := functionDict[name].funcVariableDict[variable].(list); errors {
					return Alist.toString()
				} else if Amap, errors := functionDict[name].funcVariableDict[variable].(maps); errors {
					return Amap.toString()
				} else {
					return functionDict[name].funcVariableDict[variable].(string)
				}
			}

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
					if evalType(newShow[place:count], name) == "Var" {
						oneVar := isOneVariable(newShow[place:count])
						if oneVar == true {
							if strings.Contains(newShow[place:count], "[") && strings.Contains(newShow[place:count], "]") {
								funcshowState = true
								functionProtocol(newShow[place:count], name)
								newString += funcShowReturn

							} else {
								variable := getVariable(newShow[place:count])
								if Aset, errors := functionDict[name].funcVariableDict[variable].(set); errors {
									arg = Aset.toString()
									newString += parseString(arg)
								} else if Alist, errors := functionDict[name].funcVariableDict[variable].(list); errors {
									arg = Alist.toString()
									newString += parseString(arg)
								} else if Amap, errors := functionDict[name].funcVariableDict[variable].(maps); errors {
									arg = Amap.toString()
									newString += parseString(arg)
								} else {
									arg = functionDict[name].funcVariableDict[variable].(string)
									newString += parseString(arg)
								}
							}

						} else {
							newString += getevalVarFunc(newShow[place:count], name)
						}

					} else {
						if evalType(newShow[place:count], name) == "Exp" {
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
					if evalType(newShow[place:count], name) == "Var" {
						oneVar := isOneVariable(newShow[place:count])
						if oneVar == true {
							if strings.Contains(newShow[place:count], "[") && strings.Contains(newShow[place:count], "]") {
								funcshowState = true
								functionProtocol(newShow[place:count], name)
								newString += funcShowReturn

							} else {
								variable := getVariable(newShow[place:count])
								if Aset, errors := functionDict[name].funcVariableDict[variable].(set); errors {
									arg = Aset.toString()
									newString += parseString(arg)
								} else if Alist, errors := functionDict[name].funcVariableDict[variable].(list); errors {
									arg = Alist.toString()
									newString += parseString(arg)
								} else if Amap, errors := functionDict[name].funcVariableDict[variable].(maps); errors {
									arg = Amap.toString()
									newString += parseString(arg)
								} else {
									arg = functionDict[name].funcVariableDict[variable].(string)
									newString += parseString(arg)
								}
							}

						} else {
							newString += getevalVarPeriodFunc(newShow[place:count], name)
						}
					} else {
						if evalType(newShow[place:count], name) == "Exp" {
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
				if evalType(newShow[place:count], name) == "Var" {
					oneVar := isOneVariable(newShow[place:count])
					if oneVar == true {
						if strings.Contains(newShow[place:count], "[") && strings.Contains(newShow[place:count], "]") {
							funcshowState = true
							functionProtocol(newShow[place:count], name)
							newString += funcShowReturn

						} else {
							variable := getVariable(newShow[place:count])
							if Aset, errors := functionDict[name].funcVariableDict[variable].(set); errors {
								arg = Aset.toString()
								newString += parseString(arg)
							} else if Alist, errors := functionDict[name].funcVariableDict[variable].(list); errors {
								arg = Alist.toString()
								newString += parseString(arg)
							} else if Amap, errors := functionDict[name].funcVariableDict[variable].(maps); errors {
								arg = Amap.toString()
								newString += parseString(arg)
							} else {
								arg = functionDict[name].funcVariableDict[variable].(string)
								newString += parseString(arg)
							}
						}
					} else {
						newString += getevalVarFunc(newShow[place:count], name)
					}

				} else {
					if evalType(newShow[place:count], name) == "Exp" {
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
				if evalType(newShow[place:count], name) == "Var" {
					oneVar := isOneVariable(newShow[place:count])
					if oneVar == true {
						if strings.Contains(newShow[place:count], "[") && strings.Contains(newShow[place:count], "]") {
							funcshowState = true
							functionProtocol(newShow[place:count], name)
							newString += funcShowReturn

						} else {
							variable := getVariable(newShow[place:count])
							if Aset, errors := functionDict[name].funcVariableDict[variable].(set); errors {
								arg = Aset.toString()
								newString += parseString(arg)
							} else if Alist, errors := functionDict[name].funcVariableDict[variable].(list); errors {
								arg = Alist.toString()
								newString += parseString(arg)
							} else if Amap, errors := functionDict[name].funcVariableDict[variable].(maps); errors {
								arg = Amap.toString()
								newString += parseString(arg)
							} else {
								arg = functionDict[name].funcVariableDict[variable].(string)
								newString += parseString(arg)
							}
						}
					} else {
						newString += getevalVarPeriodFunc(newShow[place:count], name)
					}
				} else {
					if evalType(newShow[place:count], name) == "Exp" {
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

// Function is used if no argument is passed into the interperter
// It runs the most recent modified program in the /Users/* directory
// Further updates may follow in the future.
func OpenLatestFile() string {
	path, err := os.Getwd()
	check(err)
	fileToRead := make(map[string]int64)
	var file string
	smallT, err := filepath.Glob("*.t")
	bigT, err := filepath.Glob("*.T")
	wholeList := make([]string, 0)
	wholeList = append(wholeList, bigT...)
	wholeList = append(wholeList, smallT...)
	if len(wholeList) == 1 {
		return path + "/" + wholeList[0]
	} else if len(wholeList) == 0 {
		return ""
	} else {
		var seconds int64
		var newestTime int64 = 0
		for _, item := range wholeList {
			var st syscall.Stat_t
			fileStat := syscall.Stat(item, &st)
			check(fileStat)
			// fmt.Println(item, fileStat)
			fileToRead[item] = st.Ctimespec.Sec
			// fmt.Println(item, fileToRead[item], st.Ctimespec.Sec)
			seconds = st.Ctimespec.Sec
			if seconds > newestTime {
				newestTime = seconds
				file = item
			}

		}

	}

	return path + "/" + file

}

// Main function
func main() {
	var Lastfile string
	var LastfileState bool
	var file *os.File
	var err error
	if len(os.Args) == 2 {
		file, err = os.Open(os.Args[1])
	} else {
		Lastfile = OpenLatestFile()
		if Lastfile == "" {
			fmt.Println("No file to run")
			time.Sleep(10 * time.Second)
			os.Exit(3)
		} else {
			LastfileState = true
		}
	}

	definitionState := false
	definitionName := ""
	conditionState := false
	conditionName := ""
	loopState := false
	loopName := ""
	var scanner *bufio.Scanner
	if LastfileState == true {
		if strings.Contains(Lastfile, ".t") || strings.Contains(Lastfile, ".T") {
			file, err = os.Open(Lastfile)
			check(err)
			scanner = bufio.NewScanner(file)
		} else {
			fmt.Println("No file found")
			time.Sleep(2 * time.Second)
			os.Exit(1)
		}

	} else {
		scanner = bufio.NewScanner(file)
	}
	defer file.Close()
	for scanner.Scan() {

		tok := scanner.Text()
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
			variables := nameSet[1][strings.Index(nameSet[1], "[")+1 : strings.Index(nameSet[1], "]")]
			variablesSet := strings.Split(variables, ",")
			Newfunction.argumentCount = len(variablesSet)
			Newfunction.argumentDict = variablesSet
			Newfunction.funcVariableDict = make(map[string]interface{})
			if Newfunction.argumentCount > 0 {
				Newfunction.argumentState = true
			}
			for v := range variablesSet {
				Newfunction.funcVariableDict[variablesSet[v]] = variablesSet[v]
			}
			functionDict[Newfunction.name] = Newfunction
			definitionName = Newfunction.name
			Newfunction.content = make([]string, 0)
			continue

		} else if definitionState == true {
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
			} else {
				ifelseCopy := ifElseDict[conditionName]
				ifelseCopy.content = append(ifElseDict[conditionName].content, tok)
				ifElseDict[conditionName] = ifelseCopy

			}
			continue
		} else if strings.Contains(tok, "]") && strings.Contains(tok, "loop") && strings.Contains(tok, "[") && !strings.Contains(tok, "[end]") && loopState == false {
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

			if strings.Contains(tok, "list") && getVariable(strings.Split(tok, "=")[1]) == "list" {
				dataStructureProtocol("list", "isMain", tok)
			} else if strings.Contains(tok, "map") && getVariable(strings.Split(tok, "=")[1]) == "map" {
				dataStructureProtocol("map", "isMain", tok)
			} else if strings.Contains(tok, "set") && getVariable(strings.Split(tok, "=")[1]) == "set" {
				dataStructureProtocol("set", "isMain", tok)
			} else {
				insertFunction(tok, "isMain")
			}
		} else if strings.Contains(tok, "[") && strings.Contains(tok, "]") && strings.LastIndex(tok, "]") > strings.LastIndex(tok, ".") {
			functionProtocol(tok, "isMain")
		} else if strings.Contains(tok, "show") {
			showTok := strings.SplitAfter(tok, "show")
			if strings.Contains(showTok[0], "show") {
				showReal(tok, "isMain")
			}
		} else if strings.Contains(tok, "?") && strings.Contains(tok, "=") {
			if strings.Index(tok, "?") < strings.Index(tok, "\"") {
				input := strings.Split(tok, "=")
				var variable string = ""
				fmt.Print(getPrompt(tok))
				scanIn := bufio.NewScanner(os.Stdin)
				scanIn.Scan()
				variable = scanIn.Text()
				vars := strings.ReplaceAll(input[0], " ", "")
				variableDict[vars] = variable
			}
		} else if strings.Contains(tok, "=") {
			varTok := strings.SplitAfter(tok, "=")
			if strings.Contains(varTok[0], "=") {
				insertVariable(tok, "isMain")
			}
		} else {
			dataStructureOperations("isMain", tok)
		}

	}
}
