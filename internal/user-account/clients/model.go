package clients

type Client struct {
	Name       string `db:"name"`
	ClientUUID string `db:"uuid"`
	OwnerName  string `db:"username"`
}
