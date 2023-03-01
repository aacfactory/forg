package module

type Require struct {
	Dir      string
	Name     string
	Version  string
	Replace  *Require
	Indirect bool
	Module   *Module
}
