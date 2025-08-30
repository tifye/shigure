package main

func V1Config() SimulatorConfig {
	return SimulatorConfig{
		userSimulator: userSimulatorConfig{
			UserConnectProbability:            80,
			UserDisconnectProbability:         20,
			InvalidDisconnectFaultProbability: 30,
		},
	}
}
