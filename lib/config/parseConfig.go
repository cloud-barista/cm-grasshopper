package config

func PrepareConfigs() error {
	err := prepareCMGrasshopperConfig()
	if err != nil {
		return err
	}

	return nil
}
