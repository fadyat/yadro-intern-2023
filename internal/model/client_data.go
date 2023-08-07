package model

import (
	"fmt"
	"yadro-intern/internal/apierror"
)

type ClientData interface {
	GetName() string
	Validate() error

	fmt.Stringer
}

type ClientArrives struct {
	name string
}

func NewClientArrives(name string) *ClientArrives {
	return &ClientArrives{name: name}
}

func (c *ClientArrives) GetName() string {
	return c.name
}

func (c *ClientArrives) String() string {
	return c.name
}

func (c *ClientArrives) Validate() error {
	return apierror.ValidateName(c.name)
}

type ClientSits struct {
	name      string
	table     int
	maxTables int
}

func NewClientSits(name string, table, maxTables int) *ClientSits {
	return &ClientSits{name: name, table: table, maxTables: maxTables}
}

func (c *ClientSits) GetName() string {
	return c.name
}

func (c *ClientSits) GetTable() int {
	return c.table
}

func (c *ClientSits) String() string {
	return fmt.Sprintf("%s %d", c.name, c.table)
}

func (c *ClientSits) Validate() error {
	if err := apierror.ValidateName(c.name); err != nil {
		return err
	}

	if err := apierror.MoreThenZero(c.table); err != nil {
		return err
	}

	if err := apierror.NotMoreThen(c.table, c.maxTables); err != nil {
		return err
	}

	return nil
}

type ClientWaits struct {
	name string
}

func NewClientWaits(name string) *ClientWaits {
	return &ClientWaits{name: name}
}

func (c *ClientWaits) GetName() string {
	return c.name
}

func (c *ClientWaits) String() string {
	return c.name
}

func (c *ClientWaits) Validate() error {
	return apierror.ValidateName(c.name)
}

type ClientLeaves struct {
	name string
}

func NewClientLeaves(name string) *ClientLeaves {
	return &ClientLeaves{name: name}
}

func (c *ClientLeaves) GetName() string {
	return c.name
}

func (c *ClientLeaves) String() string {
	return c.name
}

func (c *ClientLeaves) Validate() error {
	return apierror.ValidateName(c.name)
}
