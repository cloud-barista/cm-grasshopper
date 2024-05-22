package ssh

func (c Client) RunBash(cmd string) (string, error) {
	out, err := c.Run("bash -c '" + cmd + "'")
	if err != nil {
		return "", err
	}

	return string(out), nil
}
