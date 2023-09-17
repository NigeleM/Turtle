package main

import (
	"fmt"
	"reflect"
	"time"
)

// Reflection logic that can be used to handle certain complex data structure operations
// also can be used when custom data types are developed
func busy() {

	var a set
	a.add("1")
	a.len()
	fmt.Println(a)
	time.Sleep(2 * time.Second)
	v := reflect.ValueOf(&a).MethodByName("Len").Call(nil)
	fmt.Print(v[0])
	time.Sleep(1000 * time.Second)

}
