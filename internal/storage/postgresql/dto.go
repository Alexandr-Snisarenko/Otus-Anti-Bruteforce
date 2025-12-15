package postgresql

type subnetRow struct {
	cidr     string `db:"cidr"`
	listType string `db:"list_type"`
}
