package postgresdb

type subnetRow struct {
	CIDR     string `db:"cidr"`
	ListType string `db:"list_type"`
}
