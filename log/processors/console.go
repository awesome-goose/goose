package processors

import (
	"fmt"
)

type Console struct{}

func NewConsole() *Console {
	return &Console{}
}

func (p *Console) Process(record []byte) {
	fmt.Println(string(record))
}
