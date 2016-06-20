package hypervisor

import "github.com/gfrey/smutje/connection"

func runCommand(client connection.Client, cmd string) error {
	sess, err := client.NewSession()
	if err != nil {
		return err
	}
	defer sess.Close()

	if err := sess.Start(cmd); err != nil {
		return err
	}
	return sess.Wait()
}
