// Пакет migrations вбудовує SQL-міграції бекенду у бінарник.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
