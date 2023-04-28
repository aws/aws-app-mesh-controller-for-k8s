package utils

import (
	"github.com/onsi/ginkgo/v2"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// NewGinkgoLogger returns new zap logger with ginkgo/v2 backend.
func NewGinkgoLogger() *zap.Logger {
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		zapcore.AddSync(ginkgo.GinkgoWriter),
		zap.InfoLevel,
	)
	return zap.New(core)
}
