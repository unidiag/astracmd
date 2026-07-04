package astra

import (
	"fmt"
	"strings"
)

type AstraConnection struct {
	ID        string
	Name      string
	Login     string
	Password  string
	Interface string
	Port      int
	Debug     bool
}

func (c AstraConnection) Addr() string {
	return fmt.Sprintf("%s:%d", c.Interface, c.Port)
}

func (c AstraConnection) DSN() string {
	return fmt.Sprintf("%s:%s@%s:%d", c.Login, c.Password, c.Interface, c.Port)
}

func (c AstraConnection) DisplayDSN() string {
	return "<" + c.DSN() + ">"
}

func (c AstraConnection) MaskedDSN() string {
	return fmt.Sprintf(
		"%s:%s@%s:%d",
		c.Login,
		maskString(c.Password),
		c.Interface,
		c.Port,
	)
}

func (c AstraConnection) DisplayMaskedDSN() string {
	return "<" + c.MaskedDSN() + ">"
}

func maskString(value string) string {
	return strings.Repeat("*", len([]rune(value)))
}
