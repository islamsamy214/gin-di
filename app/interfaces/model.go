package interfaces

type Model interface {
	Create() error
	Paginate(limit, page int) ([]Model, error)
	Find() error
	Update() error
	Delete() error
}
