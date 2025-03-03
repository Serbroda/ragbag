package migrations

import "embed"

var (
	//go:embed common/*.sql
	MigrationsCommon    embed.FS
	MigrationsCommonDir = "common"
)
