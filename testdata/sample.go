package sample

import "fmt"

type Animal struct {
	Name string
}

func (a *Animal) Speak() string {
	return ""
}

type Dog struct {
	Animal
}

func (d *Dog) Speak() string {
	return fmt.Sprintf("%s says Woof!", d.Name)
}

func Greet(a *Animal) string {
	return a.Speak()
}
