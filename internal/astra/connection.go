package astra

import (
	"fmt"
	"strings"
)

type Connection struct {
	ID        string
	Name      string
	Login     string
	Password  string
	Interface string
	Port      int
	Debug     bool
}

func (c Connection) Addr() string {
	return fmt.Sprintf("%s:%d", c.Interface, c.Port)
}

func (c Connection) DSN() string {
	return fmt.Sprintf("%s:%s@%s:%d", c.Login, c.Password, c.Interface, c.Port)
}

func (c Connection) DisplayDSN() string {
	return "<" + c.DSN() + ">"
}

func (c Connection) MaskedDSN() string {
	return fmt.Sprintf(
		"%s:%s@%s:%d",
		c.Login,
		maskString(c.Password),
		c.Interface,
		c.Port,
	)
}

func (c Connection) DisplayMaskedDSN() string {
	return "<" + c.MaskedDSN() + ">"
}

func maskString(value string) string {
	return strings.Repeat("*", len([]rune(value)))
}
