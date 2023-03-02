package module_test

import (
	"fmt"
	"os"
	"testing"
)

func TestServices(t *testing.T) {
	entries, readErr := os.ReadDir(`D:\studio\workspace\go\src\github.com\go-zoo\bone\example`)
	if readErr != nil {
		t.Error(readErr)
		return
	}
	for _, entry := range entries {
		fmt.Println(entry.IsDir(), entry.Name())
	}
}
