package postgres

type insertResult int64

func (i insertResult) LastInsertId() (int64, error) {
	return int64(i), nil
}

func (i insertResult) RowsAffected() (int64, error) {
	return 1, nil
}
