package failure

import (
	"fmt"
	"os"
)

func ExitOnError(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
