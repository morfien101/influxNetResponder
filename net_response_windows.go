package main

import "flag"

var (
	configFile = flag.String("c", "c:\\Program Files\\telegraf\\plugins\\net_reponse.conf", "Path to the TOML configuration file.")
)
