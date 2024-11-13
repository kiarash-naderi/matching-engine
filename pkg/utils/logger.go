package utils

import (
	"github.com/sirupsen/logrus"
	"os"
)

var Logger = logrus.New()

func init() {
	// Logger settings
	Logger.SetOutput(os.Stdout)
	Logger.SetFormatter(&logrus.JSONFormatter{})
	Logger.SetLevel(logrus.InfoLevel)
}

// LogMatchResult logs matching results
func LogMatchResult(orderId string, result string) {
	Logger.WithFields(logrus.Fields{
		"order_id": orderId,
		"result":   result,
	}).Info("Order matching result")
}

// LogError logs errors
func LogError(err error) {
	Logger.WithFields(logrus.Fields{
		"error": err.Error(),
	}).Error("Error occurred")
}