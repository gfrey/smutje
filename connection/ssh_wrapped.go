package connection

import "github.com/gfrey/smutje/logger"

type wrappedSSHClient struct {
	Client
	wrapper string
}

func NewWrappedClient(client Client, wrapper string) Client {
	return &wrappedSSHClient{Client: client, wrapper: wrapper}
}

func (wc *wrappedSSHClient) NewSession() (Session, error) {
	s, err := wc.Client.NewSession()
	if err != nil {
		return nil, err
	}

	return &wrappedSSHSession{Session: s, wrapper: wc.wrapper}, nil
}

func (wc *wrappedSSHClient) NewLoggedSession(l logger.Logger) (Session, error) {
	s, err := wc.Client.NewLoggedSession(l)
	if err != nil {
		return nil, err
	}

	return &wrappedSSHSession{Session: s, wrapper: wc.wrapper}, nil
}

type wrappedSSHSession struct {
	Session
	wrapper string
}

func (ws *wrappedSSHSession) Start(cmd string) error {
	return ws.Session.Start(ws.wrapper + " " + cmd)
}

func (ws *wrappedSSHSession) Run(cmd string) error {
	if err := ws.Start(cmd); err != nil {
		return err
	}
	return ws.Wait()
}
