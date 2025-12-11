package assets

import (
	_ "embed"
)

//go:embed entrypoint.sh
var EntrypointScript string
