package interfaces

type Seeder interface {
	// Run seeds the database, reporting the first failure to the caller
	Run() error
}
