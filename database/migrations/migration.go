package migrations

type Migration interface {
	Up()
	Down()
}
