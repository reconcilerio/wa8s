package components

import (
	"fmt"
	"testing"
)

func TestExtractWit(t *testing.T) {
	ctx := t.Context()

	imports, exports, err := ExtractWIT(ctx, witToolsWasm)
	if err != nil {
		t.Error(err)
	}
	fmt.Println("IMPORTS:")
	for _, i := range imports {
		fmt.Println(i)
	}
	fmt.Println("EXPORTS:")
	for _, e := range exports {
		fmt.Println(e)
	}
}
