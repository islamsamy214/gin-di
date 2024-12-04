package services

type MyService struct{}

func NewService() *MyService {
	return &MyService{}
}

func (s *MyService) GetHello() string {
	return "Hello"
}
