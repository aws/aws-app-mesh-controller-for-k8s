package cloudmap

import (
	"github.com/spf13/pflag"
)

const (
	flagSetCloudMapTTL = "cloudmap-dns-ttl"
)

type Config struct {
	//Specifies the DNS TTL value to be used while creating CloudMap services.
	CloudMapServiceTTL int64
}

func (cfg *Config) BindFlags(fs *pflag.FlagSet) {
	fs.Int64Var(&cfg.CloudMapServiceTTL, flagSetCloudMapTTL, defaultServiceDNSConfigTTL,
		`CloudMap Service DNS TTL value`)
}

func (cfg *Config) BindEnv() error {
	return nil
}

func (cfg *Config) Validate() error {
	return nil
}
