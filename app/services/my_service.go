package services

type MyService struct{}

func NewMyService() *MyService {
	return &MyService{}
}

func (s *MyService) GetHello() string {
	return "Hello"
}
