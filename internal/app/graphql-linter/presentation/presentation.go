package presentation

type Presenter interface {
	Run()
}

type CLI struct{}

func NewCLI() (CLI, error) {
	return CLI{}, nil
}

func (c CLI) Run() error {
	return nil
}
