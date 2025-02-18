package providers

type AppServiceProvider struct{}

func NewAppServiceProvider() *AppServiceProvider {
	return &AppServiceProvider{}
}

func (s *AppServiceProvider) Register() {
	//
}
