package federation

type Execute struct {
	testdataBaseDir    string
	testdataInvalidDir string
}

func NewExecute(testdataBaseDir, testdataInvalidDir string) Execute {
	return Execute{
		testdataBaseDir:    testdataBaseDir,
		testdataInvalidDir: testdataInvalidDir,
	}
}

func (e Execute) Run() error {
	return nil
}
