package providers

type ConsoleServiceProvider struct{}

func NewConsoleServiceProvider() *ConsoleServiceProvider {
	return &ConsoleServiceProvider{}
}

func (r *ConsoleServiceProvider) Register() {
	//
}

func (r *ConsoleServiceProvider) Boot() {
	//
}
