package processes

type Result struct {
	No      int
	Name    string
	Succeed bool
	Result  interface{}
	Error   error
}
