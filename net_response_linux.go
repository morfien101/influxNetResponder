package main

import "flag"

var (
	configFile = flag.String("c", "/etc/telegraf/plugins/net_reponse.conf", "Path to the TOML configuration file.")
)
