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

	[write] file.txt
		content or variable
	[end]

	[append] file.txt
		content
	[end]
*/
func Filefunction(fileToken string) string {

	var token string
	// Read the data and if it is not on one line loop
	if strings.Contains(fileToken, "[read]") {

		if strings.Contains(fileToken, "[end]") {

			tok := strings.ReplaceAll(fileToken, "[read] ", "")
			tok = strings.ReplaceAll(tok, " [end]", " ")
			newtok := strings.Split(tok, " ")

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

						fmt.Println(string(fileRead))
						strings.Split(string(fileRead), "\n")
						token = "b = [" + strings.Join(strings.Split(string(fileRead), "\n"), ",") + "]"
						fmt.Println(token)
					}
					break
				}

			}

		} else {

		}

	} else if strings.Contains(fileToken, "[write]") {
		if strings.Contains(fileToken, "[end]") {

		} else {

		}

	} else if strings.Contains(fileToken, "[append]") {
		if strings.Contains(fileToken, "[end]") {

		} else {

		}

	}
	return token
}
