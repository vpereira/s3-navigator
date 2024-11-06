package main

type ConnectionInformation struct {
	Name                  string `yaml:"name"`
	Endpoint              string `yaml:"endpoint"`
	AccessKey             string `yaml:"access_key"`
	SecretKey             string `yaml:"secret_key"`
	IgnoreSSLVerification bool   `yaml:"ignore_ssl_verification"`
}
