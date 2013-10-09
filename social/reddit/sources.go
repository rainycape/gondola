package reddit

var (
	HotSource           = "hot"
	NewSource           = "new"
	TopSource           = "top"
	ControversialSource = "controversial"
)

var (
	SortNew = &Parameter{
		sources: []string{NewSource},
		values:  map[string]string{"sort": "new"},
	}

	SortRising = &Parameter{
		sources: []string{NewSource},
		values:  map[string]string{"sort": "rising"},
	}

	TimeToday = &Parameter{
		sources: []string{TopSource, ControversialSource},
		values:  map[string]string{"t": "today"},
	}

	TimeHour = &Parameter{
		sources: []string{TopSource, ControversialSource},
		values:  map[string]string{"t": "hour"},
	}

	TimeWeek = &Parameter{
		sources: []string{TopSource, ControversialSource},
		values:  map[string]string{"t": "week"},
	}

	TimeMonth = &Parameter{
		sources: []string{TopSource, ControversialSource},
		values:  map[string]string{"t": "month"},
	}

	TimeYear = &Parameter{
		sources: []string{TopSource, ControversialSource},
		values:  map[string]string{"t": "year"},
	}

	TimeAll = &Parameter{
		sources: []string{TopSource, ControversialSource},
		values:  map[string]string{"t": "all"},
	}
)

type Parameter struct {
	sources []string
	values  map[string]string
}

func (p *Parameter) isValid(source string) bool {
	for _, s := range p.sources {
		if source == s {
			return true
		}
	}
	return false
}

func SourceIsValid(source string) bool {
	return source == HotSource || source == NewSource ||
		source == TopSource || source == ControversialSource
}
