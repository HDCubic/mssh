package reg

import "testing"

func TestHelp(t *testing.T) {
	Help("")
	Help("help")
	Help("test")
}
