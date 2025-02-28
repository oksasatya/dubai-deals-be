package config

import (
	"github.com/veritrans/go-midtrans"
	"os"
)

type MidtransConfig struct {
	ClientKey string
	ServerKey string
	Env       *midtrans.EnvironmentType
	Snap      *midtrans.SnapGateway
	Core      *midtrans.CoreGateway
}

func InitMidtrans() *MidtransConfig {
	serverKey := os.Getenv("MIDTRANS_SERVER_KEY")
	clientKey := os.Getenv("MIDTRANS_CLIENT_KEY")
	envMode := os.Getenv("MIDTRANS_ENV")

	var env midtrans.EnvironmentType
	if envMode == "sandbox" {
		env = midtrans.Sandbox
	} else {
		env = midtrans.Production
	}

	// midtrans client
	midClient := midtrans.Client{}
	midClient.ServerKey = serverKey
	midClient.ClientKey = clientKey
	midClient.APIEnvType = env

	snapGateway := &midtrans.SnapGateway{
		Client: midClient,
	}

	coreGateway := &midtrans.CoreGateway{
		Client: midClient,
	}

	return &MidtransConfig{
		ClientKey: clientKey,
		ServerKey: serverKey,
		Env:       &env,
		Snap:      snapGateway,
		Core:      coreGateway,
	}
}
