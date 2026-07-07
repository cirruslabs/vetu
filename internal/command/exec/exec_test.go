package exec

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestExecCommandFlags(t *testing.T) {
	// Reset package-level globals
	interactive = false
	tty = false

	cmd := NewCommand()

	// Interspersed flag parsing should be disabled.
	// Exec command's own flags (e.g. -i, -t) must be parsed.
	// But flags belonging to the remote command (e.g. -l, -c) must NOT be parsed.
	args := []string{"-i", "my-vm", "ls", "-l"}

	var parsedArgs []string
	cmd.RunE = func(c *cobra.Command, args []string) error {
		parsedArgs = args
		return nil
	}

	cmd.SetArgs(args)
	err := cmd.Execute()
	require.NoError(t, err)

	require.True(t, interactive, "expected -i flag to be parsed")
	require.False(t, tty, "expected -t flag to be false")

	// Positional args should include the remote command and its flags
	require.Equal(t, []string{"my-vm", "ls", "-l"}, parsedArgs)
}

func TestExecCommandFlagsNoTerminator(t *testing.T) {
	interactive = false
	tty = false

	cmd := NewCommand()

	args := []string{"my-vm", "sh", "-c", "echo ok"}

	var parsedArgs []string
	cmd.RunE = func(c *cobra.Command, args []string) error {
		parsedArgs = args
		return nil
	}

	cmd.SetArgs(args)
	err := cmd.Execute()
	require.NoError(t, err)

	require.False(t, interactive)
	require.False(t, tty)

	require.Equal(t, []string{"my-vm", "sh", "-c", "echo ok"}, parsedArgs)
}

func TestExecCommandFlagsWithTerminator(t *testing.T) {
	interactive = false
	tty = false

	cmd := NewCommand()

	// With explicit -- terminator
	args := []string{"my-vm", "--", "sh", "-c", "echo ok"}

	// We can test the actual parsing logic by calling the runExec function.
	// We'll pass a dummy cmd, but runExec will validate that the args are correctly shifted.
	// Since we don't have a running VM directory in this unit test, runExec will eventually
	// fail with "no VM my-vm" or similar local.Open error, but we can verify the shift
	// happens before that, or we can mock/stub or just check the error.
	// Actually, let's write a simple helper test for the shift logic if we want, or we can
	// test the command execution. Since runExec will return an error because the VM doesn't exist,
	// let's check what error it returns. It should return an error saying "VM 'my-vm' is not running"
	// or similar, showing that it parsed the name correctly.
	// More precisely, let's look at what runExec does:
	// name := args[0]
	// commandName := args[1] // will be "--" which gets stripped
	// localName, err := localname.NewFromString(name) -> succeeds for "my-vm"
	// local.Open(localName) -> fails because there is no VM "my-vm" in the local storage,
	// returning an os.ErrNotExist.
	// Let's assert that!
	cmd.SetArgs(args)
	err := cmd.Execute()
	require.Error(t, err)
	require.Contains(t, err.Error(), "is not running")
}
