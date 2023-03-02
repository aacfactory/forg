package module_test

import (
	"fmt"
	"github.com/aacfactory/forg/module"
	"testing"
)

func TestParseAnnotations(t *testing.T) {
	s := `get
@fn get
@validate
@authorization
@permission
@internal
@title Get
abcd
@description 

>>>
Get a user @a
@b
''>>>''
''<<<''
----------
errors:
| Name                     | Code    | Description                   |
|--------------------------|---------|-------------------------------|
| users_get_failed         | 500     | get user failed               |
| users_get_nothing        | 404     | user was not found            |
<<<
1234
`

	annotations, parseErr := module.ParseAnnotations(s)
	if parseErr != nil {
		t.Error(parseErr)
		return
	}
	for key, value := range annotations {
		fmt.Println(key, "=", value)
	}
}
