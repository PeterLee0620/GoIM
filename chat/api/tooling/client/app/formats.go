package app

import "fmt"

func formatMessage(name string, msg string) string {
	return fmt.Sprintf("%s: %s", name, msg)
}
