package interfaces

type Migration interface {
	Up()
	Down()
}
