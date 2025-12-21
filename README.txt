Provides functionality for parsing and editing .env files

Usage:

import (
	"log/slog"
	"os"

	"github.com/4nd3r5on/go-envfile"
	"github.com/4nd3r5on/go-envfile/common"
)

func main() {
	envfile.UpdateFile(
		"./.env",
		[]common.Update{
			{
				Key:   "SOME_SECRET",
				Value: "abc daf ghi",
			},
			{
				Key:   "KURWA",
				Value: "bober",
			},
            {
				Key:   "NUM",
				Value: "5",
			},
		},
		envfile.UpdateFileOptions{
			Backup: true,
		},
	)
}

Will preserve original comments, "export" keywords, etc

Have support for sections but not tested yet