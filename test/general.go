package test

import (
	"github.com/bowenislandsong/winc_node_setup/pkg/config"
	"os"
)

func main() {
	cred_path := os.Args[1]
	config.Credential_config(cred_path, "","")
}
