// File handling for turtle Library
package files

import (
	"fmt"
	"os"
	"strings"
)

/*
file read

	[read] file.txt to b [end]

	[write] file.txt [end]

	[write] file.txt a [end]


	// future enhancement
	[write] file.txt
		content or variable
	[end]

	[write] file.txt
		" hello, my name is nigele "
	[end]

	[write] file.txt
		" hello, my name is nigele.
		 the day is gone and I'm tired
		"
	[end]

	// write variables
	[write] file.txt
		a b c
		// can also add delimeters betweem variables
		or a, b, c
	[end]

	// append follws the same structure
	[append] file.txt
		content or variable
	[end]
*/
func Fileread(fileToken string) string {

	var token string
	// Read the data and if it is not on one line loop
	if !strings.Contains(fileToken, "[end]") {
		panic("[end] is required to execute statement")
	}

	// if key word to isn't in token then panic
	if !strings.Contains(fileToken, " to ") {
		panic("Must contain the word to")
	}

	tok := strings.ReplaceAll(fileToken, "[read] ", "")
	tok = strings.ReplaceAll(tok, " [end]", " ")
	newtok := strings.Split(tok, " ")

	// parse out variable from statement
	variable_to_hold_list := strings.Trim(fileToken[strings.Index(fileToken, " to ")+3:strings.Index(fileToken, "[end]")], " ")
	fmt.Println(variable_to_hold_list)
	if variable_to_hold_list == "" || variable_to_hold_list == " " {
		panic("missing variable from expression")
	}

	fmt.Println("newtok: ", newtok)
	// add the variable logic
	for _, value := range newtok {
		if value == " " || value == "" {
			continue

		} else {
			fileRead, err := os.ReadFile(value)
			if err != nil {
				fmt.Println("Error")
				panic(err)
			} else {

				// future enhancement is if the file only has one line
				// assign that line to a varialbe ... think through implementation
				strings.Split(string(fileRead), "\n")
				token = variable_to_hold_list + " = [" + strings.Join(strings.Split(string(fileRead), "\n"), ",") + "]"

			}
			break
		}

	}
	return token
}

func Filewrite(file_header string, fileToken interface{}) {

	fmt.Println("file header : ", file_header)
	fmt.Println("token : ", fileToken)

}

func Fileappend(file_header string, fileToken string) {

	fmt.Println("file header : ", file_header)
	fmt.Println("token : ", fileToken)

}
