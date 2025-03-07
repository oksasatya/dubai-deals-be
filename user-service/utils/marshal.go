package utils

import (
	"encoding/json"
	"github.com/sirupsen/logrus"
)

func MarshalPayload(payload any) []byte {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		logrus.Errorf("Error marshalling payload: %v", err)
		return nil
	}
	logrus.Infof("Event data being sent: %s", string(payloadBytes))
	return payloadBytes
}
