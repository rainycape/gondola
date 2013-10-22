package table

func Key(ctx string, singular string, plural string) string {
	return ctx + singular + plural
}
