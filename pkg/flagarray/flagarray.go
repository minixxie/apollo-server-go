package flagarray

type FlagArray []string

func (i *FlagArray) String() string {
	return "FlagArray"
}

func (i *FlagArray) Set(value string) error {
	*i = append(*i, value)
	return nil
}
