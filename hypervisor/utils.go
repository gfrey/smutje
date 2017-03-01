package hypervisor

import "github.com/gfrey/gconn"

func runCommand(client gconn.Client, cmd string, args ...string) error {
	sess, err := client.NewSession(cmd, args...)
	if err != nil {
		return err
	}
	defer sess.Close()

	if err := sess.Start(); err != nil {
		return err
	}
	return sess.Wait()
}
