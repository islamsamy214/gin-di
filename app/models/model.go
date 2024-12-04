package models

type Model interface {
	Create(args ...interface{}) error
	Get(args ...interface{}) error
	Paginate(args ...interface{}) error
	Find(args ...interface{}) error
	Update(args ...interface{}) error
	Delete(args ...interface{}) error
}
